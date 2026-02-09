package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/go-logr/zerologr"
	"github.com/justinas/alice"
	pbv1connect "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1/blankpagev1connect"
	pauthv1beta1connect "github.com/nicjohnson145/blankpage/gen/go/pauth/v1beta1/pauthv1beta1connect"
	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/nicjohnson145/blankpage/internal/pauth"
	pstorage "github.com/nicjohnson145/blankpage/internal/pauth/storage"
	"github.com/nicjohnson145/blankpage/internal/service"
	"github.com/nicjohnson145/blankpage/internal/storage"
	"github.com/nicjohnson145/blankpage/internal/svcconfig"
	"github.com/nicjohnson145/connecthelp/codec"
	intercepters "github.com/nicjohnson145/connecthelp/interceptors/server"
	"github.com/nicjohnson145/hlp/set"
	"github.com/spf13/viper"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	svcconfig.InitConfig()

	// Disable adding v-level to log output, zerolog has levels
	zerologr.VerbosityFieldName = ""

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logging.Init(&logging.LoggingConfig{
		Level:  logging.LogLevel(viper.GetString(svcconfig.LogLevel)),
		Format: logging.LogFormat(viper.GetString(svcconfig.LogFormat)),
	})
	rootLogr := zerologr.New(&logger)

	reflector := grpcreflect.NewStaticReflector(
		pbv1connect.BlankPageServiceName,
	)

	storer, storerCleanup, err := storage.NewFromEnv(logger)
	defer storerCleanup()
	if err != nil {
		logger.Err(err).Msg("error creating storer")
		return err
	}

	pstore, pstoreCleanup, err := pstorage.NewFromEnv(logger)
	defer pstoreCleanup()
	if err != nil {
		logger.Err(err).Msg("error creating pauth storage")
		return err
	}

	srv := service.NewService(service.ServiceConfig{
		Storer:         storer,
		PurgeEnabled:   viper.GetBool(svcconfig.PurgeEnabled),
		DefaultPerPage: 50,
	})

	adminEmail := viper.GetString(svcconfig.InitialAdminEmail)
	adminPassword := viper.GetString(svcconfig.InitialAdminPassword)
	psrv := pauth.NewService(pauth.ServiceConfig{
		PurgeEnabled:         viper.GetBool(svcconfig.PurgeEnabled),
		Store:                pstore,
		InitialAdminEmail:    adminEmail,
		InitialAdminPassword: adminPassword,
		InitialAdminRoles:    []string{pauth.RoleAdministrator},
	})

	// Call our bootstrap function on startup, in case its the first one
	created, err := psrv.Bootstrap(ctx)
	if err != nil {
		logger.Err(err).Msg("error bootstrapping")
		return err
	}
	if created {
		logger.Info().Msgf("initial admin email: %v", adminEmail)
		logger.Info().Msgf("initial admin password: %v", adminPassword)
	}

	additionalPublicEndpoints := set.New(
		pbv1connect.BlankPageServicePurgeProcedure,
	)
	pauthInterceptor, err := pauth.NewConnectInterceptor(pauth.ConnectInterceptorConfig{
		Store: pstore,
		BypassFunc: func(route string) bool {
			return additionalPublicEndpoints.Contains(route)
		},
	})
	if err != nil {
		logger.Err(err).Msg("error creating auth connect interceptor")
		return err
	}

	mux := http.NewServeMux()

	interceptors := []connect.Interceptor{
		intercepters.NewContextLoggerInterceptor(intercepters.ContextLoggerInterceptorConfig{
			RootLogger:        rootLogr,
			NoAttachRequestID: true,
		}),
		intercepters.NewPanicInterceptor(intercepters.PanicInterceptorConfig{}),
		intercepters.NewMethodLoggingInterceptor(intercepters.MethodLoggingInterceptorConfig{}),
		intercepters.NewPayloadLoggingInterceptor(intercepters.PayloadLoggingInterceptorConfig{
			RequestMethods:  viper.GetString(svcconfig.PayloadInterceptorRequestMethods),
			ResponseMethods: viper.GetString(svcconfig.PayloadInterceptorResponseMethods),
			Pretty:          viper.GetBool(svcconfig.PayloadInterceptorPretty),
		}),
		intercepters.NewProtovalidateInterceptor(intercepters.ProtovalidateInterceptorConfig{}),
		pauthInterceptor,
	}

	// Core service routing
	mux.Handle(pbv1connect.NewBlankPageServiceHandler(
		srv,
		connect.WithInterceptors(interceptors...),
		connect.WithCodec(codec.NewProtoJSONCodec(codec.ProtoJSONCodecOpts{
			ProtoJsonOpts: protojson.MarshalOptions{
				UseProtoNames: true,
			},
		})),
	))

	// Auth routing
	mux.Handle(pauthv1beta1connect.NewPAuthServiceHandler(
		psrv,
		connect.WithInterceptors(interceptors...),
		connect.WithCodec(codec.NewProtoJSONCodec(codec.ProtoJSONCodecOpts{
			ProtoJsonOpts: protojson.MarshalOptions{
				UseProtoNames: true,
			},
		})),
	))

	// OPDS routing
	opdsInterceptors := alice.New(
		service.ContextLoggerMiddleware(rootLogr),
		service.MethodLoggingMiddleware,
		pauth.BasicAuthMiddleware(pstore),
	)
	mux.Handle(service.RouteRoot(), opdsInterceptors.Then(http.HandlerFunc(srv.OPDSRoot)))
	mux.Handle(service.RouteFilterPattern(), opdsInterceptors.Then(http.HandlerFunc(srv.OPDSFilter)))
	mux.Handle(service.RouteSingleSeriesPattern(), opdsInterceptors.Then(http.HandlerFunc(srv.OPDSSingleSeries)))
	mux.Handle(service.RouteDownloadPattern(), opdsInterceptors.Then(http.HandlerFunc(srv.OPDSDownload)))

	// Reflection routing
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	port := viper.GetString(svcconfig.Port)
	lis, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		logger.Err(err).Msg("error listening")
		return err
	}

	httpServer := http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 3 * time.Second,
	}

	// Setup signal handlers so we can gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-sigChan
		logger.Info().Msgf("got signal %v, attempting graceful shutdown", s)
		dieCtx, dieCancel := context.WithTimeout(ctx, 10*time.Second)
		defer dieCancel()
		_ = httpServer.Shutdown(dieCtx)
	}()

	logger.Info().Msgf("starting server on port %v", port)
	if err := httpServer.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Err(err).Msg("error serving")
		return err
	}

	return nil
}
