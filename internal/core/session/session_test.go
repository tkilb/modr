package session

import (
	"os"
	"testing"
)

func TestSessionManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "modr-session-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{BaseDir: tmpDir}

	// Initial state: empty
	active, err := sm.GetActiveGame()
	if err != nil {
		t.Fatalf("unexpected error on empty session: %v", err)
	}
	if active != "" {
		t.Fatalf("expected empty active game, got %q", active)
	}

	// Set active game
	if err := sm.SetActiveGame("rimworld"); err != nil {
		t.Fatalf("failed to set active game: %v", err)
	}

	active, err = sm.GetActiveGame()
	if err != nil {
		t.Fatalf("failed to get active game: %v", err)
	}
	if active != "rimworld" {
		t.Fatalf("expected 'rimworld', got %q", active)
	}

	// Clear active game
	if err := sm.ClearActiveGame(); err != nil {
		t.Fatalf("failed to clear active game: %v", err)
	}

	active, err = sm.GetActiveGame()
	if err != nil {
		t.Fatalf("failed to get active game after clear: %v", err)
	}
	if active != "" {
		t.Fatalf("expected empty active game after clear, got %q", active)
	}
}

func TestSessionManager_EnvOverride(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "modr-session-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := &SessionManager{BaseDir: tmpDir}
	_ = sm.SetActiveGame("rimworld")

	t.Setenv("MODR_GAME", "starvalor")
	active, err := sm.GetActiveGame()
	if err != nil {
		t.Fatalf("failed to get active game: %v", err)
	}
	if active != "starvalor" {
		t.Fatalf("expected env override 'starvalor', got %q", active)
	}
}
