package games

import (
	"fmt"
	"sync"

	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/manifest"
)

// GamePaths holds resolved filesystem locations for a game on the current machine
type GamePaths struct {
	GameDir   string
	ModsDir   string
	ConfigDir string
}

// ApplyResult contains post-apply metadata such as configured files and active items
type ApplyResult struct {
	ConfigPath string
	ActiveMods []string
	Warnings   []string
}

// GameDriver defines the behavior for game-specific mod management and configuration
type GameDriver interface {
	Name() string
	DisplayName() string
	ResolvePaths(env *config.Environment, ws *workspace.Workspace) (*GamePaths, error)
	PreApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) error
	PostApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) (*ApplyResult, error)
}

var (
	registryMu sync.RWMutex
	drivers    = make(map[string]GameDriver)
)

// RegisterDriver registers a driver implementation
func RegisterDriver(driver GameDriver) {
	registryMu.Lock()
	defer registryMu.Unlock()
	drivers[driver.Name()] = driver
}

// GetDriver retrieves a driver by name, falling back to generic driver if available
func GetDriver(name string) (GameDriver, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if d, ok := drivers[name]; ok {
		return d, nil
	}
	if name == "" || name == "generic" {
		if d, ok := drivers["generic"]; ok {
			return d, nil
		}
	}
	// Fall back to generic driver if registered
	if d, ok := drivers["generic"]; ok {
		return d, nil
	}

	return nil, fmt.Errorf("no driver registered for %q", name)
}
