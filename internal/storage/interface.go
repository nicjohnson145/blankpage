package storage

import (
	"context"
	"errors"
	"fmt"

	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/nicjohnson145/blankpage/internal/svcconfig"
	"github.com/spf13/viper"
)

var (
	ErrUnknownShelfError = errors.New("unknown shelf")
	ErrUnknownBookError  = errors.New("unknown book")
	ErrNotFoundError     = errors.New("not found")
)

type Storer interface {
	Purge(ctx context.Context) error
	AddBook(ctx context.Context, req *pbv1.AddBookRequest) error
	ReadBook(ctx context.Context, id string) (*pbv1.Book, error)
	RemoveBook(ctx context.Context, req *pbv1.RemoveBookRequest) error
	ListMetadata(ctx context.Context, req *pbv1.ListBooksRequest) ([]*pbv1.Metadata, uint32, error)
	GetAllSeries(ctx context.Context) ([]string, error)
	GetBooksBySeries(ctx context.Context, series string) ([]*pbv1.Metadata, error)
	CreateShelf(ctx context.Context, shelf *pbv1.Shelf) error
	ListShelves(ctx context.Context) ([]*pbv1.Shelf, error)
	AddBooksToShelf(ctx context.Context, req *pbv1.AddBooksToShelfRequest) error
	RemoveBooksFromShelf(ctx context.Context, req *pbv1.RemoveBooksFromShelfRequest) error
	ListBooksForShelf(ctx context.Context, req *pbv1.ListBooksForShelfRequest) ([]*pbv1.Metadata, error)
}

func NewFromEnv() (Storer, func(), error) {
	kind, err := svcconfig.ParseStorageKind(viper.GetString(svcconfig.StorageType))
	if err != nil {
		return nil, func() {}, err
	}

	switch kind {
	case svcconfig.StorageKindMemory:
		return NewMemory(MemoryConfig{}), func() {}, nil
	default:
		return nil, func() {}, fmt.Errorf("unhandled storage kind of %v", kind)
	}
}
