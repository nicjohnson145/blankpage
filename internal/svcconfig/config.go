package svcconfig

import (
	"crypto/rand"
	"strings"

	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/spf13/viper"
)

//go:generate go-enum -f $GOFILE -marshal -names -flag

/*
ENUM(
memory
postgres
)
*/
type StorageKind string

const (
	// Port is the port the service will listen on
	Port = "port"
	// LogLevel controls the verbosity of logging
	LogLevel = "log.level"
	// LogFormat controls the format of log messages
	LogFormat = "log.format"

	// PayloadInterceptorRequestMethods are the methods to log request payloads for
	PayloadInterceptorRequestMethods = "interceptors.payload.request_methods"
	// PayloadInterceptorResponseMethods are the methods to log response payloads for
	PayloadInterceptorResponseMethods = "interceptors.payload.response_methods"
	// PayloadInterceptorPretty indicates if the logging should be pretty
	PayloadInterceptorPretty = "interceptors.payload.pretty"

	// StorageType controls what kind of storage used for books/metadata/etc
	StorageType = "storage.type"

	// The Postgres* constants control the paramters used to connect to the database when the storage type is postgres
	PostgresDatabaseUser     = "postgres.database_user"
	PostgresDatabasePassword = "postgres.database_password"
	PostgresDatabaseHost     = "postgres.database_host"
	PostgresDatabasePort     = "postgres.database_port"
	PostgresDatabaseName     = "postgres.database_name"
	PostgresDatabaseSSL      = "postgres.database_ssl"

	// PurgeEnabled controls if the purge endpoints of this service are enabled. Should only ever be enabled in a
	// testing context
	PurgeEnabled = "purge_enabled"

	// InitialAdminEmail is the email to seed the service with upon first startup. If not given, the default will
	// be used
	InitialAdminEmail = "initial_admin.email"
	// InitialAdminPassword is the password to seed the service with upon first startup. If not given, one will be
	// generated
	InitialAdminPassword = "initial_admin.password"
)

var (
	DefaultPort      = "8080"
	DefaultLogFormat = logging.LogFormatJson.String()
	DefaultLogLevel  = logging.LogLevelInfo.String()

	DefaultStorageType = StorageKindMemory.String()

	DefaultPostgresDatabasePort = 5432
	DefaultPostgresDatabaseSSL = "disable"

	DefaultPurgeEnabled = false

	DefaultInitialAdminEmail    = "admin@example.com"
	DefaultInitialAdminPassword = rand.Text()
)

func InitConfig() {
	viper.SetDefault(Port, DefaultPort)

	viper.SetDefault(LogFormat, DefaultLogFormat)
	viper.SetDefault(LogLevel, DefaultLogLevel)

	viper.SetDefault(StorageType, DefaultStorageType)

	viper.SetDefault(PostgresDatabasePort, DefaultPostgresDatabasePort)
	viper.SetDefault(PostgresDatabaseSSL, DefaultPostgresDatabaseSSL)

	viper.SetDefault(PurgeEnabled, DefaultPurgeEnabled)

	viper.SetDefault(InitialAdminEmail, DefaultInitialAdminEmail)
	viper.SetDefault(InitialAdminPassword, DefaultInitialAdminPassword)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}
