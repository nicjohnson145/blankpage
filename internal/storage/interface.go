package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/go-logr/zerologr"
	_ "github.com/lib/pq" // import postgres driver
	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/nicjohnson145/blankpage/internal/svcconfig"
	"github.com/nicjohnson145/hlp"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	hsqlx "github.com/nicjohnson145/hlp/sqlx"
)

//go:embed postgres-migrations/*.sql
var postgresMigration embed.FS

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

func NewFromEnv(logger zerolog.Logger) (Storer, func(), error) {
	kind, err := svcconfig.ParseStorageKind(viper.GetString(svcconfig.StorageType))
	if err != nil {
		return nil, func() {}, err
	}

	cleanup := func() {}

	// Do we need to connect to a DB? if so, do it
	var db *sql.DB
	switch kind {
	case svcconfig.StorageKindPostgres:
		connectionStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			viper.GetString(svcconfig.PostgresDatabaseUser),
			viper.GetString(svcconfig.PostgresDatabasePassword),
			viper.GetString(svcconfig.PostgresDatabaseHost),
			viper.GetInt(svcconfig.PostgresDatabasePort),
			viper.GetString(svcconfig.PostgresDatabaseName),
			viper.GetString(svcconfig.PostgresDatabaseSSL),
		)
		d, err := sql.Open("postgres", connectionStr)
		if err != nil {
			return nil, func() {}, fmt.Errorf("error opening database: %w", err)
		}
		db = d

		cleanup = func() {
			_ = db.Close()
		}
	}

	if db != nil {
		logger.Info().Msg("waiting for DB to show connectable")
		opts := hsqlx.DBWaitOpts{
			Timeout: hlp.Ptr(10 * time.Second),
			Logger:  hlp.Ptr(zerologr.New(&logger)),
		}
		if err := hsqlx.WaitForDBConnectable(db, opts); err != nil {
			return nil, cleanup, err
		}
	}

	var migrationFunc func() error
	switch kind {
	case svcconfig.StorageKindPostgres:
		driver, err := postgres.WithInstance(db, &postgres.Config{MigrationsTable: "blankpage_migrations"})
		if err != nil {
			return nil, cleanup, fmt.Errorf("error creating driver: %w", err)
		}
		subFS, err := fs.Sub(postgresMigration, "postgres-migrations")
		if err != nil {
			return nil, cleanup, fmt.Errorf("error creating sub fs: %w", err)
		}

		migrationFunc = goMigrateMigration(subFS, driver, "postgres")
	}

	if migrationFunc != nil {
		logger.Info().Msg("executing migrations")
		if err := migrationFunc(); err != nil {
			return nil, cleanup, fmt.Errorf("error migrating: %w", err)
		}
	}

	switch kind {
	case svcconfig.StorageKindMemory:
		return NewMemory(MemoryConfig{}), func() {}, nil
	case svcconfig.StorageKindPostgres:
		return NewPostgres(PostgresConfig{
			DB: db,
		}), cleanup, nil
	default:
		return nil, func() {}, fmt.Errorf("unhandled storage kind of %v", kind)
	}
}

func goMigrateMigration(sourceFS fs.FS, driver database.Driver, dbName string) func() error {
	return func() error {
		source, err := iofs.New(sourceFS, ".")
		if err != nil {
			return fmt.Errorf("error creating migration source: %w", err)
		}

		migrater, err := migrate.NewWithInstance("iofs", source, dbName, driver)
		if err != nil {
			return fmt.Errorf("error creating migration instance: %w", err)
		}

		if err := migrater.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("error executing migrations: %w", err)
		}

		return nil
	}
}
