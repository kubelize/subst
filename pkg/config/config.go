package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Configuration struct {
	EnvRegex              string   `mapstructure:"env-regex"`
	RootDirectory         string   `mapstructure:"root-dir"`
	EjsonKey              []string `mapstructure:"ejson-key"`
	SkipDecrypt           bool     `mapstructure:"skip-decrypt"`
	Output                string   `mapstructure:"output"`
	KustomizeBuildOptions string   `mapstructure:"kustomize-build-options"`
}

func LoadConfiguration(cfgFile string, cmd *cobra.Command, directory string) (*Configuration, error) {
	v := viper.New()

	cmd.Flags().VisitAll(func(flag *flag.Flag) {
		flagName := flag.Name
		if flagName != "config" && flagName != "help" {
			if err := v.BindPFlag(flagName, flag); err != nil {
				panic(fmt.Sprintf("failed binding flag %q: %v\n", flagName, err.Error()))
			}
		}
	})

	cfg := &Configuration{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed unmarshaling configuration: %w", err)
	}

	// Root Directory
	cfg.RootDirectory = directory

	// Set kustomize build options from environment if not set via flag
	if cfg.KustomizeBuildOptions == "" {
		cfg.KustomizeBuildOptions = os.Getenv("KUSTOMIZE_BUILD_OPTIONS")
	}

	log.Debug().Msgf("Configuration: %+v\n", cfg)
	return cfg, nil

}

func PrintConfiguration(cfg *Configuration) {
	fmt.Fprintln(os.Stderr, " Configuration")
	e := reflect.ValueOf(cfg).Elem()
	typeOfCfg := e.Type()

	for i := 0; i < e.NumField(); i++ {
		var pattern string
		switch e.Field(i).Kind() {
		case reflect.Bool:
			pattern = "%s: %t\n"
		default:
			pattern = "%s: %s\n"
		}
		fmt.Fprintf(os.Stderr, pattern, typeOfCfg.Field(i).Name, e.Field(i).Interface())
	}
}
