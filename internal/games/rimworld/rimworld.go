package rimworld

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/games"
	"rimworld-mods/internal/manifest"
	"rimworld-mods/internal/order"
)

func init() {
	games.RegisterDriver(&RimWorldDriver{})
}

// RimWorldDriver provides topological mod sorting and ModsConfig.xml generation
type RimWorldDriver struct{}

func (r *RimWorldDriver) Name() string {
	return "rimworld"
}

func (r *RimWorldDriver) DisplayName() string {
	return "RimWorld"
}

func (r *RimWorldDriver) ResolvePaths(env *config.Environment, ws *workspace.Workspace) (*games.GamePaths, error) {
	modsDir := env.RimWorldModsDir
	configDir := env.RimWorldConfigDir

	if modsDir == "" || configDir == "" {
		home, _ := os.UserHomeDir()
		// Fallback to workspace spec if set
		if ws.Spec != nil && ws.Spec.Paths != nil {
			machineKey := env.Machine
			aliases := []string{machineKey}
			if machineKey == "linux-box" {
				aliases = append(aliases, "desktop")
			} else if machineKey == "linux-book" {
				aliases = append(aliases, "laptop", "desktop")
			} else if machineKey == "steam-deck" {
				aliases = append(aliases, "steamdeck")
			}

			for _, alias := range aliases {
				if mp, ok := ws.Spec.Paths[alias]; ok {
					if modsDir == "" {
						modsDir = expandPath(mp["mods_dir"], home)
					}
					if configDir == "" {
						configDir = expandPath(mp["config_dir"], home)
					}
				}
			}
		}
	}

	return &games.GamePaths{
		ModsDir:   modsDir,
		ConfigDir: configDir,
	}, nil
}

func (r *RimWorldDriver) PreApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) error {
	return nil
}

func (r *RimWorldDriver) PostApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) (*games.ApplyResult, error) {
	paths, err := r.ResolvePaths(env, ws)
	if err != nil {
		return nil, err
	}

	sorter := order.NewSorter()
	vendorDir := ws.VendorDir
	customDir := ws.CustomDir
	forkedDir := ws.ForkedDir

	// Add declared vendor mods
	for modID, entry := range mf.Mods {
		if !entry.IsEnabled() || entry.IsCustom() || entry.IsForked() {
			continue
		}
		modDirPath := filepath.Join(vendorDir, modID)
		var meta *order.ModMetaData
		if fi, err := os.Stat(modDirPath); err == nil && fi.IsDir() {
			aboutPath, err := order.FindAboutXML(modDirPath)
			if err == nil {
				if parsed, err := order.ParseAboutXML(aboutPath); err == nil {
					meta = parsed
				}
			}
		}
		node := order.BuildModNode(modID, entry, meta, modDirPath, false)
		sorter.AddNode(node)
	}

	// Auto-discover custom mods
	if fi, err := os.Stat(customDir); err == nil && fi.IsDir() {
		entries, _ := os.ReadDir(customDir)
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			customID := e.Name()
			if entry, ok := mf.Mods[customID]; ok && !entry.IsEnabled() {
				continue
			}
			cDir := filepath.Join(customDir, customID)
			var meta *order.ModMetaData
			if aboutPath, err := order.FindAboutXML(cDir); err == nil {
				if parsed, err := order.ParseAboutXML(aboutPath); err == nil {
					meta = parsed
				}
			}
			var entry manifest.ModEntry
			if existing, ok := mf.Mods[customID]; ok {
				entry = existing
			}
			node := order.BuildModNode(customID, entry, meta, cDir, true)
			sorter.AddNode(node)
		}
	}

	// Auto-discover forked mods
	if fi, err := os.Stat(forkedDir); err == nil && fi.IsDir() {
		entries, _ := os.ReadDir(forkedDir)
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			forkedID := e.Name()
			if entry, ok := mf.Mods[forkedID]; ok && !entry.IsEnabled() {
				continue
			}
			fDir := filepath.Join(forkedDir, forkedID)
			var meta *order.ModMetaData
			if aboutPath, err := order.FindAboutXML(fDir); err == nil {
				if parsed, err := order.ParseAboutXML(aboutPath); err == nil {
					meta = parsed
				}
			}
			var entry manifest.ModEntry
			if existing, ok := mf.Mods[forkedID]; ok {
				entry = existing
			} else {
				entry.Source = "forked"
			}
			node := order.BuildModNode(forkedID, entry, meta, fDir, false)
			sorter.AddNode(node)
		}
	}

	// Filter available DLCs
	var activeDLCs []string
	for _, dlcPkg := range order.CanonicalDLCs {
		if mf.IsDLCEnabled(dlcPkg) {
			activeDLCs = append(activeDLCs, dlcPkg)
		}
	}

	res, err := sorter.Sort(activeDLCs)
	if err != nil {
		return nil, fmt.Errorf("mod load order resolution failed: %w", err)
	}

	configPath := filepath.Join(paths.ConfigDir, "ModsConfig.xml")
	existingVersion := order.ReadExistingVersion(configPath)
	if existingVersion == "" {
		existingVersion = "1.5.4297 rev513"
	}

	var activePkgs []string
	for _, node := range res.OrderedMods {
		activePkgs = append(activePkgs, node.NormalizedPackageID())
	}

	xmlBytes, err := order.GenerateModsConfigXML(activePkgs, activeDLCs, existingVersion)
	if err != nil {
		return nil, fmt.Errorf("failed generating ModsConfig.xml: %w", err)
	}

	_, err = order.DeployModsConfig(configPath, xmlBytes, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed deploying ModsConfig.xml: %w", err)
	}

	return &games.ApplyResult{
		ConfigPath: configPath,
		ActiveMods: activePkgs,
		Warnings:   res.Warnings,
	}, nil
}

func expandPath(path, home string) string {
	if path == "" {
		return ""
	}
	path = os.ExpandEnv(path)
	if home != "" && (path == "~" || strings.HasPrefix(path, "~/")) {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}
