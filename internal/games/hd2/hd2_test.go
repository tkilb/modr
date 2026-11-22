package hd2_test

import (
	"os"
	"path/filepath"
	"testing"

	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/games"
	"rimworld-mods/internal/games/hd2"
	"rimworld-mods/internal/linker"
	"rimworld-mods/internal/manifest"
)

func TestHD2Driver_SteamDeckRejection(t *testing.T) {
	driver := &hd2.HD2Driver{}
	env := &config.Environment{Machine: "steam-deck"}
	ws := &workspace.Workspace{GameName: "helldivers2"}

	// 1. ResolvePaths on Steam Deck must fail
	_, err := driver.ResolvePaths(env, ws)
	if err == nil {
		t.Fatal("expected error calling ResolvePaths on Steam Deck, got nil")
	}

	// 2. PreApply on Steam Deck must fail
	err = driver.PreApply(env, ws, nil, false)
	if err == nil {
		t.Fatal("expected error calling PreApply on Steam Deck, got nil")
	}

	// 3. DeployLinks on Steam Deck must fail
	_, err = driver.DeployLinks(env, ws, nil, &games.GamePaths{ModsDir: "/tmp"}, false)
	if err == nil {
		t.Fatal("expected error calling DeployLinks on Steam Deck, got nil")
	}
}

