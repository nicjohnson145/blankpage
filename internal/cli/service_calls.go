package cli

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
)

func (c *CLI) ExtractMetadata(ctx context.Context, content []byte) (*pbv1.Metadata, error) {
	resp, err := c.client.ExtractMetadata(ctx, connect.NewRequest(&pbv1.ExtractMetadataRequest{
		Content: content,
	}))
	if err != nil {
		return nil, fmt.Errorf("error querying service: %w", err)
	}

	return resp.Msg.Metadata, nil
}
