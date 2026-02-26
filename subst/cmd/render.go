package cmd

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/kubelize/subst/internal/utils"
	"github.com/kubelize/subst/pkg/config"
	"github.com/kubelize/subst/pkg/subst"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

func newRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render structure with substitutions",
		Long: heredoc.Doc(`
			Run 'subst discover' to return directories that contain plugin compatible files. Mainly used for automatic plugin discovery by ArgoCD`),
		Example: `# Render the local manifests
subst render 
# Render in a different directory
subst render ../examples/02-overlays/clusters/cluster-01`,
		RunE: render,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	return cmd
}

func addRenderFlags(flags *flag.FlagSet) {
	flags.StringSlice("ejson-key", []string{}, heredoc.Doc(`
			Specify EJSON Private key used for decryption.
			May be specified multiple times or separate values with commas`))
	flags.Bool("skip-decrypt", false, heredoc.Doc(`
			Skip decryption`))
	flags.String("env-regex", "^ARGOCD_ENV_.*$", heredoc.Doc(`
	        Only expose environment variables that match the given regex`))
	flags.String("output", "yaml", heredoc.Doc(`
	        Output format. One of: yaml, json`))
	flags.String("kustomize-build-options", "", heredoc.Doc(`
	        Additional build options for kustomize. Example: --load-restrictor LoadRestrictionsNone`))

}

func render(cmd *cobra.Command, args []string) error {
	start := time.Now() // Start time measurement

	dir, err := rootDirectory(args)
	if err != nil {
		return err
	}

	configuration, err := config.LoadConfiguration(cfgFile, cmd, dir)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}

	// Use the new simplified Subst
	m, err := subst.NewSubst(*configuration)
	if err != nil {
		return err
	}

	if m != nil {
		err = m.Build()
		if err != nil {
			return err
		}
		if m.Manifests != nil {
			for _, f := range m.Manifests {
				if configuration.Output == "json" {
					// Convert bytes to map for JSON output
					var data map[interface{}]interface{}
					if err := utils.UnmarshalJSONorYAMLToInterface(f, &data); err != nil {
						log.Error().Msgf("failed to unmarshal for JSON: %s", err)
						continue
					}
					err = utils.PrintJSON(data)
					if err != nil {
						log.Error().Msgf("failed to print JSON: %s", err)
					}
				} else {
					err = utils.PrintYAMLBytes(f)
					if err != nil {
						log.Error().Msgf("failed to print YAML: %s", err)
					}
				}
			}
		}
	}
	elapsed := time.Since(start) // Calculate elapsed time
	log.Debug().Msgf("Build time for rendering: %s", elapsed)

	return nil
}
