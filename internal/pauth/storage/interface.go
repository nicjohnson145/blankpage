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
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	pbv1beta1 "github.com/nicjohnson145/blankpage/gen/go/pauth/v1beta1"
	"github.com/nicjohnson145/blankpage/internal/svcconfig"
	"github.com/nicjohnson145/hlp"
	hsqlx "github.com/nicjohnson145/hlp/sqlx"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

//go:embed postgres-migrations/*.sql
var postgresMigration embed.FS

var (
	ErrUserAlreadyExistsError        = errors.New("a user with this id already exists")
	ErrUnknownUserError              = errors.New("unknown user")
	ErrIncorrectPasswordError        = errors.New("password does not match")
	ErrNoPasswordConfiguredError     = errors.New("no password configured")
	ErrSessionUnknownOrInactiveError = errors.New("session unknwon or inactive")
)

type CreateUserRequest struct {
	User           *pbv1beta1.User
	HashedPassword *string
}

type SetUserPasswordRequest struct {
	UserId         string
	HashedPassword string
}

type CreateSessionRequest struct {
	Session     *Session
	ActiveUntil time.Time
}

type LoginInfo struct {
	User           *pbv1beta1.User
	StoredPassword string
}

type Storer interface {
	Purge(ctx context.Context) error

	CreateUser(ctx context.Context, req *CreateUserRequest) error
	ReadUser(ctx context.Context, userId string) (*pbv1beta1.User, error)
	ReadLoginInfo(ctx context.Context, userID *string, email *string) (*LoginInfo, error)
	ListUsers(ctx context.Context, req *pbv1beta1.ListUsersRequest) ([]*pbv1beta1.User, error)
	SetUserPassword(ctx context.Context, req *SetUserPasswordRequest) error
	UpdateUser(ctx context.Context, user *pbv1beta1.User) error
	DeleteUser(ctx context.Context, req *pbv1beta1.DeleteUserRequest) error
	ListUserRoles(ctx context.Context, userId string) ([]string, error)
	GrantUserRoles(ctx context.Context, req *pbv1beta1.GrantUserRoleRequest) error
	RevokeUserRoles(ctx context.Context, req *pbv1beta1.RevokeUserRoleRequest) error

	GetActiveSession(ctx context.Context, sessionKey string) (*Session, error)
	CreateSession(ctx context.Context, req *CreateSessionRequest) error
	RemoveSession(ctx context.Context, sessionKey string) error
}

func NewFromEnv(logger zerolog.Logger) (Storer, func(), error) {
	return newFromEnvWithNow(logger, nil)
}

// only here to be called directly from the storage integration tests
func newFromEnvWithNow(logger zerolog.Logger, nowFunc func() time.Time) (Storer, func(), error) {
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
		driver, err := postgres.WithInstance(db, &postgres.Config{MigrationsTable: "pauth_migrations"})
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
		return NewMemory(MemoryConfig{
			Now: nowFunc,
		}), func() {}, nil
	case svcconfig.StorageKindPostgres:
		return NewPostgres(PostgresConfig{
			DB:      db,
			NowFunc: nowFunc,
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
