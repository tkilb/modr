package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/session"
	"rimworld-mods/internal/core/workspace"
	"rimworld-mods/internal/games"
	_ "rimworld-mods/internal/games/generic"
	_ "rimworld-mods/internal/games/rimworld"
)

var (
	explicitGameArg string
	flagGameValue   string
	sessionMgr      = session.NewSessionManager()
)

func main() {
	// If invoked via legacy alias 'rimmod', default target game to rimworld
	if strings.Contains(filepath.Base(os.Args[0]), "rimmod") {
		explicitGameArg = "rimworld"
	}

	env, err := config.LoadEnvironment("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing environment: %v\n", err)
		os.Exit(1)
	}

	// Support 'modr <game> <command>' syntax by intercepting the first arg if it matches a known game
	interceptGameArg(env)

	rootCmd := &cobra.Command{
		Use:   "modr",
		Short: "Deterministic multi-game mod orchestrator",
		Long: `modr is a deterministic, manifest-driven CLI for managing, fetching,
linking, ordering, and synchronizing game mods across Arch Linux workstations and Steam Deck.`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVarP(&flagGameValue, "game", "g", "", "Target game workspace (overrides session context)")

	// Register core commands
	rootCmd.AddCommand(
		newUseCmd(env),
		newGamesCmd(env),
		newApplyCmd(env),
		newStatusCmd(env),
		newUpdateCmd(env),
		newTidyCmd(env),
		newDeckCmd(env),
		newDevCmd(env),
		newOrderCmd(env),
		newImportCmd(env),
	)

	// Ensure Cobra's default help command is available
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long:  `Help provides help for any command in the application.`,
		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Printf("Unknown help topic %#q\n", args)
				_ = c.Root().Help()
			} else {
				cmd.InitDefaultHelpFlag()
				_ = cmd.Help()
			}
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// interceptGameArg checks if os.Args[1] is a known game and extracts it so subcommands receive clean arguments.
func interceptGameArg(env *config.Environment) {
	if len(os.Args) < 2 {
		return
	}

	firstArg := os.Args[1]
	// Skip if it's already a flag or standard top-level command
	if strings.HasPrefix(firstArg, "-") {
		return
	}
	standardCmds := map[string]bool{
		"use": true, "games": true, "list": true, "apply": true,
		"status": true, "update": true, "tidy": true, "deck": true,
		"dev": true, "order": true, "import": true, "add": true,
		"help": true, "completion": true,
	}
	if standardCmds[firstArg] {
		return
	}

	// Check if firstArg matches a known game directory or registered driver
	knownGames, err := workspace.ListWorkspaces(env.RepoRoot)
	if err != nil {
		return
	}

	for _, g := range knownGames {
		if strings.EqualFold(g, firstArg) {
			explicitGameArg = strings.ToLower(firstArg)
			// Remove the game name from os.Args
			os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
			return
		}
	}
}

// resolveTargetWorkspace determines the active game workspace using:
// 1. Explicit -g/--game flag
// 2. Intercepted <game> positional prefix (e.g. 'modr rimworld status')
// 3. TTY / Session context (from 'modr use <game>')
// 4. Current working directory (if inside games/<game>)
// 5. Default fallback to rimworld if only one game exists
func resolveTargetWorkspace(env *config.Environment) (*workspace.Workspace, games.GameDriver, error) {
	gameName := flagGameValue
	if gameName == "" {
		gameName = explicitGameArg
	}

	// Priority 3: Session context
	if gameName == "" {
		active, err := sessionMgr.GetActiveGame()
		if err == nil && active != "" {
			gameName = active
		}
	}

	// Priority 3: Working directory
	if gameName == "" {
		cwd, err := os.Getwd()
		if err == nil {
			gamesPrefix := filepath.Join(env.RepoRoot, "games")
			if strings.HasPrefix(cwd, gamesPrefix) {
				rel, err := filepath.Rel(gamesPrefix, cwd)
				if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
					parts := strings.Split(filepath.ToSlash(rel), "/")
					if len(parts) > 0 && parts[0] != "" {
						gameName = parts[0]
					}
				}
			}
		}
	}

	// Priority 4: If still empty, check if rimworld workspace exists as default
	if gameName == "" {
		available, _ := workspace.ListWorkspaces(env.RepoRoot)
		if len(available) == 1 {
			gameName = available[0]
		} else {
			return nil, nil, fmt.Errorf("no game context selected.\nRun 'modr use <game>' to set context, specify 'modr <game> <command>', or pass '-g <game>'.\nAvailable games: %s", strings.Join(available, ", "))
		}
	}

	ws, err := workspace.FindWorkspace(env.RepoRoot, gameName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed resolving workspace for game %q: %w", gameName, err)
	}

	driverName := "generic"
	if ws.Spec != nil && ws.Spec.Driver != "" {
		driverName = ws.Spec.Driver
	}
	drv, err := games.GetDriver(driverName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed resolving driver %q for game %q: %w", driverName, gameName, err)
	}

	return ws, drv, nil
}
