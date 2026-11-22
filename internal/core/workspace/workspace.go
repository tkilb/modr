package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Name           string                       `yaml:"name"`
	DisplayName    string                       `yaml:"display_name"`
	Aliases        []string                     `yaml:"aliases,omitempty"`
	Driver         string                       `yaml:"driver"`
	AllowSteamDeck *bool                        `yaml:"allow_steam_deck,omitempty"`
	SteamDeck      *bool                        `yaml:"steam_deck,omitempty"`
	Framework      *FrameworkSpec               `yaml:"framework,omitempty"`
	Paths          map[string]map[string]string `yaml:"paths"`
}

// SupportsSteamDeck reports whether the game is permitted to run on or sync to Steam Deck.
// Any game can omit Steam Deck support generically by:
//  1. Setting `allow_steam_deck: false` or `steam_deck: false` in its game.yaml
//  2. Omitting `steam-deck` / `steamdeck` from its declared paths map
func (s *GameSpec) SupportsSteamDeck() bool {
	if s == nil {
		return true
	}
	if s.AllowSteamDeck != nil {
		return *s.AllowSteamDeck
	}
	if s.SteamDeck != nil {
		return *s.SteamDeck
	}
	// If machine paths are explicitly declared but Steam Deck is omitted, treat as unsupported
	if len(s.Paths) > 0 {
		_, hasDeck := s.Paths["steam-deck"]
		_, hasDeckAlias := s.Paths["steamdeck"]
		if !hasDeck && !hasDeckAlias {
			return false
		}
	}
	return true
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

// SupportsSteamDeck reports whether the workspace is permitted to run on or sync to Steam Deck
func (w *Workspace) SupportsSteamDeck() bool {
	if w == nil {
		return true
	}
	if w.Spec != nil {
		return w.Spec.SupportsSteamDeck()
	}
	return true
}

// normalizeName strips hyphens, underscores, and lowercases a game identifier for flexible matching
func normalizeName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// MatchesGame reports whether a name or alias matches this specification
func (s *GameSpec) MatchesGame(query string) bool {
	if s == nil {
		return false
	}
	qNorm := normalizeName(query)
	if normalizeName(s.Name) == qNorm {
		return true
	}
	if normalizeName(s.DisplayName) == qNorm {
		return true
	}
	for _, a := range s.Aliases {
		if normalizeName(a) == qNorm {
			return true
		}
	}
	return false
}

// FindWorkspace resolves a game workspace within repoRoot/games/<gameName>
// It resolves exact directory matches first, then matches against game.yaml spec name and aliases.
func FindWorkspace(repoRoot, gameName string) (*Workspace, error) {
	if gameName == "" {
		return nil, fmt.Errorf("game name cannot be empty")
	}

	gameDir := filepath.Join(repoRoot, "games", gameName)
	fi, err := os.Stat(gameDir)
	if err == nil && fi.IsDir() {
		return loadWorkspaceFromDir(gameDir, gameName)
	}

	// If direct directory does not match, inspect all games/* directories for spec name or aliases
	allGames, err := ListWorkspaces(repoRoot)
	if err == nil {
		qNorm := normalizeName(gameName)
		for _, dirName := range allGames {
			candidateDir := filepath.Join(repoRoot, "games", dirName)
			if normalizeName(dirName) == qNorm {
				return loadWorkspaceFromDir(candidateDir, dirName)
			}
			specPath := filepath.Join(candidateDir, "game.yaml")
			if data, err := os.ReadFile(specPath); err == nil {
				var spec GameSpec
				if err := yaml.Unmarshal(data, &spec); err == nil {
					if spec.MatchesGame(gameName) {
						return loadWorkspaceFromDir(candidateDir, dirName)
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("game workspace not found for %q in %s/games", gameName, repoRoot)
}

func loadWorkspaceFromDir(gameDir, gameName string) (*Workspace, error) {
	ws := &Workspace{
		GameName:     gameName,
		RootDir:      gameDir,
		ManifestPath: filepath.Join(gameDir, "manifests", "mods.yaml"),
		LockfilePath: filepath.Join(gameDir, "manifests", "mods.lock.yaml"),
		VendorDir:    filepath.Join(gameDir, "mods", "vendor"),
		ForkedDir:    filepath.Join(gameDir, "mods", "forked"),
		CustomDir:    filepath.Join(gameDir, "mods", "custom"),
	}

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

// ResolveGameIdentifier resolves any game alias or name to its directory name and canonical spec name
func ResolveGameIdentifier(repoRoot, query string) (string, bool) {
	ws, err := FindWorkspace(repoRoot, query)
	if err != nil || ws == nil {
		return "", false
	}
	return filepath.Base(ws.RootDir), true
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
