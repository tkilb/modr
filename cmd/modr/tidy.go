package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/tidy"
)

func newTidyCmd(env *config.Environment) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "tidy",
		Short: "Prune unreferenced vendor directories, lock entries, and stale symlinks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, driver, err := resolveTargetWorkspace(env)
			if err != nil {
				return err
			}

			paths, err := driver.ResolvePaths(env, ws)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Printf("%s[DRY-RUN]%s Previewing cleanup for %s...\n", colorCyan, colorReset, ws.GameName)
			} else {
				fmt.Printf("Running tidy on %s...\n", ws.GameName)
			}

			report, err := tidy.RunTidyWorkspace(env, ws, paths.ModsDir, dryRun)
			if err != nil {
				return err
			}

			fmt.Println(strings.Repeat("=", 60))
			fmt.Printf("Tidy Results (%s):\n", ws.GameName)
			fmt.Printf("  Pruned lock entries: %d\n", len(report.PrunedLockEntries))
			fmt.Printf("  Pruned vendor dirs:  %d\n", len(report.PrunedVendorDirs))
			fmt.Printf("  Pruned stale links:  %d\n", len(report.PrunedSymlinks))
			if report.GoModTidied {
				fmt.Println("  Go module dependencies tidied.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview tidy operations without deleting files")
	return cmd
}
