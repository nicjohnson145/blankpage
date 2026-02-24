package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	blankpagev1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	blankpagev1connect "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1/blankpagev1connect"
	pauthv1beta1 "github.com/nicjohnson145/pauth/gen/go/pauth/v1beta1"
	pauthv1beta1connect "github.com/nicjohnson145/pauth/gen/go/pauth/v1beta1/pauthv1beta1connect"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func Purge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "purge the service (DANGER!)",
		Long:  "Hit the /Purge endpoint for both auth & core objects, this endpoint may not be enabled depending on service configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withLogging(cmd, func(logger zerolog.Logger) error {
				return withBlankpageClient(func(blankpageClient blankpagev1connect.BlankPageServiceClient) error {
					return withPAuthClient(func(pauthClient pauthv1beta1connect.PAuthServiceClient) error {
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()

						logger.Info().Msg("purging core objects")
						if _, err := blankpageClient.Purge(ctx, connect.NewRequest(&blankpagev1.PurgeRequest{})); err != nil {
							return fmt.Errorf("error purging core objects: %w", err)
						}

						logger.Info().Msg("purging auth")
						if _, err := pauthClient.Purge(ctx, connect.NewRequest(&pauthv1beta1.PurgeRequest{})); err != nil {
							return fmt.Errorf("error purging core objects: %w", err)
						}

						return nil
					})
				})
			})
		},
	}

	return cmd
}
