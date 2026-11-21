package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/fetcher"
	"rimworld-mods/internal/importer"
	"rimworld-mods/internal/manifest"
)

func newImportCmd(env *config.Environment) *cobra.Command {
	var (
		slugOverride string
		dryRun       bool
		applyAfter   bool
		disabled     bool
	)

	cmd := &cobra.Command{
		Use:     "import <source>",
		Aliases: []string{"add"},
		Short:   "Import a new mod from a URL (Steam Workshop, GitHub, Git) or local archive (.zip, .dll)",
		Long: `Imports a mod into the game workspace from a Steam Workshop, GitHub, Git URL, or local archive:
  1. Detects source type (URL or local .zip/.dll file) and stages the mod
  2. Inspects metadata (About.xml or plugin DLL) to determine name and package ID
  3. Normalizes directory structure (e.g. unwrapping BepInEx/plugins)
  4. Derives a clean, collision-free kebab-case slug (or uses --slug)
  5. Moves mod into <game>/mods/vendor/<slug>
  6. Adds mod entry to <game>/manifests/mods.yaml while preserving comments
  7. Optionally applies and configures mods (--apply)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rawURL := args[0]

			ws, _, err := resolveTargetWorkspace(env)
			if err != nil {
				return err
			}

			src, err := importer.ParseSource(rawURL)
			if err != nil {
				return fmt.Errorf("failed to parse source: %w", err)
			}

			manifestPath := ws.ManifestPath
			mf, err := manifest.LoadManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("failed to read manifest %s: %w", manifestPath, err)
			}

			existingSlugs := make(map[string]bool)
			for s := range mf.Mods {
				existingSlugs[s] = true
			}

			if dryRun {
				fmt.Printf("%s[DRY-RUN]%s Previewing mod import into %s from %s (%s)...\n",
					colorCyan, colorReset, ws.GameName, src.SourceType, rawURL)
			} else {
				if src.SourceType == "archive" {
					fmt.Printf("%s==> Step 1: Staging mod for %s from archive (%s)...%s\n",
						colorBold, ws.GameName, rawURL, colorReset)
				} else {
					fmt.Printf("%s==> Step 1: Downloading mod for %s from %s (%s)...%s\n",
						colorBold, ws.GameName, src.SourceType, rawURL, colorReset)
				}
			}

			meta, stageDir, cleanup, err := importer.FetchAndStage(env, src, dryRun)
			if err != nil {
				return err
			}
			defer cleanup()

			// Step 2: Determine slug
			slug := slugOverride
			if slug != "" {
				slug = importer.ToKebabCase(slug)
			} else {
				slug = importer.DeriveSlug(meta.PackageID, existingSlugs)
			}

			fmt.Printf("\n%s==> Step 2: Mod metadata resolved:%s\n", colorBold, colorReset)
			fmt.Printf("  Slug:        %s%s%s\n", colorCyan, slug, colorReset)
			fmt.Printf("  Name:        %s\n", meta.Name)
			fmt.Printf("  Package ID:  %s\n", meta.PackageID)
			fmt.Printf("  Source:      %s\n", src.SourceType)
			if meta.Description != "" {
				descPreview := meta.Description
				if len(descPreview) > 100 {
					descPreview = descPreview[:100] + "..."
				}
				fmt.Printf("  Description: %s\n", descPreview)
			}

			if _, ok := mf.Mods[slug]; ok {
				fmt.Printf("  %s[WARN]%s Slug '%s' is already declared in %s (will update entry)\n",
					colorYellow, colorReset, slug, manifestPath)
			}

			// Step 3: Move to vendor directory
			vendorDest := filepath.Join(ws.VendorDir, slug)
			if dryRun {
				fmt.Printf("\n%s[DRY-RUN]%s Would move downloaded mod to: %s\n", colorCyan, colorReset, vendorDest)
				fmt.Printf("%s[DRY-RUN]%s Would register '%s' in %s\n", colorCyan, colorReset, slug, manifestPath)
			} else {
				fmt.Printf("\n%s==> Step 3: Placing mod into cache...%s\n", colorBold, colorReset)
				_ = os.RemoveAll(vendorDest)
				if err := os.MkdirAll(filepath.Dir(vendorDest), 0755); err != nil {
					return fmt.Errorf("failed to create vendor dir: %w", err)
				}
				if err := fetcher.CopyDir(stageDir, vendorDest); err != nil {
					return fmt.Errorf("failed to copy mod to %s: %w", vendorDest, err)
				}
				if src.SourceType == "steam_workshop" {
					_ = os.RemoveAll(filepath.Join(vendorDest, ".git"))
				}
				fmt.Printf("  Placed at %s\n", vendorDest)

				// Step 4: Write to manifests/mods.yaml
				fmt.Printf("\n%s==> Step 4: Updating %s...%s\n", colorBold, manifestPath, colorReset)
				enabledVal := !disabled
				newEntry := manifest.ModEntry{
					Name:        meta.Name,
					Source:      src.SourceType,
					WorkshopID:  src.WorkshopID,
					Repo:        src.Repo,
					Tag:         src.Tag,
					URL:         src.URL,
					PackageID:   meta.PackageID,
					Enabled:     &enabledVal,
					Description: meta.Description,
				}

				if err := importer.SaveManifestWithComments(manifestPath, slug, newEntry); err != nil {
					return fmt.Errorf("failed to update %s: %w", manifestPath, err)
				}
				fmt.Printf("  %sAdded '%s' to %s%s\n", colorGreen, slug, manifestPath, colorReset)
			}

			// Step 5: Optional Apply
			if applyAfter {
				fmt.Printf("\n%s==> Step 5: Applying changes...%s\n", colorBold, colorReset)
				if err := runApply(env, dryRun); err != nil {
					return fmt.Errorf("apply failed after import: %w", err)
				}
			} else {
				fmt.Printf("\n%s[OK] Mod '%s' imported successfully into %s.%s\n", colorGreen, slug, ws.GameName, colorReset)
				fmt.Printf("Run '%smodr apply%s' to deploy symlinks.\n", colorBold, colorReset)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&slugOverride, "slug", "", "Custom slug name for manifests/mods.yaml (default: auto-derived kebab-case)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview import without downloading or modifying manifest")
	cmd.Flags().BoolVar(&applyAfter, "apply", false, "Automatically reconcile and sort mods after import")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Import mod as disabled (enabled: false)")

	return cmd
}
