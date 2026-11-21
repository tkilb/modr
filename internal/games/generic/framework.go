package generic

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/fetcher"
	"rimworld-mods/internal/games"
)

// FrameworkHandler defines the operations required to inspect and bootstrap a game modding framework.
type FrameworkHandler interface {
	Type() string
	IsInstalled(gameDir string, version string) (installed bool, details string)
	Install(gameDir, version, customURL string, dryRun bool) error
}

var frameworkHandlers = map[string]FrameworkHandler{
	"bepinex5": &BepInEx5Handler{},
	"bepinex":  &BepInEx5Handler{},
}

// RegisterFrameworkHandler allows registering handlers for other frameworks (e.g. melonloader, ue4ss).
func RegisterFrameworkHandler(h FrameworkHandler) {
	frameworkHandlers[strings.ToLower(h.Type())] = h
}

// GetFrameworkHandler returns a handler for the given framework type.
func GetFrameworkHandler(fwType string) (FrameworkHandler, bool) {
	h, ok := frameworkHandlers[strings.ToLower(fwType)]
	return h, ok
}

// EnsureFramework verifies the framework is installed, or automatically installs it if missing.
func EnsureFramework(fw *workspace.FrameworkSpec, paths *games.GamePaths, gameDisplayName string, dryRun bool) error {
	if fw == nil {
		return nil
	}

	if paths.GameDir == "" {
		return fmt.Errorf("game directory not resolved for framework %s", fw.Type)
	}

	// Fail cleanly if the game itself is not installed on this machine
	if fi, err := os.Stat(paths.GameDir); err != nil || !fi.IsDir() {
		if dryRun {
			fmt.Printf("  \033[36m[DRY-RUN]\033[0m Target game directory not found locally: %s\n", paths.GameDir)
			return nil
		}
		return fmt.Errorf("game install directory does not exist: %s\nPlease install %s on this machine before applying mods", paths.GameDir, gameDisplayName)
	}

	handler, ok := GetFrameworkHandler(fw.Type)
	if !ok {
		return fmt.Errorf("unsupported framework type %q declared in game.yaml", fw.Type)
	}

	installed, detail := handler.IsInstalled(paths.GameDir, fw.Version)
	if installed {
		return nil
	}

	fmt.Printf("\n==> Framework requirement: %s\n", detail)
	return handler.Install(paths.GameDir, fw.Version, fw.URL, dryRun)
}

// CheckFrameworkStatus returns whether the required framework is installed in the game root.
func CheckFrameworkStatus(ws *workspace.Workspace, paths *games.GamePaths) (installed bool, details string) {
	if ws.Spec == nil || ws.Spec.Framework == nil {
		return true, "None required"
	}
	fw := ws.Spec.Framework
	handler, ok := GetFrameworkHandler(fw.Type)
	if !ok {
		return false, fmt.Sprintf("unsupported framework type: %s", fw.Type)
	}
	return handler.IsInstalled(paths.GameDir, fw.Version)
}

// BepInEx5Handler manages BepInEx 5.x x64 mod loader installation for Unity games.
type BepInEx5Handler struct{}

func (b *BepInEx5Handler) Type() string {
	return "bepinex5"
}

func (b *BepInEx5Handler) IsInstalled(gameDir string, version string) (bool, string) {
	if gameDir == "" {
		return false, "game directory not resolved"
	}
	winhttp := filepath.Join(gameDir, "winhttp.dll")
	bepDir := filepath.Join(gameDir, "BepInEx", "core")
	_, errDll := os.Stat(winhttp)
	fiCore, errCore := os.Stat(bepDir)
	if errDll == nil && errCore == nil && fiCore.IsDir() {
		return true, fmt.Sprintf("BepInEx %s (Installed)", version)
	}
	return false, fmt.Sprintf("BepInEx %s missing in %s", version, gameDir)
}

func (b *BepInEx5Handler) Install(gameDir, version, customURL string, dryRun bool) error {
	dlURL := customURL
	if dlURL == "" {
		if version == "" {
			version = "5.4.23.4"
		}
		dlURL = fmt.Sprintf("https://github.com/BepInEx/BepInEx/releases/download/v%s/BepInEx_win_x64_%s.zip", version, version)
	}

	if dryRun {
		fmt.Printf("  \033[36m[DRY-RUN]\033[0m Would download BepInEx from:\n    %s\n", dlURL)
		fmt.Printf("  \033[36m[DRY-RUN]\033[0m Would extract into: %s\n", gameDir)
		return nil
	}

	fmt.Printf("==> Downloading BepInEx %s from %s...\n", version, dlURL)
	tmpZip, err := os.CreateTemp("", "modr-bepinex-*.zip")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmpZip.Close()
		_ = os.Remove(tmpZip.Name())
	}()

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(dlURL)
	if err != nil {
		return fmt.Errorf("failed to download BepInEx archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed downloading BepInEx, HTTP status: %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpZip, resp.Body); err != nil {
		return fmt.Errorf("failed to write BepInEx archive: %w", err)
	}
	_ = tmpZip.Close()

	fmt.Printf("==> Extracting BepInEx into %s...\n", gameDir)
	if err := fetcher.UnzipToDir(tmpZip.Name(), gameDir); err != nil {
		return fmt.Errorf("failed extracting BepInEx archive: %w", err)
	}

	// Ensure BepInEx/plugins exists
	_ = os.MkdirAll(filepath.Join(gameDir, "BepInEx", "plugins"), 0755)

	fmt.Printf("  \033[32m✔\033[0m BepInEx %s successfully installed into: %s\n", version, gameDir)
	fmt.Printf("  \033[33m[IMPORTANT]\033[0m Linux / Steam Deck Launch Option:\n")
	fmt.Printf("  Set Steam Launch Options for this game to: \033[1mWINEDLLOVERRIDES=\"winhttp=n,b\" %%command%%\033[0m\n\n")

	return nil
}