func TestHD2Driver_PatchStackingAndSafety(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "modr_hd2_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	wsRoot := filepath.Join(tempDir, "games", "helldivers-2")
	customModsDir := filepath.Join(wsRoot, "mods", "custom")
	dataDir := filepath.Join(tempDir, "game", "data")

	if err := os.MkdirAll(customModsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Place an 80GB-style base game file in data/ (MUST NOT BE DELETED!)
	baseGameArchive := filepath.Join(dataDir, "02582f3da1f8daf5.stream")
	if err := os.WriteFile(baseGameArchive, []byte("base-game-data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Place an existing manual (non-symlink) patch_0 in data/ (MUST NOT BE OVERWRITTEN!)
	existingPatch := filepath.Join(dataDir, "9ba626afa44a3aa3.patch_0")
	if err := os.WriteFile(existingPatch, []byte("existing-manual-patch"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create Mod 1 with patch_0 files for 9ba626afa44a3aa3
	mod1Dir := filepath.Join(customModsDir, "mod-alpha")
	_ = os.MkdirAll(mod1Dir, 0755)
	_ = os.WriteFile(filepath.Join(mod1Dir, "9ba626afa44a3aa3.patch_0"), []byte("mod1-patch"), 0644)
	_ = os.WriteFile(filepath.Join(mod1Dir, "9ba626afa44a3aa3.patch_0.stream"), []byte("mod1-stream"), 0644)

	// Create Mod 2 also targeting 9ba626afa44a3aa3 with patch_0 files (collision candidate!)
	mod2Dir := filepath.Join(customModsDir, "mod-beta")
	_ = os.MkdirAll(mod2Dir, 0755)
	_ = os.WriteFile(filepath.Join(mod2Dir, "9ba626afa44a3aa3.patch_0"), []byte("mod2-patch"), 0644)
	_ = os.WriteFile(filepath.Join(mod2Dir, "9ba626afa44a3aa3.patch_0.gpu_resources"), []byte("mod2-gpu"), 0644)

	ws := &workspace.Workspace{
		GameName:  "helldivers2",
		RootDir:   wsRoot,
		CustomDir: customModsDir,
		VendorDir: filepath.Join(wsRoot, "mods", "vendor"),
		ForkedDir: filepath.Join(wsRoot, "mods", "forked"),
	}

	env := &config.Environment{Machine: "linux-box"}
	paths := &games.GamePaths{ModsDir: dataDir}
	mf := &manifest.Manifest{
		Mods: map[string]manifest.ModEntry{
			"mod-alpha": {Source: "custom"},
			"mod-beta":  {Source: "custom"},
		},
	}

	driver := &hd2.HD2Driver{}

	// Run DeployLinks
	results, err := driver.DeployLinks(env, ws, mf, paths, false)
	if err != nil {
		t.Fatalf("DeployLinks failed: %v", err)
	}

	// 1. Verify base game file was preserved untouched
	if data, err := os.ReadFile(baseGameArchive); err != nil || string(data) != "base-game-data" {
		t.Fatalf("base game archive was modified or removed!")
	}

	// 2. Verify manual existing patch_0 was preserved untouched
	if data, err := os.ReadFile(existingPatch); err != nil || string(data) != "existing-manual-patch" {
		t.Fatalf("existing manual patch_0 was overwritten!")
	}

	// 3. Since patch_0 already existed as non-symlink, managed patches should start at patch_1 and patch_2
	p1 := filepath.Join(dataDir, "9ba626afa44a3aa3.patch_1")
	p1Stream := filepath.Join(dataDir, "9ba626afa44a3aa3.patch_1.stream")
	p2 := filepath.Join(dataDir, "9ba626afa44a3aa3.patch_2")
	p2GPU := filepath.Join(dataDir, "9ba626afa44a3aa3.patch_2.gpu_resources")

	if fi, err := os.Lstat(p1); err != nil || (fi.Mode()&os.ModeSymlink) == 0 {
		t.Fatalf("expected symlink for patch_1 at %s, got err: %v", p1, err)
	}
	if fi, err := os.Lstat(p1Stream); err != nil || (fi.Mode()&os.ModeSymlink) == 0 {
		t.Fatalf("expected symlink for patch_1.stream at %s", p1Stream)
	}
	if fi, err := os.Lstat(p2); err != nil || (fi.Mode()&os.ModeSymlink) == 0 {
		t.Fatalf("expected symlink for patch_2 at %s, got err: %v", p2, err)
	}
	if fi, err := os.Lstat(p2GPU); err != nil || (fi.Mode()&os.ModeSymlink) == 0 {
		t.Fatalf("expected symlink for patch_2.gpu_resources at %s", p2GPU)
	}

	// 4. Test Stale Symlink Purge: disable mod-beta, rerun DeployLinks
	f := false
	mf.Mods["mod-beta"] = manifest.ModEntry{Source: "custom", Enabled: &f}

	results, err = driver.DeployLinks(env, ws, mf, paths, false)
	if err != nil {
		t.Fatalf("DeployLinks rerun failed: %v", err)
	}

	// patch_2 symlinks should now be unlinked
	if _, err := os.Lstat(p2); !os.IsNotExist(err) {
		t.Errorf("expected patch_2 to be removed after disabling mod-beta, but it still exists")
	}
	if _, err := os.Lstat(p2GPU); !os.IsNotExist(err) {
		t.Errorf("expected patch_2.gpu_resources to be removed after disabling mod-beta")
	}

	// patch_1 should remain
	if _, err := os.Lstat(p1); err != nil {
		t.Errorf("expected patch_1 to remain active: %v", err)
	}

	// Base game file and manual patch_0 still untouched
	if _, err := os.Stat(baseGameArchive); err != nil {
		t.Errorf("base game archive missing: %v", err)
	}
	if _, err := os.Stat(existingPatch); err != nil {
		t.Errorf("existing manual patch missing: %v", err)
	}

	_ = results
}

func TestHD2Driver_DryRun(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "modr_hd2_dryrun_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	wsRoot := filepath.Join(tempDir, "games", "helldivers-2")
	customModsDir := filepath.Join(wsRoot, "mods", "custom")
	dataDir := filepath.Join(tempDir, "game", "data")

	_ = os.MkdirAll(customModsDir, 0755)
	_ = os.MkdirAll(dataDir, 0755)

	modDir := filepath.Join(customModsDir, "test-mod")
	_ = os.MkdirAll(modDir, 0755)
	_ = os.WriteFile(filepath.Join(modDir, "9ba626afa44a3aa3.patch_0"), []byte("dry-data"), 0644)

	ws := &workspace.Workspace{
		GameName:  "helldivers2",
		RootDir:   wsRoot,
		CustomDir: customModsDir,
	}
	paths := &games.GamePaths{ModsDir: dataDir}
	env := &config.Environment{Machine: "linux-box"}
	mf := &manifest.Manifest{
		Mods: map[string]manifest.ModEntry{
			"test-mod": {Source: "custom"},
		},
	}

	driver := &hd2.HD2Driver{}
	results, err := driver.DeployLinks(env, ws, mf, paths, true)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != linker.StatusCreated {
		t.Errorf("expected StatusCreated in dry-run, got %s", results[0].Status)
	}

	// Verify no actual symlink was created on disk in dry-run
	expectedTarget := filepath.Join(dataDir, "9ba626afa44a3aa3.patch_0")
	if _, err := os.Lstat(expectedTarget); !os.IsNotExist(err) {
		t.Errorf("dry-run should not create files on disk, but %s was found", expectedTarget)
	}
}
