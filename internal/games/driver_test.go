package games_test

import (
	"testing"

	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/games"
	_ "rimworld-mods/internal/games/generic"
	_ "rimworld-mods/internal/games/hd2"
	_ "rimworld-mods/internal/games/rimworld"
)

func TestGetDriver(t *testing.T) {
	rw, err := games.GetDriver("rimworld")
	if err != nil {
		t.Fatalf("failed to get rimworld driver: %v", err)
	}
	if rw.Name() != "rimworld" {
		t.Errorf("expected name 'rimworld', got %s", rw.Name())
	}

	gen, err := games.GetDriver("generic")
	if err != nil {
		t.Fatalf("failed to get generic driver: %v", err)
	}
	if gen.Name() != "generic" {
		t.Errorf("expected name 'generic', got %s", gen.Name())
	}

	hd, err := games.GetDriver("hd2")
	if err != nil {
		t.Fatalf("failed to get hd2 driver: %v", err)
	}
	if hd.Name() != "hd2" {
		t.Errorf("expected name 'hd2', got %s", hd.Name())
	}

	h2, err := games.GetDriver("helldivers2")
	if err != nil {
		t.Fatalf("failed to get helldivers2 driver: %v", err)
	}
	if h2.Name() != "helldivers2" {
		t.Errorf("expected name 'helldivers2', got %s", h2.Name())
	}

	// Unknown driver should fall back to generic
	fallback, err := games.GetDriver("some_unknown_game")
	if err != nil {
		t.Fatalf("expected fallback to generic driver: %v", err)
	}
	if fallback.Name() != "generic" {
		t.Errorf("expected fallback to 'generic', got %s", fallback.Name())
	}
}

func TestGenericDriver_ResolvePaths(t *testing.T) {
	gen, err := games.GetDriver("generic")
	if err != nil {
		t.Fatal(err)
	}

	ws := &workspace.Workspace{
		GameName: "starvalor",
		Spec: &workspace.GameSpec{
			Name: "starvalor",
			Paths: map[string]map[string]string{
				"desktop": {
					"mods_dir": "/tmp/starvalor/BepInEx/plugins",
				},
			},
		},
	}

	env := &config.Environment{
		Machine: "linux-box",
	}

	paths, err := gen.ResolvePaths(env, ws)
	if err != nil {
		t.Fatalf("unexpected error resolving paths: %v", err)
	}
	if paths.ModsDir != "/tmp/starvalor/BepInEx/plugins" {
		t.Errorf("expected /tmp/starvalor/BepInEx/plugins, got %s", paths.ModsDir)
	}
}
