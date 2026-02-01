package cli

import (
	"net/http"
	"time"

	pbv1connect "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1/blankpagev1connect"
)

type CLIConfig struct {
	HttpClient    *http.Client
	ServerAddress string
}

func NewCLI(conf CLIConfig) *CLI {
	client := conf.HttpClient
	if client == nil {
		client = &http.Client{
			Timeout: 3 * time.Second,
		}
	}

	return &CLI{
		client: pbv1connect.NewBlankPageServiceClient(
			client,
			conf.ServerAddress,
		),
		config: conf,
	}
}

type CLI struct {
	client pbv1connect.BlankPageServiceClient
	config CLIConfig
}
