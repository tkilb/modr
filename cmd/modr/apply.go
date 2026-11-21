package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/fetcher"
	"rimworld-mods/internal/linker"
)

// ANSI colors
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

func newApplyCmd(env *config.Environment) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Reconcile game installation to declared manifest state",
		Long: `Reconciles your target game installation against manifests/mods.yaml:
  1. Fetches any declared mods missing from cache
  2. Deploys symlinks into the game's mod directory
  3. Executes game-specific driver configuration (e.g. topological load order for RimWorld)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(env, dryRun)
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview actions without modifying files")
	return cmd
}

func runApply(env *config.Environment, dryRun bool) error {
	ws, driver, err := resolveTargetWorkspace(env)
	if err != nil {
		return err
	}

	paths, err := driver.ResolvePaths(env, ws)
	if err != nil {
		return fmt.Errorf("failed resolving game paths: %w", err)
	}

	displayName := ws.GameName
	if ws.Spec != nil && ws.Spec.DisplayName != "" {
		displayName = ws.Spec.DisplayName
	}

	if dryRun {
		fmt.Printf("%s[DRY-RUN]%s Previewing reconciliation for %s%s%s...\n", colorCyan, colorReset, colorBold, displayName, colorReset)
	} else {
		fmt.Printf("Reconciling %s%s%s (driver: %s)...\n", colorBold, displayName, colorReset, driver.Name())
	}
	fmt.Printf("Target Mods Dir: %s%s%s\n\n", colorCyan, paths.ModsDir, colorReset)

	mf, err := ws.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to read manifest %s: %w", ws.ManifestPath, err)
	}

	// Pre-apply hook
	if err := driver.PreApply(env, ws, mf, dryRun); err != nil {
		return fmt.Errorf("pre-apply check failed: %w", err)
	}

	// Step 1: Cache verification & fetching
	fmt.Printf("%s==> Step 1: Checking mod cache...%s\n", colorBold, colorReset)
	requireAboutXML := (driver.Name() == "rimworld")
	missing := fetcher.FindMissingModsInDir(ws.VendorDir, mf, requireAboutXML)
	if len(missing) > 0 {
		fmt.Printf("Found %d missing mod(s): %s\n", len(missing), strings.Join(missing, ", "))
		for _, m := range missing {
			if dryRun {
				fmt.Printf("  %s[DRY-RUN]%s Would fetch missing mod: %s\n", colorCyan, colorReset, m)
			} else {
				fmt.Printf("  Fetching missing mod: %s...\n", m)
				if err := fetcher.FetchModToDir(env, ws.VendorDir, m, mf.Mods[m], false); err != nil {
					return fmt.Errorf("failed fetching %s: %w", m, err)
				}
			}
		}
	} else {
		fmt.Printf("  %sAll declared mods are present in cache.%s\n", colorGreen, colorReset)
	}

	// Step 2: Symlink deployment
	fmt.Printf("\n%s==> Step 2: Reconciling mod symlinks...%s\n", colorBold, colorReset)
	l := linker.NewWorkspaceLinker(env, mf, paths.ModsDir, ws.VendorDir, ws.CustomDir, ws.ForkedDir)
	l.RequireAboutXML = (driver.Name() == "rimworld")
	linkResults, err := l.LinkAll(dryRun)
	if err != nil {
		return fmt.Errorf("symlink reconciliation failed: %w", err)
	}

	for _, res := range linkResults {
		switch res.Status {
		case linker.StatusCreated:
			fmt.Printf("  %s+ Linked:%s %s\n", colorGreen, colorReset, res.ModID)
		case linker.StatusUpdated:
			fmt.Printf("  %s~ Updated:%s %s\n", colorYellow, colorReset, res.ModID)
		case linker.StatusUpToDate:
			fmt.Printf("  %s= Up-to-date:%s %s\n", colorGray, colorReset, res.ModID)
		case linker.StatusUnlinked:
			fmt.Printf("  %s- Removed (unmanaged):%s %s\n", colorYellow, colorReset, res.ModID)
		case linker.StatusSkipped:
			fmt.Printf("  %s! Skipped:%s %s (%s)\n", colorYellow, colorReset, res.ModID, res.Message)
		case linker.StatusError:
			fmt.Printf("  %s✖ Error:%s %s (%s)\n", colorRed, colorReset, res.ModID, res.Message)
		}
	}

	// Step 3: Game-specific post-apply configuration
	postRes, err := driver.PostApply(env, ws, mf, dryRun)
	if err != nil {
		return fmt.Errorf("post-apply configuration failed: %w", err)
	}

	if postRes != nil {
		for _, warn := range postRes.Warnings {
			fmt.Printf("  %s[WARN]%s %s\n", colorYellow, colorReset, warn)
		}
		if postRes.ConfigPath != "" {
			if dryRun {
				fmt.Printf("  %s[DRY-RUN]%s Would write %d active mods to: %s\n",
					colorCyan, colorReset, len(postRes.ActiveMods), postRes.ConfigPath)
			} else {
				fmt.Printf("  %sSuccessfully deployed %d active mods to:%s %s\n",
					colorGreen, len(postRes.ActiveMods), colorReset, postRes.ConfigPath)
			}
		}
	}

	fmt.Printf("\n%s[OK] %s reconciliation complete.%s\n", colorGreen, displayName, colorReset)
	return nil
}
