package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"rimworld-mods/internal/core/workspace"
)

func TestSupportsSteamDeck(t *testing.T) {
	tests := []struct {
		name     string
		spec     *workspace.GameSpec
		expected bool
	}{
		{
			name:     "nil spec defaults to true",
			spec:     nil,
			expected: true,
		},
		{
			name: "normal game with no explicit setting",
			spec: &workspace.GameSpec{
				Name: "starvalor",
			},
			expected: true,
		},
		{
			name: "generic game with explicit allow_steam_deck false",
			spec: func() *workspace.GameSpec {
				f := false
				return &workspace.GameSpec{
					Name:           "customgame",
					AllowSteamDeck: &f,
				}
			}(),
			expected: false,
		},
		{
			name: "generic game with explicit steam_deck false alias",
			spec: func() *workspace.GameSpec {
				f := false
				return &workspace.GameSpec{
					Name:      "spacegame",
					SteamDeck: &f,
				}
			}(),
			expected: false,
		},
		{
			name: "generic game with declared desktop/laptop paths but omitting steam-deck",
			spec: &workspace.GameSpec{
				Name: "deskonlygame",
				Paths: map[string]map[string]string{
					"desktop": {"mods_dir": "/tmp/mods"},
					"laptop":  {"mods_dir": "/tmp/mods"},
				},
			},
			expected: false,
		},
		{
			name: "generic game with declared steam-deck paths and no disable flag",
			spec: &workspace.GameSpec{
				Name: "deckgame",
				Paths: map[string]map[string]string{
					"desktop":    {"mods_dir": "/tmp/mods"},
					"steam-deck": {"mods_dir": "/tmp/deckmods"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.SupportsSteamDeck()
			if got != tt.expected {
				t.Errorf("SupportsSteamDeck() = %v; want %v", got, tt.expected)
			}
		})
	}
}

func TestFindWorkspace_Aliases(t *testing.T) {
	// Create temporary workspace structure
	tempDir, err := os.MkdirTemp("", "modr_ws_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	gameDir := filepath.Join(tempDir, "games", "helldivers-2")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	gameYAML := `name: helldivers2
display_name: "Helldivers 2"
aliases:
  - helldivers-2
  - hd2
driver: hd2
allow_steam_deck: false
`
	if err := os.WriteFile(filepath.Join(gameDir, "game.yaml"), []byte(gameYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Resolve by directory name
	ws, err := workspace.FindWorkspace(tempDir, "helldivers-2")
	if err != nil {
		t.Fatalf("failed finding by directory name: %v", err)
	}
	if ws.Spec.Name != "helldivers2" {
		t.Errorf("expected spec name 'helldivers2', got %q", ws.Spec.Name)
	}
	if ws.SupportsSteamDeck() {
		t.Errorf("expected helldivers-2 to not support Steam Deck")
	}

	// 2. Resolve by spec name (helldivers2)
	ws, err = workspace.FindWorkspace(tempDir, "helldivers2")
	if err != nil {
		t.Fatalf("failed finding by spec name: %v", err)
	}
	if ws.Spec.DisplayName != "Helldivers 2" {
		t.Errorf("expected display name 'Helldivers 2', got %q", ws.Spec.DisplayName)
	}

	// 3. Resolve by alias (hd2)
	ws, err = workspace.FindWorkspace(tempDir, "hd2")
	if err != nil {
		t.Fatalf("failed finding by alias hd2: %v", err)
	}
	if ws.Spec.Name != "helldivers2" {
		t.Errorf("expected spec name 'helldivers2', got %q", ws.Spec.Name)
	}

	// 4. Test ResolveGameIdentifier
	dirName, ok := workspace.ResolveGameIdentifier(tempDir, "hd2")
	if !ok || dirName != "helldivers-2" {
		t.Errorf("expected ResolveGameIdentifier('hd2') = 'helldivers-2', true; got %q, %v", dirName, ok)
	}

	dirName, ok = workspace.ResolveGameIdentifier(tempDir, "helldivers2")
	if !ok || dirName != "helldivers-2" {
		t.Errorf("expected ResolveGameIdentifier('helldivers2') = 'helldivers-2', true; got %q, %v", dirName, ok)
	}
}
