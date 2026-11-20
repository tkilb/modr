package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"rimworld-mods/internal/config"
	"rimworld-mods/internal/core/workspace"
)

func newUseCmd(env *config.Environment) *cobra.Command {
	var clearFlag bool

	cmd := &cobra.Command{
		Use:   "use [game]",
		Short: "Set or inspect active game context for the current terminal session",
		Long: `Sets or displays the active game workspace for your current terminal tab.
This state is scoped to your current terminal session and does not alter global config or .zshrc.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if clearFlag {
				if err := sessionMgr.ClearActiveGame(); err != nil {
					return fmt.Errorf("failed to clear session context: %w", err)
				}
				fmt.Printf("%s✔ Session context cleared.%s\n", colorGreen, colorReset)
				return nil
			}

			if len(args) == 0 {
				active, err := sessionMgr.GetActiveGame()
				if err != nil {
					return err
				}
				key := sessionMgr.GetSessionKey()
				if active == "" {
					available, _ := workspace.ListWorkspaces(env.RepoRoot)
					fmt.Printf("No active game context set for session (%s).\n", key)
					if len(available) > 0 {
						fmt.Printf("Available games: %s\n", strings.Join(available, ", "))
						fmt.Printf("Run 'modr use <game>' to select one.\n")
					}
					return nil
				}
				fmt.Printf("Active game: %s%s%s (session %s)\n", colorCyan, active, colorReset, key)
				return nil
			}

			targetGame := strings.ToLower(args[0])
			ws, err := workspace.FindWorkspace(env.RepoRoot, targetGame)
			if err != nil {
				available, _ := workspace.ListWorkspaces(env.RepoRoot)
				return fmt.Errorf("unknown game %q. Available games: %s", targetGame, strings.Join(available, ", "))
			}

			if err := sessionMgr.SetActiveGame(targetGame); err != nil {
				return fmt.Errorf("failed to set session context: %w", err)
			}

			key := sessionMgr.GetSessionKey()
			displayName := targetGame
			if ws.Spec != nil && ws.Spec.DisplayName != "" {
				displayName = ws.Spec.DisplayName
			}

			fmt.Printf("%s✔ Switched active game context to %s (%s) for session %s%s\n",
				colorGreen, displayName, targetGame, key, colorReset)
			return nil
		},
	}

	cmd.Flags().BoolVar(&clearFlag, "clear", false, "Clear current session context")
	return cmd
}

func newGamesCmd(env *config.Environment) *cobra.Command {
	return &cobra.Command{
		Use:     "games",
		Aliases: []string{"list"},
		Short:   "List configured game workspaces in the monorepo",
		RunE: func(cmd *cobra.Command, args []string) error {
			gamesList, err := workspace.ListWorkspaces(env.RepoRoot)
			if err != nil {
				return err
			}

			active, _ := sessionMgr.GetActiveGame()

			fmt.Printf("%sConfigured Game Workspaces:%s\n", colorBold, colorReset)
			fmt.Printf("%s %s %s\n", pad("GAME ID", 18), pad("DISPLAY NAME", 24), pad("DRIVER", 12))
			fmt.Println(strings.Repeat("-", 60))

			for _, g := range gamesList {
				ws, err := workspace.FindWorkspace(env.RepoRoot, g)
				if err != nil {
					continue
				}
				disp := g
				drv := "generic"
				if ws.Spec != nil {
					if ws.Spec.DisplayName != "" {
						disp = ws.Spec.DisplayName
					}
					if ws.Spec.Driver != "" {
						drv = ws.Spec.Driver
					}
				}

				marker := "  "
				color := ""
				if g == active {
					marker = "* "
					color = colorCyan
				}

				fmt.Printf("%s%s %s %s\n",
					marker,
					colorPad(g, 16, color),
					pad(disp, 24),
					pad(drv, 12),
				)
			}

			if active != "" {
				fmt.Printf("\n(* = active session context)\n")
			}
			return nil
		},
	}
}
