package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	blankpagev1connect "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1/blankpagev1connect"
	pauthv1beta1 "github.com/nicjohnson145/pauth/gen/go/pauth/v1beta1"
	pauthv1beta1connect "github.com/nicjohnson145/pauth/gen/go/pauth/v1beta1/pauthv1beta1connect"
	"github.com/nicjohnson145/blankpage/internal/cliconfig"
	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/nicjohnson145/hlp"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	baseHttpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
)

// This function is not suitable for long-lived (i.e as long as a token duration) connections
func getTokenInterceptor(httpClient *http.Client) connect.UnaryInterceptorFunc {
	var token string

	hClient := httpClient
	if hClient == nil {
		hClient = &http.Client{
			Timeout: 3 * time.Second,
		}
	}

	client := pauthv1beta1connect.NewPAuthServiceClient(hClient, viper.GetString(cliconfig.FlagServerUrl))

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			var reqToken string
			// if we've got a token from config, use that every time. Otherwise if we've already made a login request,
			// use that one. Finally, get a new token if we must
			if val := viper.GetString(cliconfig.AuthenticationAccessKey); val != "" {
				reqToken = val
			} else if token != "" {
				reqToken = token
			} else {
				resp, err := client.Login(ctx, connect.NewRequest(&pauthv1beta1.LoginRequest{
					Email:    hlp.Ptr(viper.GetString(cliconfig.AuthenticationEmail)),
					Password: viper.GetString(cliconfig.AuthenticationPassword),
				}))
				if err != nil {
					return nil, fmt.Errorf("error getting access token: %w", err)
				}
				reqToken = resp.Msg.AccessKey
				token = resp.Msg.AccessKey
			}

			req.Header().Set("Authorization", reqToken)
			return next(ctx, req)
		}
	}
}

func withLogging(cmd *cobra.Command, workFunc func(logger zerolog.Logger) error) error {
	return workFunc(logging.Init(&logging.LoggingConfig{
		// We've safely validated this already
		Level:  hlp.Must(logging.ParseLogLevel(hlp.Must(cmd.Flags().GetString(cliconfig.FlagLogLevel)))),
		Format: logging.LogFormatHuman,
	}))
}

func withBlankpageClient(workFunc func(client blankpagev1connect.BlankPageServiceClient) error) error {
	return workFunc(blankpagev1connect.NewBlankPageServiceClient(
		baseHttpClient,
		viper.GetString(cliconfig.FlagServerUrl),
		connect.WithInterceptors(
			getTokenInterceptor(baseHttpClient),
		),
	))
}

func withPAuthClient(workFunc func(client pauthv1beta1connect.PAuthServiceClient) error) error {
	return workFunc(pauthv1beta1connect.NewPAuthServiceClient(
		baseHttpClient,
		viper.GetString(cliconfig.FlagServerUrl),
		connect.WithInterceptors(
			getTokenInterceptor(baseHttpClient),
		),
	))
}
