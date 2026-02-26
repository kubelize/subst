package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc"
	"github.com/kubelize/subst/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover if plugin is applicable to the given directory",
		Long: heredoc.Doc(`
			Run 'subst discover' to return directories that contain plugin compatible files. Mainly used for automatic plugin discovery by ArgoCD`),
		RunE: discover,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	return cmd
}

func discover(cmd *cobra.Command, args []string) error {
	dir, err := rootDirectory(args)
	if err != nil {
		return err
	}

	_, err = config.LoadConfiguration(cfgFile, cmd, dir)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}

	if hasSubstFiles(dir) {
		log.Debug().Msg("Found subst.yaml files - subst plugin applicable")
		fmt.Println("subst")
		return nil
	}

	log.Debug().Msg("No subst.yaml files found")
	return fmt.Errorf("no subst.yaml files found in directory %s", dir)
}

// hasSubstFiles checks if directory or subdirectories contain subst.yaml
func hasSubstFiles(dir string) bool {
	found := false
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on error
		}
		if !info.IsDir() && info.Name() == "subst.yaml" {
			found = true
			return filepath.SkipDir // Stop walking once found
		}
		return nil
	})
	return found
}
