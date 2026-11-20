package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/deck"
	"rimworld-mods/internal/manifest"
)

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func colorPad(text string, width int, color string) string {
	padded := pad(text, width)
	if color != "" {
		return color + padded + colorReset
	}
	return padded
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func newStatusCmd(env *config.Environment) *cobra.Command {
	var checkDeck bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display game mod status, integrity report, and symlink health",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, driver, err := resolveTargetWorkspace(env)
			if err != nil {
				return err
			}

			paths, err := driver.ResolvePaths(env, ws)
			if err != nil {
				return err
			}

			disp := ws.GameName
			if ws.Spec != nil && ws.Spec.DisplayName != "" {
				disp = ws.Spec.DisplayName
			}

			fmt.Printf("%sModr - %s Status & Integrity Report%s\n", colorBold, disp, colorReset)
			fmt.Printf("Machine Target:        %s%s%s\n", colorCyan, env.Machine, colorReset)
			fmt.Printf("Game Driver:           %s\n", driver.Name())
			fmt.Printf("Target Mods Path:      %s%s%s\n", colorCyan, paths.ModsDir, colorReset)
			fmt.Printf("Workspace Root:        %s\n", ws.RootDir)
			if paths.ConfigDir != "" {
				fmt.Printf("Config Dir:            %s\n", paths.ConfigDir)
			}
			fmt.Println(strings.Repeat("=", 80))

			mf, err := ws.LoadManifest()
			if err != nil {
				return fmt.Errorf("failed to read manifest %s: %w", ws.ManifestPath, err)
			}

			lf, _ := ws.LoadLockfile()
			if lf == nil {
				lf = &manifest.Lockfile{Mods: make(map[string]manifest.LockEntry)}
			}

			// 1. Manifest 3rd-party mods table
			fmt.Printf("\n%sDeclared Community Mods (%s):%s\n", colorBold, ws.ManifestPath, colorReset)
			fmt.Printf("%s %s %s %s %s %s\n",
				pad("MOD ID", 28),
				pad("ENABLED", 8),
				pad("SOURCE", 16),
				pad("DOWNLOADED", 12),
				pad("VERSION/TAG", 16),
				pad("LINK STATUS", 15),
			)
			fmt.Println(strings.Repeat("-", 101))

			var modIDs []string
			for id := range mf.Mods {
				modIDs = append(modIDs, id)
			}
			sort.Strings(modIDs)

			enabledCount := 0
			downloadedCount := 0
			linkedCount := 0

			for _, id := range modIDs {
				m := mf.Mods[id]
				source := m.Source
				if source == "" {
					source = "steam_workshop"
				}
				if m.IsCustom() || m.IsForked() {
					continue
				}

				enabled := m.IsEnabled()
				if enabled {
					enabledCount++
				}

				// Downloaded check
				cachedPath := filepath.Join(ws.VendorDir, id)
				isDownloaded := false
				if fi, err := os.Stat(cachedPath); err == nil && fi.IsDir() {
					if entries, err := os.ReadDir(cachedPath); err == nil && len(entries) > 0 {
						isDownloaded = true
						downloadedCount++
					}
				}

				dlText := "Missing"
				dlColor := colorRed
				if isDownloaded {
					dlText = "Yes"
					dlColor = colorGreen
				}

				// Version / Lock info
				versionInfo := "unlocked"
				vColor := colorYellow
				if lockEntry, ok := lf.Mods[id]; ok {
					if lockEntry.WorkshopTimeUpdated > 0 {
						versionInfo = fmt.Sprintf("rev:%d", lockEntry.WorkshopTimeUpdated)
						vColor = ""
					} else if lockEntry.GitCommit != "" {
						c := lockEntry.GitCommit
						if len(c) > 7 {
							c = c[:7]
						}
						versionInfo = fmt.Sprintf("git:%s", c)
						vColor = ""
					} else if lockEntry.GitRef != "" {
						versionInfo = lockEntry.GitRef
						vColor = ""
					}
				}

				// Link status
				linkPath := filepath.Join(paths.ModsDir, id)
				linkText, linkColor := checkLinkStatus(linkPath, cachedPath)
				if linkText == "Linked" {
					linkedCount++
				}

				enText := "no"
				enColor := colorGray
				if enabled {
					enText = "yes"
					enColor = colorGreen
				}

				fmt.Printf("%s %s %s %s %s %s\n",
					pad(id, 28),
					colorPad(enText, 8, enColor),
					pad(source, 16),
					colorPad(dlText, 12, dlColor),
					colorPad(versionInfo, 16, vColor),
					colorPad(linkText, 15, linkColor),
				)
			}

			// 2. Official Expansions (if RimWorld)
			if driver.Name() == "rimworld" {
				fmt.Printf("\n%sOfficial Expansions (DLCs):%s\n", colorBold, colorReset)
				fmt.Printf("%s %s %s\n", pad("EXPANSION", 20), pad("ENABLED", 8), pad("PACKAGE ID", 30))
				fmt.Println(strings.Repeat("-", 62))

				canonicalDLCs := []string{"royalty", "ideology", "biotech", "anomaly", "odyssey"}
				for _, dlc := range canonicalDLCs {
					en := mf.IsDLCEnabled(dlc)
					enText := "no"
					enColor := colorGray
					if en {
						enText = "yes"
						enColor = colorGreen
					}
					label := capitalize(dlc)
					pkgID := "ludeon.rimworld." + dlc
					fmt.Printf("%s %s %s\n", pad(label, 20), colorPad(enText, 8, enColor), pad(pkgID, 30))
				}
			}

			// 3. Custom Authored Mods
			fmt.Printf("\n%sCustom Authoring Mods (%s):%s\n", colorBold, ws.CustomDir, colorReset)
			fmt.Printf("%s %s %s %s\n",
				pad("MOD ID", 24),
				pad("ENABLED", 8),
				pad("WORKSPACE", 16),
				pad("LINK STATUS", 15),
			)
			fmt.Println(strings.Repeat("-", 69))

			customModsMap := make(map[string]bool)
			var customMods []string
			if entries, err := os.ReadDir(ws.CustomDir); err == nil {
				for _, e := range entries {
					if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
						customMods = append(customMods, e.Name())
						customModsMap[e.Name()] = true
					}
				}
			}
			sort.Strings(customMods)

			for _, cID := range customMods {
				cPath := filepath.Join(ws.CustomDir, cID)
				enabled := true
				if m, ok := mf.Mods[cID]; ok {
					enabled = m.IsEnabled()
				}
				enText := "no"
				enColor := colorGray
				if enabled {
					enText = "yes"
					enColor = colorGreen
				}

				linkPath := filepath.Join(paths.ModsDir, cID)
				linkText, linkColor := checkLinkStatus(linkPath, cPath)

				fmt.Printf("%s %s %s %s\n",
					pad(cID, 24),
					colorPad(enText, 8, enColor),
					pad("local (custom)", 16),
					colorPad(linkText, 15, linkColor),
				)
			}

			// 4. Forked Community Mods
			fmt.Printf("\n%sForked Community Mods (%s):%s\n", colorBold, ws.ForkedDir, colorReset)
			fmt.Printf("%s %s %s %s %s\n",
				pad("MOD ID", 24),
				pad("ENABLED", 8),
				pad("UPSTREAM", 28),
				pad("COMMIT/REF", 16),
				pad("LINK STATUS", 15),
			)
			fmt.Println(strings.Repeat("-", 97))

			forkedModsMap := make(map[string]bool)
			var forkedMods []string
			if entries, err := os.ReadDir(ws.ForkedDir); err == nil {
				for _, e := range entries {
					if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
						forkedMods = append(forkedMods, e.Name())
						forkedModsMap[e.Name()] = true
					}
				}
			}
			sort.Strings(forkedMods)

			for _, fID := range forkedMods {
				fPath := filepath.Join(ws.ForkedDir, fID)
				enabled := true
				upstream := "unknown"
				commit := "-"
				if m, ok := mf.Mods[fID]; ok {
					enabled = m.IsEnabled()
					if m.UpstreamRepo != "" {
						upstream = m.UpstreamRepo
					}
					if m.UpstreamCommit != "" {
						commit = m.UpstreamCommit
						if len(commit) > 7 {
							commit = commit[:7]
						}
					} else if m.UpstreamBranch != "" {
						commit = m.UpstreamBranch
					}
				}
				enText := "no"
				enColor := colorGray
				if enabled {
					enText = "yes"
					enColor = colorGreen
				}

				linkPath := filepath.Join(paths.ModsDir, fID)
				linkText, linkColor := checkLinkStatus(linkPath, fPath)

				fmt.Printf("%s %s %s %s %s\n",
					pad(fID, 24),
					colorPad(enText, 8, enColor),
					pad(upstream, 28),
					pad(commit, 16),
					colorPad(linkText, 15, linkColor),
				)
			}

			// 5. Summary / Deck
			fmt.Printf("\n%sSummary:%s %d declared (%d enabled), %d cached, %d custom, %d forked, %d active links\n",
				colorBold, colorReset, len(mf.Mods), enabledCount, downloadedCount, len(customMods), len(forkedMods), linkedCount)

			if checkDeck {
				fmt.Printf("\n%sSteam Deck Reachability (%s):%s\n", colorBold, env.DeckHost, colorReset)
				if err := deck.CheckConnection(env); err != nil {
					fmt.Printf("  %s✖ Deck offline or unreachable: %v%s\n", colorRed, err, colorReset)
				} else {
					fmt.Printf("  %s✔ Deck online and reachable via SSH%s\n", colorGreen, colorReset)
				}
			}

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().BoolVarP(&checkDeck, "check-deck", "d", false, "Test SSH connectivity to Steam Deck")
	return cmd
}

func checkLinkStatus(linkPath, expectedTarget string) (string, string) {
	lfi, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "Unlinked", colorGray
		}
		return "Error", colorRed
	}

	if (lfi.Mode() & os.ModeSymlink) != 0 {
		target, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			target, _ = os.Readlink(linkPath)
		}
		expectedClean, _ := filepath.EvalSymlinks(expectedTarget)
		if expectedClean == "" {
			expectedClean = filepath.Clean(expectedTarget)
		}

		if target == expectedClean {
			return "Linked", colorGreen
		}
		return "Drifted Target", colorYellow
	}

	if lfi.IsDir() {
		return "Direct Dir", colorYellow
	}

	return "Unknown", colorYellow
}
