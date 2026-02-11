package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/huh"
	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	pbv1connect "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1/blankpagev1connect"
	"github.com/nicjohnson145/blankpage/internal/cliconfig"
	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/nicjohnson145/hlp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AddBook() *cobra.Command {
	const (
		flagTitle     = "title"
		flagAuthor    = "author"
		flagSeries    = "series"
		flagSeriesNum = "series-num"
	)
	var (
		overrideTitle        string
		overrideAuthor       string
		overrideSeries       string
		overrideSeriesNumber float32
	)

	cmd := &cobra.Command{
		Use:   "add-book <BOOK>",
		Args:  cobra.ExactArgs(1),
		Short: "Add a book to a blankpage server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize our logger
			logger := logging.Init(&logging.LoggingConfig{
				// We've safely validated this already
				Level:  hlp.Must(logging.ParseLogLevel(hlp.Must(cmd.Flags().GetString(cliconfig.FlagLogLevel)))),
				Format: logging.LogFormatHuman,
			})

			// Read the supplied file
			content, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			httpClient := &http.Client{
				Timeout: 3 * time.Second,
			}

			// Build a client
			client := pbv1connect.NewBlankPageServiceClient(
				httpClient,
				viper.GetString(cliconfig.FlagServerUrl),
				connect.WithInterceptors(
					getTokenInterceptor(httpClient),
				),
			)

			// Get our metadata
			logger.Debug().Msg("executing metadata parse request")
			resp, err := client.ExtractMetadata(cmd.Context(), connect.NewRequest(&pbv1.ExtractMetadataRequest{
				Content: content,
			}))
			if err != nil {
				return fmt.Errorf("error executing metadata request: %w", err)
			}

			var resolvedTitle string
			var resolvedAuthor string
			var resolvedSeries string
			var resolvedSeriesNum string

			if resp.Msg.Metadata.Title != "" {
				resolvedTitle = resp.Msg.Metadata.Title
			}
			if resp.Msg.Metadata.Author != nil {
				resolvedAuthor = *resp.Msg.Metadata.Author
			}
			if resp.Msg.Metadata.Series != nil {
				resolvedSeries = *resp.Msg.Metadata.Series
			}
			if resp.Msg.Metadata.SeriesNumber != nil {
				resolvedSeriesNum = fmt.Sprint(*resp.Msg.Metadata.SeriesNumber)
			}

			// if we're allowed to be interactive, let the user correct anything before we send it
			if !viper.GetBool(cliconfig.FlagNoInteractive) {
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Title").
							Value(&resolvedTitle),
						huh.NewInput().
							Title("Author").
							Value(&resolvedAuthor),
						huh.NewInput().
							Title("Series").
							Value(&resolvedSeries),
						huh.NewInput().
							Title("Series Number").
							Validate(func(s string) error {
								if s == "" {
									return nil
								}

								_, err := strconv.ParseFloat(s, 32)
								if err != nil {
									return fmt.Errorf("series number must be a number")
								}

								return nil
							}).
							Value(&resolvedSeriesNum),
					),
				)
				if err := form.RunWithContext(cmd.Context()); err != nil {
					return fmt.Errorf("error executing user form: %w", err)
				}
			}

			// Add our book with the collected metadata
			metadata := &pbv1.Metadata{}
			if resolvedTitle != "" {
				metadata.Title = resolvedTitle
			}
			if resolvedAuthor != "" {
				metadata.Author = hlp.Ptr(resolvedAuthor)
			}
			if resolvedSeries != "" {
				metadata.Series = hlp.Ptr(resolvedSeries)
			}
			if resolvedSeriesNum != "" {
				metadata.SeriesNumber = hlp.Ptr(float32(hlp.Must(strconv.ParseFloat(resolvedSeriesNum, 32))))
			}

			logger.Info().Msg("uploading book")
			_, err = client.AddBook(cmd.Context(), connect.NewRequest(&pbv1.AddBookRequest{
				Book: &pbv1.Book{
					Metadata: metadata,
					Content:  content,
				},
			}))
			if err != nil {
				return fmt.Errorf("error uploading book: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&overrideTitle, flagTitle, "", "Force the title of the book")
	cmd.Flags().StringVar(&overrideAuthor, flagAuthor, "", "Force the author of the book")
	cmd.Flags().StringVar(&overrideSeries, flagSeries, "", "Force the series of the book")
	cmd.Flags().Float32Var(&overrideSeriesNumber, flagSeriesNum, 0, "Force the series number of the book")

	return cmd
}
