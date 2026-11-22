# AGENTS.md

Operational guidelines, conventions, and architectural context for autonomous AI agents working in this repository.

---

## 1. Project Overview & Architecture

This repository is a centralized personal monorepo for managing, versioning, authoring, and deploying custom game mod setups across Linux environments: **Arch Linux Workstations** (`linux-box` desktop, `linux-book` laptop) and **Steam Deck** (`steam-deck` on SteamOS).

### Key Architectural Pillars
- **Unified Operational CLI (`modr`):** A native Go CLI (`cmd/modr`, `internal/`) that orchestrates all mod lifecycle operations across games without fragile shell script glue.
- **Multi-Game Workspaces (`games/<game>/`):** Each game maintains an isolated workspace with its own `game.yaml` configuration, `manifests/` (`mods.yaml`, `mods.lock.yaml`), and `mods/` (`vendor/`, `forked/`, `custom/`).
- **Session-Scoped Terminal Context:** Pure-Go TTY session context (`modr use <game>`) written to `/run/user/$UID/modr/` without requiring shell wrappers or `.zshrc` modifications.
- **Game Driver Engine (`internal/games`):** Pluggable driver interface:
  - `RimWorldDriver`: Topological DAG sorting over `About.xml` metadata (`internal/order`) and automatic `ModsConfig.xml` deployment.
  - `GenericDriver`: Drop-in symlink manager for games requiring simple folder injection (e.g. BepInEx plugins in Star Valor).
  - `HD2Driver`: Helldivers 2 driver with patch stacking (`.patch_0`, `.patch_1`, ...) and safe symlink reconciliation in `data/`. Strictly disallows Steam Deck execution and synchronization.
- **Decoupled Mod Management:** Bypasses Steam Workshop client lock-in using CLI fetching (`steamcmd`, GitHub releases, git repos).
- **Deterministic Pinning:** Uses declarative manifests and lockfiles with rollback capabilities.
- **Idempotent Deployment:** Symlinks cached vendor mods, forked patches, and custom mods directly into game targets.
- **Remote Synchronization:** Synchronizes repository manifests, lockfiles, and active mod lists over SSH/rsync to Steam Deck targets via `modr deck push`.

---

## 2. Directory Layout

```text
.
├── Makefile                     # Build and install runner for modr CLI
├── bin/                         # Compiled binaries (bin/modr, bin/rimmod symlink)
├── cmd/modr/                    # Cobra CLI application commands
├── config/                      # Machine-specific environment overrides
│   ├── config.env.example       # Template environment variables
│   ├── env.desktop.env          # Arch Linux desktop defaults (linux-box)
│   ├── env.laptop.env           # Arch Linux laptop defaults (linux-book)
│   ├── env.steamdeck.env        # Steam Deck defaults (steam-deck)
│   └── local.env                # Local host-specific active config (required, git-ignored)
├── docs/                        # Deep-dive guides and documentation
│   ├── mod-management.md        # Detailed guide on declaring, updating, and rolling back mods
│   └── steamdeck-setup.md       # Steam Deck SSH configuration and paths
├── internal/
│   ├── config/                  # Host & path resolution
│   ├── core/                    # Session context & workspace resolution
│   ├── games/                   # Game drivers (generic, rimworld)
│   ├── fetcher/                 # SteamCMD, GitHub release, and Git fetch engine
│   ├── linker/                  # Idempotent symlink reconciliation manager
│   ├── order/                   # RimWorld About.xml topological DAG engine
│   ├── updater/                 # Upstream API check and lockfile sync
│   └── deck/                    # Remote SSH/rsync push/pull operations
├── games/
│   ├── rimworld/                # RimWorld workspace
│   │   ├── game.yaml            # Paths & driver spec (driver: rimworld)
│   │   ├── manifests/           # mods.yaml, mods.lock.yaml
│   │   └── mods/                # vendor/ (cache), forked/, custom/
│   ├── starvalor/               # Star Valor workspace
│   │   ├── game.yaml            # Paths & driver spec (driver: generic)
│   │   ├── manifests/           # mods.yaml, mods.lock.yaml
│   │   └── mods/                # vendor/ (cache), forked/, custom/
│   └── helldivers-2/            # Helldivers 2 workspace (CLI: helldivers2, hd2; no Steam Deck)
│       ├── game.yaml            # Paths & driver spec (driver: hd2, allow_steam_deck: false)
│       ├── manifests/           # mods.yaml, mods.lock.yaml
│       └── mods/                # vendor/ (cache), forked/, custom/
└── templates/                   # Starter templates for custom mod scaffolding
```

---

## 3. Autonomous Execution & Safety Boundaries

### Authorized Autonomous Operations
As defined in [`GEMINI.md`](file:///home/tylerkilburn/Git/rimworld-mods/GEMINI.md), agents are authorized to autonomously execute local CLI tasks without asking for confirmation:
- Compilation & Builds: `dotnet build`, `make build`, `make install`, `modr dev build <mod> --dry-run`.
- Mod Management & Inspection: `modr status`, `modr apply --dry-run`, `modr update --dry-run`, `modr tidy --dry-run`, `modr order`, `modr deck push --check`, `modr deck push --dry-run`, `modr games`, `modr use`.
- Local Read/Inspection/Search: `fd`, `rg`, `ls`, `cat`, Go test runs (`go test ./...`).
- Scoped Editing: Use [`replace_file_content`](file:///home/tylerkilburn/Git/rimworld-mods) and [`write_to_file`](file:///home/tylerkilburn/Git/rimworld-mods) for precise modifications.

### Operations Requiring Confirmation or User Hands-Off
- **Never Commit Code:** Do not run `git commit` or suggest committing automatically; the user manages git commits manually.
- **No Remote Network Mutations:** Do not run mutating remote operations (e.g., `git push`, external cloud API state mutations) without explicit user authorization.
- **Steam Deck Sync Confirmation:** Modifying files on the remote Steam Deck (`modr deck push`) mutates remote target state. Run dry-runs (`modr deck push --dry-run`) first or confirm before running mutating syncs.
- **No DLL Crawling / Decompilation:** Never crawl, disassemble, inspect IL tokens of, or reflect into compiled binary DLLs (e.g. `Assembly-CSharp.dll`, engine assemblies). All modding decisions, logic, and debugging must be derived from mod source code, game XML definitions, documentation, and observable mod state.

---

## 4. Development & Tooling Standards

### Go Tooling (`modr`)
- Adhere to idiomatic Go: explicit error handling, table-driven unit tests, and clean package separation in `internal/`.
- Maintain build output path to [`bin/modr`](file:///home/tylerkilburn/Git/rimworld-mods/bin/modr) and installation to `~/bin/modr` (with `~/bin/rimmod` alias) via `make install`.
- Dry-run flags: Every mutating command must provide `--dry-run` (`-n`) preview capabilities.

### Custom Mod Development
- **XML & Defs Configuration:** Custom genes, numerical balance values, skill gates, time intervals, and tuning factors must be defined in XML (standard Def fields, `DefModExtension`s, or custom `Def` classes) rather than hardcoded in C# logic.
- **Target Framework & Game Version:** C# mod projects target appropriate runtime (.NET Framework 4.7.2 for RimWorld 1.6, or BepInEx targets for Unity games like Star Valor).
