package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rimworld-mods/internal/config"
)

// BuildMod compiles C# assemblies for a custom mod using dotnet build.
func BuildMod(env *config.Environment, modName string, configuration string, dryRun bool) error {
	if configuration == "" {
		configuration = "Release"
	}

	findModDir := func() string {
		candidates := []string{
			filepath.Join(env.RepoRoot, "games", "rimworld", "mods", "custom", modName),
			filepath.Join(env.RepoRoot, "games", "rimworld", "mods", "forked", modName),
			filepath.Join(env.RepoRoot, "mods", "custom", modName),
			filepath.Join(env.RepoRoot, "mods", "forked", modName),
		}
		// Glob across any game workspace
		globCustom, _ := filepath.Glob(filepath.Join(env.RepoRoot, "games", "*", "mods", "custom", modName))
		candidates = append(candidates, globCustom...)
		globForked, _ := filepath.Glob(filepath.Join(env.RepoRoot, "games", "*", "mods", "forked", modName))
		candidates = append(candidates, globForked...)

		for _, c := range candidates {
			if fi, err := os.Stat(c); err == nil && fi.IsDir() {
				return c
			}
		}
		return ""
	}

	modDir := findModDir()
	if modDir == "" {
		return fmt.Errorf("mod directory not found in games/*/mods/custom/ or forked/: %s", modName)
	}

	// Find .csproj in mod directory or Source/
	findCsproj := func() string {
		searchGlobs := [][]string{
			{modDir, "*.csproj"},
			{modDir, "Source", "*.csproj"},
			{modDir, "Source", "*", "*.csproj"},
		}
		for _, g := range searchGlobs {
			matches, _ := filepath.Glob(filepath.Join(g...))
			for _, m := range matches {
				base := filepath.Base(m)
				if strings.Contains(base, "Backup") || strings.Contains(base, "old") {
					continue
				}
				return m
			}
		}
		return ""
	}

	csprojPath := findCsproj()

	if csprojPath == "" {
		return fmt.Errorf("no .csproj file found in %s or %s/Source", modDir, modDir)
	}

	args := []string{"build", csprojPath, "-c", configuration}

	if dryRun {
		fmt.Printf("  [DRY-RUN] Would run: dotnet %v\n", args)
		return nil
	}

	cmd := exec.Command("dotnet", args...)
	cmd.Dir = modDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dotnet build failed: %w", err)
	}

	return nil
}
