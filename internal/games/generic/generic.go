package generic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/games"
	"rimworld-mods/internal/manifest"
)

func init() {
	games.RegisterDriver(&GenericDriver{})
}

// GenericDriver provides drop-in symlink management for games without special load order engines
type GenericDriver struct{}

func (g *GenericDriver) Name() string {
	return "generic"
}

func (g *GenericDriver) DisplayName() string {
	return "Generic Mod Driver"
}

func (g *GenericDriver) ResolvePaths(env *config.Environment, ws *workspace.Workspace) (*games.GamePaths, error) {
	home, _ := os.UserHomeDir()

	// Check environment variable override: e.g. MODR_STARVALOR_MODS_DIR or MODR_MODS_DIR
	envKey := fmt.Sprintf("MODR_%s_MODS_DIR", strings.ToUpper(ws.GameName))
	if val := os.Getenv(envKey); val != "" {
		return &games.GamePaths{ModsDir: expandPath(val, home)}, nil
	}

	if ws.Spec != nil && ws.Spec.Paths != nil {
		// Look up machine specific paths (e.g. "linux-box" -> "desktop", "steam-deck" -> "steamdeck")
		machineKey := env.Machine
		// Map aliases
		aliases := []string{machineKey}
		if machineKey == "linux-box" {
			aliases = append(aliases, "desktop")
		} else if machineKey == "linux-book" {
			aliases = append(aliases, "laptop", "desktop")
		} else if machineKey == "steam-deck" {
			aliases = append(aliases, "steamdeck")
		}

		for _, alias := range aliases {
			if machinePaths, ok := ws.Spec.Paths[alias]; ok {
				if mDir, ok := machinePaths["mods_dir"]; ok && mDir != "" {
					cfgDir := machinePaths["config_dir"]
					return &games.GamePaths{
						ModsDir:   expandPath(mDir, home),
						ConfigDir: expandPath(cfgDir, home),
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no mods directory path configured for game %s on machine %s", ws.GameName, env.Machine)
}

func (g *GenericDriver) PreApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) error {
	return nil
}

func (g *GenericDriver) PostApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) (*games.ApplyResult, error) {
	return &games.ApplyResult{}, nil
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
