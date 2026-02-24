package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/nicjohnson145/blankpage/internal/service"
	"github.com/nicjohnson145/blankpage/internal/storage"
	"github.com/nicjohnson145/blankpage/internal/svcconfig"
	"github.com/nicjohnson145/connecthelp/codec"
	intercepters "github.com/nicjohnson145/connecthelp/interceptors/server"
	"github.com/nicjohnson145/hlp"
	"github.com/nicjohnson145/hlp/set"
	"github.com/nicjohnson145/pauth"
	pauthv1beta1connect "github.com/nicjohnson145/pauth/gen/go/pauth/v1beta1/pauthv1beta1connect"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed dist
var uiDistFS embed.FS

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

	pstore, pstoreCleanup, err := newPauthStorage(logger)
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
		InitialAdminRoles: []string{
			service.RoleAdmin,
			service.RoleBookUploader,
			service.RoleBookUpdater,
			service.RoleShelfAdmin,
			service.RoleViewOnly,
		},
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

	// build our UI FS
	uiFS, err := fs.Sub(uiDistFS, "dist")
	if err != nil {
		logger.Err(err).Msg("error building subFS")
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

	// UI routing
	mux.Handle("/", http.FileServerFS(uiFS))

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

func newPauthStorage(logger zerolog.Logger) (pauth.Storer, func(), error) {
	kind, err := svcconfig.ParseStorageKind(viper.GetString(svcconfig.StorageType))
	if err != nil {
		return nil, nil, err
	}

	switch kind {
	case svcconfig.StorageKindMemory:
		return pauth.NewMemoryStore(pauth.MemoryStoreOpts{})
	case svcconfig.StorageKindPostgres:
		return pauth.NewPostgresStore(pauth.PostgresStoreOpts{
			Logger: hlp.Ptr(zerologr.New(&logger)),
			ConnectionOpts: &pauth.PostgresStoreConnectionOpts{
				User:     viper.GetString(svcconfig.PostgresDatabaseUser),
				Password: viper.GetString(svcconfig.PostgresDatabasePassword),
				Host:     viper.GetString(svcconfig.PostgresDatabaseHost),
				Port:     viper.GetInt(svcconfig.PostgresDatabasePort),
				DBName:   viper.GetString(svcconfig.PostgresDatabaseName),
				SSLMode:  viper.GetString(svcconfig.PostgresDatabaseSSL),
			},
		})
	default:
		return nil, func() {}, fmt.Errorf("unhandled storage kind %v", kind)
	}

}
