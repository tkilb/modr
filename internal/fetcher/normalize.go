package fetcher

import (
	"os"
	"path/filepath"
)

// NormalizePluginDirs flattens common BepInEx packaging structures (e.g. BepInEx/plugins/<Mod>/ or plugins/<Mod>/).
func NormalizePluginDirs(dir string) {
	// Case 1: BepInEx/plugins
	bepPlugins := filepath.Join(dir, "BepInEx", "plugins")
	if fi, err := os.Stat(bepPlugins); err == nil && fi.IsDir() {
		PromoteDir(dir, bepPlugins)
		_ = os.RemoveAll(filepath.Join(dir, "BepInEx"))
	}

	// Case 2: plugins/
	plugins := filepath.Join(dir, "plugins")
	if fi, err := os.Stat(plugins); err == nil && fi.IsDir() {
		PromoteDir(dir, plugins)
		_ = os.RemoveAll(plugins)
	}

	// If root contains only 1 entry and that entry is a directory (not .git), unwrap it
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) == 1 && entries[0].IsDir() && entries[0].Name() != ".git" {
		PromoteDir(dir, filepath.Join(dir, entries[0].Name()))
	}
}

// PromoteDir moves all contents of subDir into rootDir.
func PromoteDir(rootDir, subDir string) {
	entries, err := os.ReadDir(subDir)
	if err != nil {
		return
	}
	tempStage, err := os.MkdirTemp("", "modr-promote-*")
	if err != nil {
		return
	}
	defer os.RemoveAll(tempStage)

	for _, e := range entries {
		src := filepath.Join(subDir, e.Name())
		dst := filepath.Join(tempStage, e.Name())
		if err := os.Rename(src, dst); err != nil {
			_ = CopyDir(src, dst)
		}
	}

	_ = os.RemoveAll(rootDir)
	_ = os.MkdirAll(rootDir, 0755)

	stagedEntries, _ := os.ReadDir(tempStage)
	for _, e := range stagedEntries {
		src := filepath.Join(tempStage, e.Name())
		dst := filepath.Join(rootDir, e.Name())
		if err := os.Rename(src, dst); err != nil {
			_ = CopyDir(src, dst)
		}
	}
}
