package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/fetcher"
	"rimworld-mods/internal/games/generic"
)

func newSetupCmd(env *config.Environment) *cobra.Command {
	var (
		dryRun bool
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Bootstrap and install required modding frameworks (e.g. BepInEx 5 for Star Valor)",
		Long: `Downloads and configures the required mod loader / framework for the active game:
  - For Star Valor: downloads BepInEx 5 (x64) and installs it into the game directory
  - Creates necessary directory structures (e.g. BepInEx/plugins)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, driver, err := resolveTargetWorkspace(env)
			if err != nil {
				return err
			}

			if ws.Spec == nil || ws.Spec.Framework == nil {
				fmt.Printf("No framework bootstrap required for %s.\n", ws.GameName)
				return nil
			}

			fw := ws.Spec.Framework
			paths, err := driver.ResolvePaths(env, ws)
			if err != nil {
				return fmt.Errorf("failed resolving paths: %w", err)
			}

			if paths.GameDir == "" {
				return fmt.Errorf("cannot install framework: game_dir is not resolved for %s", ws.GameName)
			}

			disp := ws.GameName
			if ws.Spec != nil && ws.Spec.DisplayName != "" {
				disp = ws.Spec.DisplayName
			}

			fmt.Printf("%sModr - Framework Setup for %s%s\n", colorBold, disp, colorReset)
			fmt.Printf("Framework:    %s (version %s)\n", fw.Type, fw.Version)
			fmt.Printf("Target Path:  %s\n\n", paths.GameDir)

			installed, detail := generic.CheckFrameworkStatus(ws, paths)
			if installed && !force {
				fmt.Printf("%s✔ %s%s\n", colorGreen, detail, colorReset)
				fmt.Printf("Use '--force' to re-download and re-install.\n")
				return nil
			}

			if dryRun {
				fmt.Printf("%s[DRY-RUN]%s Would download %s from:\n  %s\n", colorCyan, colorReset, fw.Type, fw.URL)
				fmt.Printf("%s[DRY-RUN]%s Would extract into: %s\n", colorCyan, colorReset, paths.GameDir)
				return nil
			}

			// Verify game installation exists
			if fi, err := os.Stat(paths.GameDir); err != nil || !fi.IsDir() {
				return fmt.Errorf("game install directory does not exist: %s\nPlease install %s on this machine before running 'modr setup'", paths.GameDir, disp)
			}

			fmt.Printf("==> Downloading %s (%s)...\n", fw.Type, fw.URL)
			tmpZip, err := os.CreateTemp("", "modr-framework-*.zip")
			if err != nil {
				return err
			}
			defer func() {
				_ = tmpZip.Close()
				_ = os.Remove(tmpZip.Name())
			}()

			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Get(fw.URL)
			if err != nil {
				return fmt.Errorf("failed to download framework archive: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed downloading framework, HTTP status: %d", resp.StatusCode)
			}

			if _, err := io.Copy(tmpZip, resp.Body); err != nil {
				return fmt.Errorf("failed to write framework archive: %w", err)
			}
			_ = tmpZip.Close()

			fmt.Printf("==> Extracting into %s...\n", paths.GameDir)
			if err := fetcher.UnzipToDir(tmpZip.Name(), paths.GameDir); err != nil {
				return fmt.Errorf("failed extracting framework archive: %w", err)
			}

			// Ensure BepInEx/plugins exists
			pluginsDir := filepath.Join(paths.GameDir, "BepInEx", "plugins")
			_ = os.MkdirAll(pluginsDir, 0755)

			fmt.Printf("\n%s[OK] %s %s successfully installed into:%s\n  %s\n\n",
				colorGreen, fw.Type, fw.Version, colorReset, paths.GameDir)

			if fw.Type == "bepinex5" || fw.Type == "bepinex" {
				fmt.Printf("%s[IMPORTANT] Linux / Steam Deck Launch Option:%s\n", colorYellow, colorReset)
				fmt.Printf("To enable BepInEx under Proton, set your Steam Launch Options for Star Valor to:\n\n")
				fmt.Printf("  %sWINEDLLOVERRIDES=\"winhttp=n,b\" %%command%%%s\n\n", colorBold, colorReset)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview framework installation without downloading")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-download and re-extraction even if already present")
	return cmd
}
