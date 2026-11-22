package hd2

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/games"
	"rimworld-mods/internal/linker"
	"rimworld-mods/internal/manifest"
)

func init() {
	driver := &HD2Driver{}
	games.RegisterDriver(driver)
	games.RegisterDriver(&namedHD2Driver{HD2Driver: driver, name: "helldivers2"})
}

type namedHD2Driver struct {
	*HD2Driver
	name string
}

func (n *namedHD2Driver) Name() string {
	return n.name
}

// HD2Driver provides Helldivers 2 mod management with collision-free patch index stacking
// and safe non-destructive symlink reconciliation in the game's shared data/ directory.
type HD2Driver struct{}

func (h *HD2Driver) Name() string {
	return "hd2"
}

func (h *HD2Driver) DisplayName() string {
	return "Helldivers 2"
}

// patchRegex matches Helldivers 2 engine patch files, e.g. "9ba626afa44a3aa3.patch_0.stream"
var patchRegex = regexp.MustCompile(`^([0-9a-fA-F]+)\.patch_(\d+)(.*)$`)

func (h *HD2Driver) ResolvePaths(env *config.Environment, ws *workspace.Workspace) (*games.GamePaths, error) {
	if env.Machine == "steam-deck" {
		return nil, fmt.Errorf("helldivers 2 is strictly disallowed on Steam Deck")
	}

	home, _ := os.UserHomeDir()

	// Environment variable override
	for _, envKey := range []string{"MODR_HELLDIVERS2_MODS_DIR", "MODR_HD2_MODS_DIR"} {
		if val := os.Getenv(envKey); val != "" {
			return &games.GamePaths{ModsDir: expandPath(val, home)}, nil
		}
	}

	if ws.Spec != nil && ws.Spec.Paths != nil {
		machineKey := env.Machine
		aliases := []string{machineKey}
		if machineKey == "linux-box" {
			aliases = append(aliases, "desktop")
		} else if machineKey == "linux-book" {
			aliases = append(aliases, "laptop", "desktop")
		}

		for _, alias := range aliases {
			if mp, ok := ws.Spec.Paths[alias]; ok {
				if mDir, ok := mp["mods_dir"]; ok && mDir != "" {
					gDir := mp["game_dir"]
					resolvedMDir := expandPath(mDir, home)
					resolvedGDir := expandPath(gDir, home)
					if resolvedGDir == "" && resolvedMDir != "" {
						resolvedGDir = filepath.Dir(resolvedMDir)
					}
					return &games.GamePaths{
						GameDir: resolvedGDir,
						ModsDir: resolvedMDir,
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no data/mods directory path configured for Helldivers 2 on machine %s", env.Machine)
}

func (h *HD2Driver) PreApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) error {
	if env.Machine == "steam-deck" {
		return fmt.Errorf("helldivers 2 is strictly disallowed on Steam Deck and cannot be deployed on this machine")
	}
	return nil
}

func (h *HD2Driver) PostApply(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, dryRun bool) (*games.ApplyResult, error) {
	return &games.ApplyResult{}, nil
}

type patchFileInfo struct {
	archiveHash string
	suffix      string
	srcPath     string
}

// DeployLinks sequences patch files across declared mods and deploys symlinks into the data/ folder.
// Base game data files and manual (non-symlink) patches are never modified or purged.
func (h *HD2Driver) DeployLinks(env *config.Environment, ws *workspace.Workspace, mf *manifest.Manifest, paths *games.GamePaths, dryRun bool) ([]linker.LinkResult, error) {
	if env.Machine == "steam-deck" {
		return nil, fmt.Errorf("helldivers 2 is strictly disallowed on Steam Deck")
	}

	dataDir := paths.ModsDir
	if dataDir == "" {
		return nil, fmt.Errorf("data directory is not configured for Helldivers 2")
	}

	if !dryRun {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to ensure data directory %s: %w", dataDir, err)
		}
	}

	// 1. Detect existing non-symlink patch files in dataDir to avoid collisions
	maxExistingIndex := make(map[string]int)
	if entries, err := os.ReadDir(dataDir); err == nil {
		for _, e := range entries {
			m := patchRegex.FindStringSubmatch(e.Name())
			if len(m) >= 3 {
				hash := strings.ToLower(m[1])
				fullPath := filepath.Join(dataDir, e.Name())
				fi, err := os.Lstat(fullPath)
				if err == nil && (fi.Mode()&os.ModeSymlink) == 0 {
					// Real non-symlink file
					if idx, err := strconv.Atoi(m[2]); err == nil {
						if currentMax, ok := maxExistingIndex[hash]; !ok || idx > currentMax {
							maxExistingIndex[hash] = idx
						}
					}
				}
			}
		}
	}

	// Track next available patch index per archive hash
	nextPatchIndex := make(map[string]int)
	for hash, maxIdx := range maxExistingIndex {
		nextPatchIndex[hash] = maxIdx + 1
	}

	// 2. Discover candidate mods
	candidateMods := discoverHD2CandidateMods(ws, mf)

	// Sort mod IDs deterministically by priority (ascending), then alphabetically
	var modIDs []string
	for id := range candidateMods {
		modIDs = append(modIDs, id)
	}
	sort.Slice(modIDs, func(i, j int) bool {
		pI := 0
		pJ := 0
		if mf != nil {
			if entry, ok := mf.Mods[modIDs[i]]; ok {
				pI = entry.Priority
			}
			if entry, ok := mf.Mods[modIDs[j]]; ok {
				pJ = entry.Priority
			}
		}
		if pI != pJ {
			return pI < pJ
		}
		return modIDs[i] < modIDs[j]
	})

	var results []linker.LinkResult
	managedDestPaths := make(map[string]bool)

	// 3. Process each mod and link patch files
	for _, modID := range modIDs {
		modPath := candidateMods[modID]
		patches, err := scanModPatchFiles(modPath)
		if err != nil {
			results = append(results, linker.LinkResult{
				ModID:   modID,
				Status:  linker.StatusError,
				Message: fmt.Sprintf("failed scanning mod files: %v", err),
			})
			continue
		}

		if len(patches) == 0 {
			// Mod contains no patch files
			results = append(results, linker.LinkResult{
				ModID:      modID,
				SourcePath: modPath,
				Status:     linker.StatusSkipped,
				Message:    "No .patch_* files found in mod directory",
			})
			continue
		}

		// Group patches by archive hash for this mod
		byHash := make(map[string][]patchFileInfo)
		for _, p := range patches {
			byHash[p.archiveHash] = append(byHash[p.archiveHash], p)
		}

		// Sort hashes deterministically
		var hashes []string
		for h := range byHash {
			hashes = append(hashes, h)
		}
		sort.Strings(hashes)

		for _, hash := range hashes {
			files := byHash[hash]
			assignedIndex := nextPatchIndex[hash]
			nextPatchIndex[hash]++

			for _, f := range files {
				destFileName := fmt.Sprintf("%s.patch_%d%s", f.archiveHash, assignedIndex, f.suffix)
				destPath := filepath.Join(dataDir, destFileName)
				managedDestPaths[filepath.Clean(destPath)] = true

				res := linkSinglePatch(modID, f.srcPath, destPath, dryRun)
				results = append(results, res)
			}
		}
	}

	// 4. Safely purge stale symlinks pointing into this workspace
	wsRootClean, _ := filepath.Abs(ws.RootDir)
	if entries, err := os.ReadDir(dataDir); err == nil {
		for _, e := range entries {
			destPath := filepath.Join(dataDir, e.Name())
			fi, err := os.Lstat(destPath)
			if err != nil || (fi.Mode()&os.ModeSymlink) == 0 {
				continue // NEVER touch regular base game files or non-symlinks!
			}

			target, err := os.Readlink(destPath)
			if err != nil {
				continue
			}

			var absTarget string
			if filepath.IsAbs(target) {
				absTarget = filepath.Clean(target)
			} else {
				absTarget = filepath.Clean(filepath.Join(dataDir, target))
			}

			// Only purge if the symlink points to our monorepo workspace and is not currently managed
			if strings.HasPrefix(absTarget, wsRootClean) && !managedDestPaths[filepath.Clean(destPath)] {
				if dryRun {
					results = append(results, linker.LinkResult{
						ModID:      e.Name(),
						DestPath:   destPath,
						SourcePath: absTarget,
						Status:     linker.StatusUnlinked,
						Message:    "[DRY-RUN] Would remove stale Helldivers 2 mod symlink",
					})
				} else {
					if err := os.Remove(destPath); err != nil {
						results = append(results, linker.LinkResult{
							ModID:    e.Name(),
							DestPath: destPath,
							Status:   linker.StatusError,
							Message:  fmt.Sprintf("failed removing stale symlink: %v", err),
						})
					} else {
						results = append(results, linker.LinkResult{
							ModID:      e.Name(),
							DestPath:   destPath,
							SourcePath: absTarget,
							Status:     linker.StatusUnlinked,
							Message:    "Removed stale Helldivers 2 mod symlink",
						})
					}
				}
			}
		}
	}

	return results, nil
}

func linkSinglePatch(modID, srcPath, destPath string, dryRun bool) linker.LinkResult {
	fi, err := os.Lstat(destPath)
	if err == nil {
		if (fi.Mode() & os.ModeSymlink) != 0 {
			target, err := os.Readlink(destPath)
			if err == nil {
				var absTarget string
				if filepath.IsAbs(target) {
					absTarget = filepath.Clean(target)
				} else {
					absTarget = filepath.Clean(filepath.Join(filepath.Dir(destPath), target))
				}
				absSrc, _ := filepath.Abs(srcPath)
				if absTarget == filepath.Clean(absSrc) {
					return linker.LinkResult{
						ModID:      filepath.Base(destPath),
						SourcePath: srcPath,
						DestPath:   destPath,
						Status:     linker.StatusUpToDate,
						Message:    fmt.Sprintf("[%s] Patch symlink up-to-date", modID),
					}
				}
			}
			// Symlink exists but points elsewhere: remove to relink
			if !dryRun {
				_ = os.Remove(destPath)
			}
		} else {
			// Real file exists: DO NOT OVERWRITE!
			return linker.LinkResult{
				ModID:      filepath.Base(destPath),
				SourcePath: srcPath,
				DestPath:   destPath,
				Status:     linker.StatusError,
				Message:    fmt.Sprintf("[%s] Collision: destination %s is a non-symlink file and will not be overwritten", modID, filepath.Base(destPath)),
			}
		}
	}

	if dryRun {
		return linker.LinkResult{
			ModID:      filepath.Base(destPath),
			SourcePath: srcPath,
			DestPath:   destPath,
			Status:     linker.StatusCreated,
			Message:    fmt.Sprintf("[%s] [DRY-RUN] Would link patch to %s", modID, filepath.Base(destPath)),
		}
	}

	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		absSrc = srcPath
	}

	if err := os.Symlink(absSrc, destPath); err != nil {
		return linker.LinkResult{
			ModID:      filepath.Base(destPath),
			SourcePath: absSrc,
			DestPath:   destPath,
			Status:     linker.StatusError,
			Message:    err.Error(),
		}
	}

	return linker.LinkResult{
		ModID:      filepath.Base(destPath),
		SourcePath: absSrc,
		DestPath:   destPath,
		Status:     linker.StatusCreated,
		Message:    fmt.Sprintf("[%s] Linked patch -> %s", modID, filepath.Base(destPath)),
	}
}

func scanModPatchFiles(modDir string) ([]patchFileInfo, error) {
	var patches []patchFileInfo
	err := filepath.Walk(modDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		fileName := filepath.Base(path)
		m := patchRegex.FindStringSubmatch(fileName)
		if len(m) >= 4 {
			patches = append(patches, patchFileInfo{
				archiveHash: strings.ToLower(m[1]),
				suffix:      m[3],
				srcPath:     path,
			})
		}
		return nil
	})
	return patches, err
}

func discoverHD2CandidateMods(ws *workspace.Workspace, mf *manifest.Manifest) map[string]string {
	candidates := make(map[string]string)
	disabled := make(map[string]bool)

	if mf != nil {
		for modID, entry := range mf.Mods {
			if !entry.IsEnabled() {
				disabled[modID] = true
				continue
			}
			var candPath string
			if entry.IsCustom() {
				candPath = filepath.Join(ws.CustomDir, modID)
			} else if entry.IsForked() {
				candPath = filepath.Join(ws.ForkedDir, modID)
			} else {
				candPath = filepath.Join(ws.VendorDir, modID)
			}

			if fi, err := os.Stat(candPath); err == nil && fi.IsDir() {
				candidates[modID] = candPath
			}
		}
	}

	// Auto-discover in custom/
	if entries, err := os.ReadDir(ws.CustomDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !disabled[e.Name()] {
				candidates[e.Name()] = filepath.Join(ws.CustomDir, e.Name())
			}
		}
	}

	// Auto-discover in forked/
	if entries, err := os.ReadDir(ws.ForkedDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !disabled[e.Name()] {
				candidates[e.Name()] = filepath.Join(ws.ForkedDir, e.Name())
			}
		}
	}

	return candidates
}

func expandPath(path, home string) string {
	if path == "" {
		return ""
	}
	path = os.ExpandEnv(path)
	if home != "" && (path == "~" || strings.HasPrefix(path, "~/")) {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}
