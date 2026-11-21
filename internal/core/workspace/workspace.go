package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"rimworld-mods/internal/manifest"
)

// FrameworkSpec defines a modding runtime/loader (e.g. BepInEx 5) required by the game
type FrameworkSpec struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
	URL     string `yaml:"url"`
}

// GameSpec holds configuration from game.yaml
type GameSpec struct {
	Name        string                       `yaml:"name"`
	DisplayName string                       `yaml:"display_name"`
	Driver      string                       `yaml:"driver"`
	Framework   *FrameworkSpec               `yaml:"framework,omitempty"`
	Paths       map[string]map[string]string `yaml:"paths"`
}

// Workspace represents a game's isolated mod management space in games/<game>
type Workspace struct {
	GameName     string
	RootDir      string
	ManifestPath string
	LockfilePath string
	VendorDir    string
	ForkedDir    string
	CustomDir    string
	Spec         *GameSpec
}

// FindWorkspace resolves a game workspace within repoRoot/games/<gameName>
func FindWorkspace(repoRoot, gameName string) (*Workspace, error) {
	if gameName == "" {
		return nil, fmt.Errorf("game name cannot be empty")
	}

	gameDir := filepath.Join(repoRoot, "games", gameName)
	fi, err := os.Stat(gameDir)
	if err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("game workspace not found at %s: %w", gameDir, err)
	}

	ws := &Workspace{
		GameName:     gameName,
		RootDir:      gameDir,
		ManifestPath: filepath.Join(gameDir, "manifests", "mods.yaml"),
		LockfilePath: filepath.Join(gameDir, "manifests", "mods.lock.yaml"),
		VendorDir:    filepath.Join(gameDir, "mods", "vendor"),
		ForkedDir:    filepath.Join(gameDir, "mods", "forked"),
		CustomDir:    filepath.Join(gameDir, "mods", "custom"),
	}

	// Read game.yaml if present
	specPath := filepath.Join(gameDir, "game.yaml")
	if data, err := os.ReadFile(specPath); err == nil {
		var spec GameSpec
		if err := yaml.Unmarshal(data, &spec); err == nil {
			ws.Spec = &spec
		}
	}

	if ws.Spec == nil {
		ws.Spec = &GameSpec{
			Name:   gameName,
			Driver: "generic",
		}
	}

	return ws, nil
}

// ListWorkspaces discovers all game workspaces in repoRoot/games/
func ListWorkspaces(repoRoot string) ([]string, error) {
	gamesDir := filepath.Join(repoRoot, "games")
	entries, err := os.ReadDir(gamesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() && !stringsHasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// LoadManifest loads the workspace's mods.yaml
func (w *Workspace) LoadManifest() (*manifest.Manifest, error) {
	return manifest.LoadManifest(w.ManifestPath)
}

// LoadLockfile loads the workspace's mods.lock.yaml
func (w *Workspace) LoadLockfile() (*manifest.Lockfile, error) {
	lf, err := manifest.LoadLockfile(w.LockfilePath)
	if err != nil {
		return &manifest.Lockfile{Mods: make(map[string]manifest.LockEntry)}, nil
	}
	return lf, nil
}
