package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"go.uber.org/automaxprocs/maxprocs"
)

var (
	cfgFile string
	v       string
	p       int
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subst",
		Short: "Kustomize with substitution",
		Long: heredoc.Doc(`
			Create Kustomize builds with strong substitution capabilities`),
		SilenceUsage: true,
	}

	//Here is where we define the PreRun func, using the verbose flag value
	//We use the standard output for logs.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := setUpLogs(v); err != nil {
			return err
		}
		if err := setUpMaxProcs(p); err != nil {
			return err
		}
		return nil
	}

	//Default value is the warn level
	cmd.PersistentFlags().StringVarP(&v, "verbosity", "v", zerolog.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic")

	//Default value is inferred from cgroups or system
	cmd.PersistentFlags().IntVarP(&p, "maxprocs", "p", 0, "Overwrite GOMAXPROCS for the command to use (default: 0 which means respect cgroup or system)")

	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newRenderCmd())

	cmd.DisableAutoGenTag = true

	return cmd
}

// Execute runs the application
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// setUpLogs set the log output ans the log level
func setUpLogs(level string) error {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(lvl)
	return nil
}

// setUpMaxProcs set the max procs
func setUpMaxProcs(procs int) error {
	if procs > 0 {
		os.Setenv("GOMAXPROCS", strconv.Itoa(procs))
	}
	_, err := maxprocs.Set(maxprocs.Logger(log.Debug().Msgf))
	if err != nil {
		return err
	}
	return nil
}

func addCommonFlags(flags *flag.FlagSet) {
	flags.StringVar(&cfgFile, "config", "", "Config file")
	flags.Bool("debug", false, heredoc.Doc(`
			Print CLI calls of external tools to stdout (caution: setting this may
			expose sensitive data)`))
}

func rootDirectory(args []string) (directory string, err error) {
	directory = "."
	if len(args) > 0 {
		directory = args[0]
	}
	rootAbs, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("failed resolving root directory: %w", err)
	} else {
		directory = rootAbs
	}

	return directory, nil
}
