# Modr Monorepo

Centralized control center for managing, versioning, authoring, and deploying game mods across Linux machines (**Arch Linux Desktop**, **Laptop**, and **Steam Deck**).

Currently supported games:
- **RimWorld:** Topological dependency sorting (`About.xml`), official DLC management, and automatic `ModsConfig.xml` deployment.
- **Star Valor:** Generic BepInEx mod management and plugin symlinking.
- **Helldivers 2:** Sequential patch stacking (`.patch_0`, `.patch_1`, ...) and safe symlink deployment into game `data/`. *(Desktop/Laptop only; strictly blocked on Steam Deck).*

---

## Key Features

- **Unified Multi-Game CLI (`modr`):** Single native Go CLI orchestrating mod lifecycles across multiple games.
- **Deterministic Pinning:** Declarative manifests (`games/<game>/manifests/mods.yaml`) and lockfiles (`mods.lock.yaml`) with rollback capabilities.
- **Scoped Terminal Sessions:** Switch active games with `modr use <game>` (isolated per-terminal via TTY session ID, zero `.zshrc` modifications required).
- **Idempotent Deployment:** Symlinks cached vendor downloads, forked patches, and authored custom mods directly into game plugin/mod folders.
- **Steam Deck Sync:** Synchronizes monorepo states and reconciles game setups over SSH/rsync in a single command.
- **Backward Compatible:** `rimmod` remains available as an alias to `modr` that automatically defaults to RimWorld.

---

## Quickstart & Daily Usage

### Working with Game Contexts

```bash
# List all configured game workspaces
modr games

# Switch active game for current terminal session
modr use rimworld
modr use starvalor

# Display current session context
modr use

# Clear session context
modr use --clear
```

### Daily Mod Operations

When a game context is active (or by passing `modr <game> <command>`):

```bash
# Inspect status, active mods, symlinks, and Deck reachability
modr status
modr rimworld status           # One-off explicit override

# Reconcile installation (fetch missing mods, link symlinks, generate configs)
modr apply
modr apply --dry-run           # Preview reconciliation

# Check upstream sources (Steam Workshop / GitHub) and update lockfile
modr update
modr update <mod_id>

# Prune unreferenced vendor downloads, lockfile entries, and dead symlinks
modr tidy

# Inspect computed topological load order (RimWorld)
modr order

# Import a new mod from Steam Workshop, GitHub release, or Git URL
modr import "https://steamcommunity.com/sharedfiles/filedetails/?id=..."
modr import "https://github.com/owner/repo" --slug my-mod --apply

# Authoring: compile custom C# mod and re-apply
modr dev build <mod_name>

# Scaffold a new starter template for an XML or C# mod
modr dev scaffold <mod_name> --type csharp

# Synchronize monorepo state to Steam Deck over SSH/rsync
modr deck push --check         # Test SSH connectivity
modr deck push --dry-run       # Preview transfer
modr deck push                 # Sync and reconcile remote game setup
```

---

## Directory Overview

```text
.
├── Makefile                     # Build & installation runner for modr CLI
├── bin/                         # Compiled binaries (modr, rimmod symlink alias)
├── cmd/modr/                    # Unified Cobra CLI entrypoint and commands
├── config/                      # Host-specific environment configuration
├── docs/                        # Detailed guides and deep-dive documentation
├── internal/
│   ├── config/                  # Environment resolution and host detection
│   ├── core/                    # Session context manager, workspace resolution
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

## Installation & Build

Build and install `modr` into `~/bin/` (default):

```bash
# Compile and install ~/bin/modr (and symlink ~/bin/rimmod)
make install

# Run Go test suite
make test
```
