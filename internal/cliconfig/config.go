package cliconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FlagNoInteractive       = "no-interactive"
	FlagLogLevel            = "log-level"
	FlagServerUrl           = "server-url"
	AuthenticationEmail     = "authentication.email"
	AuthenticationPassword  = "authentication.password"
	AuthenticationAccessKey = "authentication.access-key"
)

var (
	DefaultFlagNoInteractive = false
	DefaultFlagLogLevel      = logging.LogLevelInfo.String()
)

func InitConfig(cmd *cobra.Command) error {

	viper.SetConfigName("cli")
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("error getting user config dir: %w", err)
	}
	viper.AddConfigPath(filepath.Join(configDir, "blankpage"))
	viper.SetConfigType("yaml")

	// Read in disk level config
	if err := viper.ReadInConfig(); err != nil {
		// If its not there then just eat it
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %s", err)
		}
	}

	// Set environment variable bits
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.SetEnvPrefix("blankpage")

	// Bind to flags
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("error binding flags: %w", err)
	}

	return nil
}
