package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/updater"
)

func newUpdateCmd(env *config.Environment) *cobra.Command {
	var dryRun bool
	var targetMod string

	cmd := &cobra.Command{
		Use:   "update [mod-id]",
		Short: "Check upstream sources and update cached vendor mods",
		Long: `Queries upstream Steam Workshop or GitHub releases for new versions,
synchronizes manifests/mods.lock.yaml, and re-fetches changed mods into cache.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, _, err := resolveTargetWorkspace(env)
			if err != nil {
				return err
			}

			if len(args) > 0 {
				targetMod = args[0]
			}

			if dryRun {
				fmt.Printf("%s[DRY-RUN]%s Checking for updates (%s)...\n", colorCyan, colorReset, ws.GameName)
			} else {
				fmt.Printf("Updating declared mods for %s...\n", ws.GameName)
			}

			summary, err := updater.UpdateWorkspace(env, ws, targetMod, dryRun)
			if err != nil {
				return err
			}

			fmt.Println(strings.Repeat("=", 60))
			fmt.Printf("Update Summary (%s):\n", ws.GameName)
			fmt.Printf("  Updated:   %d\n", len(summary.Updated))
			fmt.Printf("  Unchanged: %d\n", len(summary.Unchanged))
			if len(summary.Errors) > 0 {
				fmt.Printf("  %sErrors:    %d%s\n", colorRed, len(summary.Errors), colorReset)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview updates without modifying files")
	return cmd
}
