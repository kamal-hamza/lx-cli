# LX - LaTeX Notes Manager

A high-performance, opinionated CLI for managing LaTeX notes. Treat your notes as data while abstracting away file management complexity.

## Features

### Core Functionality

- **Note Management** - Create, edit, search, and organize LaTeX notes with ease
- **Template System** - Reusable LaTeX style templates for consistent formatting
- **Automated Building** - Compile individual notes or entire vault with latexmk
- **Knowledge Graph** - Interactive graph visualization of note connections
- **Link Management** - Track backlinks and outgoing connections between notes
- **Git Integration** - Built-in version control support
- **Health Checks** - Doctor command for vault diagnostics

### Power User Features ⚡

- **Smart Entry** - Type `lx physics` instead of `lx open physics` - the CLI intelligently determines the action
- **Interactive Dashboard** - Full-screen TUI for browsing, searching, and managing notes without leaving the terminal
- **Fuzzy Search** - Lightning-fast fuzzy matching that understands abbreviations (e.g., "gth" matches "Graph Theory")
- **Syntax Highlighting** - Beautiful LaTeX syntax highlighting in preview pane
- **User-Defined Aliases** - Create custom shortcuts for frequently-used commands or complex workflows
- **Scrollable Previews** - View full note content in dashboard with smooth scrolling

### Search & Discovery

- **Full-Text Search** - Quick grep-based search across all notes
- **Smart Organization** - Tag-based filtering and date-based organization
- **Statistics** - Insights into your note-taking patterns
- **Beautiful UI** - Clean terminal interface with syntax highlighting

## Installation

### Prerequisites

- Go 1.25.4 or higher
- LaTeX distribution with `latexmk` (for building notes)
- `git` (optional, for sync features)

### From Source

```bash
git clone https://github.com/yourusername/lx-cli.git
cd lx-cli
make build
# or
go build -o lx
```

### Using Go Install

```bash
go install github.com/yourusername/lx-cli@latest
```

## Quick Start

1. **Initialize the vault:**

    ```bash
    lx init
    ```

    This creates the vault structure at `~/.local/share/lx/`:
    - `notes/` - Your LaTeX source files
    - `templates/` - Your .sty template files
    - `cache/` - Compiled PDFs and build artifacts
    - `assets/` - Static assets (images, bibliographies, etc.)

2. **Create your first note:**

    ```bash
    lx new "My First Note"
    ```

3. **List all notes:**

    ```bash
    lx list
    ```

4. **Build a note to PDF:**

    ```bash
    lx build "my first note"
    ```

5. **Open a note (Smart Entry):**

    ```bash
    lx "my first note"    # Smart entry - opens the note
    # or the traditional way:
    lx open "my first note"
    ```

6. **Launch the Interactive Dashboard:**
    ```bash
    lx                    # No arguments launches dashboard
    ```

## Smart Entry

LX features intelligent command resolution. Instead of typing explicit commands, just provide the note name:

```bash
# These all work:
lx physics              # Opens physics note (default action)
lx "graph theory"       # Opens graph theory note
lx homework assignment  # Searches for "homework assignment"

# Traditional commands still work:
lx open physics
lx edit physics
```

The default action (`open` or `edit`) can be configured in `config.yaml`:

```yaml
default_action: open # or "edit"
```

## Interactive Dashboard

Press `lx` with no arguments to launch a full-screen interactive dashboard:

```
┌─ LX Notes Dashboard ─────────────────────────────────────────────┐
│ Vault: ~/notes                           15 notes │ Modified     │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  >  Calculus III Homework      [calc] [homework]         2h ago  │
│     Linear Algebra Notes       [math]                    1d ago  │
│     Quantum Mechanics          [physics]                 3d ago  │
│                                                                   │
├──────────────────────────────────────────────────────────────────┤
│ [↑↓/jk] Navigate  [Enter/o] Open  [e] Edit  [/] Search  [q] Quit│
└──────────────────────────────────────────────────────────────────┘
```

### Dashboard Features

- **Fuzzy Search** - Press `/` to search with intelligent fuzzy matching
- **Live Preview** - See note content with syntax highlighting in split-screen
- **Scrollable Preview** - Use PgUp/PgDn to scroll through note content
- **Quick Actions** - Open, edit, build, delete notes with single keystrokes
- **Graph View** - Toggle graph visualization with `v`
- **Always-On Preview** - Preview pane automatically updates as you navigate

### Dashboard Hotkeys

| Key           | Action                          |
| ------------- | ------------------------------- |
| `↑`/`k`       | Move up                         |
| `↓`/`j`       | Move down                       |
| `g`           | Jump to top                     |
| `G`           | Jump to bottom                  |
| `Enter`/`o`   | Open note (PDF)                 |
| `e`           | Edit note source                |
| `b`           | Build note                      |
| `d`           | Delete note (with confirmation) |
| `n`           | Create new note                 |
| `/`           | Search/filter (fuzzy matching)  |
| `v`           | Toggle graph view               |
| `PgUp`/`PgDn` | Scroll preview pane             |
| `?`           | Show help                       |
| `Esc`         | Exit search/cancel              |
| `q`           | Quit dashboard                  |

## Commands

### Core Commands

- `lx new <title>` - Create a new note
- `lx list` - List all notes in table format
- `lx open <query>` - Open a note's PDF
- `lx edit <query>` - Edit a note in your default editor
- `lx delete <query>` - Delete a note
- `lx rename <query> <new-title>` - Rename a note

### Building

- `lx build <query>` - Build a specific note to PDF
- `lx build-all` - Build all notes in parallel
- `lx watch <query>` - Watch a note and rebuild on changes
- `lx clean` - Remove all build artifacts

### Search & Discovery

- `lx grep <pattern>` - Search note contents
- `lx explore` - Interactively browse and search notes
- `lx stats` - View vault statistics
- `lx daily` - Create or open today's daily note

### Graph & Links

- `lx graph` - Interactive graph browser
- `lx graph <query>` - Start graph from a specific note
- `lx graph --dot` - Export graph in DOT format
- `lx links <query>` - Show all links for a note
- `lx reindex` - Rebuild the knowledge graph index

### Templates

- `lx new template <name>` - Create a new template
- `lx list --template` - List all templates
- `lx new --template <name> <title>` - Create note from template

### Tags

- `lx tag add <query> <tag>` - Add a tag to a note
- `lx tag remove <query> <tag>` - Remove a tag from a note
- `lx list --tag <tag>` - Filter notes by tag

### Git Integration

- `lx git <command>` - Run git commands in vault
- `lx clone <url>` - Clone a vault from a repository
- `lx sync` - Pull, commit, and push changes

### Assets & Attachments

- `lx attach <query> <file>` - Attach a file (image, PDF, etc.) to a note
- `lx export <query>` - Export note and its assets

### Utilities

- `lx config` - View configuration
- `lx doctor` - Run health checks on the vault
- `lx todo` - List all TODO items across notes
- `lx version` - Show version information
- `lx dashboard` (or `lx dash`) - Launch interactive dashboard

### Aliases

Create custom shortcuts for frequently-used commands:

- `lx alias list` - List all defined aliases
- `lx alias add <name> <command>` - Create a new alias
- `lx alias remove <name>` - Remove an alias

**Examples:**

```bash
# Create shortcuts
lx alias add hw "new -t homework"
lx alias add today "list -s modified -r"
lx alias add backup "export -f json -o ~/backup.json"

# Use them
lx hw assignment3              # Creates homework note
lx today                       # Lists recent notes
lx backup                      # Exports to JSON

# Advanced: Variable substitution
lx alias add note "new -t $1 -n '$2'"
lx note math "calculus"        # Expands to: lx new -t math -n 'calculus'
```

See [Alias Documentation](docs/ALIASES.md) for complete guide.

## Configuration

Configuration is stored at `~/.config/lx/config.yaml`.

A sample configuration file with all available options is provided in `config.sample.yaml`. Copy it to your config directory and customize as needed:

```bash
cp config.sample.yaml ~/.config/lx/config.yaml
```

### Basic Configuration

```yaml
# Default template to use when creating new notes
default_template: ""

# Default editor (uses $EDITOR environment variable if not set)
editor: ""

# Number of concurrent jobs for build-all
max_workers: 4

# Default action for smart entry (open or edit)
default_action: open

# User-defined command aliases
aliases:
    hw: new -t homework
    today: list -s modified -r
    backup: export -f json -o ~/backup.json
```

See `config.sample.yaml` for the complete list of configuration options including:

- LaTeX compiler settings
- Git integration options
- Graph visualization settings
- UI customization
- Search preferences
- Export settings
- And more

## Vault Structure

```
~/.local/share/lx/
├── notes/              # Your LaTeX notes (.tex files)
│   ├── 20240115-my-first-note.tex
│   ├── 20240116-graph-theory.tex
│   └── .latexmkrc     # LaTeX build configuration
├── templates/         # Style files (.sty)
│   ├── article.sty
│   └── notes.sty
├── cache/             # Build artifacts
│   ├── 20240115-my-first-note.pdf
│   ├── index.json     # Knowledge graph index
│   └── *.aux, *.log   # LaTeX temporary files
└── assets/            # Static files
    ├── images/
    └── bibliography/
```

## Note Format

Notes are created with metadata:

```latex
%% title: Graph Theory Basics
%% date: 2024-01-15
%% tags: [math, graph-theory]

\documentclass{article}
\usepackage{notes}  % Your custom template

\begin{document}

\section{Introduction}

Graph theory is the study of graphs...

% Link to another note
\input{../notes/discrete-math.tex}

\end{document}
```

## Fuzzy Search

LX features intelligent fuzzy search that understands:

- **Abbreviations** - "gth" matches "Graph Theory"
- **Word boundaries** - "gtb" matches "Graph Theory Basics"
- **Non-consecutive characters** - "tpa" matches "Topology and Algebra"
- **Case-insensitive** - Works regardless of capitalization
- **Relevance ranking** - Best matches appear first

Available in:

- Dashboard search (press `/`)
- Smart entry
- All search commands

## Knowledge Graph

The graph feature analyzes links between notes and provides:

- Interactive terminal-based graph browser
- Backlink tracking (which notes link to this one)
- Outgoing link tracking (which notes this one links to)
- DOT format export for visualization with Graphviz

```bash
# Interactive graph browser
lx graph

# Start from a specific note
lx graph "graph theory"

# Export to image
lx graph --dot > graph.dot
dot -Tpng graph.dot -o graph.png
```

## Workflow Examples

### Daily Workflow with Dashboard

```bash
# Launch dashboard
lx

# Press '/' to search, type "homework"
# Press Enter to open
# Press 'e' to edit
# Press 'b' to build
# Press 'q' to quit
```

### Quick Note Creation with Aliases

```bash
# Define your workflow
lx alias add qp "new -t quick-note -e"
lx alias add hw "new -t homework"
lx alias add today "daily"

# Use shortcuts
lx qp idea           # Quick note + auto-edit
lx hw assignment3    # Homework note
lx today             # Open today's note
```

### Smart Entry Workflow

```bash
# Just type what you want
lx physics           # Opens physics note (smart!)
lx graph theory      # Opens graph theory note
lx "homework 3"      # Searches and opens

# Configure default action
# Edit ~/.config/lx/config.yaml:
# default_action: edit    # Now 'lx physics' edits instead
```

## Environment Variables

- `EDITOR` - Default text editor for `lx edit`
- `XDG_DATA_HOME` - Override vault location (default: `~/.local/share`)
- `XDG_CONFIG_HOME` - Override config location (default: `~/.config`)

## Important Files

- `LICENSE` - MIT License file
- `README.md` - This file
- `config.sample.yaml` - Sample configuration file with all available options
- `docs/ALIASES.md` - Complete alias feature documentation
- `.latexmkrc` - LaTeX compilation settings (auto-generated in notes directory)
- `.gitignore` - Git ignore patterns (auto-generated in vault root)
- `~/.config/lx/config.yaml` - User configuration file
- `~/.local/share/lx/cache/index.json` - Knowledge graph index

## Development

### Project Structure

```
lx-cli/
├── cmd/                    # CLI commands
├── internal/
│   ├── adapters/          # External adapters (compiler, repos)
│   ├── core/
│   │   ├── domain/        # Domain models
│   │   ├── ports/         # Interfaces
│   │   └── services/      # Business logic
├── pkg/
│   ├── ui/                # Terminal UI components
│   └── vault/             # Vault management
└── main.go
```

### Build

```bash
# Build binary
make build

# Run tests
make test

# Install locally
make install

# Clean build artifacts
make clean
```

## Contributing

Note: This is a personal project that I created for my own use. You’re welcome to fork the repository and modify it however you like, but I won’t be accepting pull requests. I’m sharing the project publicly in case it’s useful to others.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Chroma](https://github.com/alecthomas/chroma) - Syntax highlighting
- [go-fuzzyfinder](https://github.com/ktr0731/go-fuzzyfinder) - Fuzzy finding

---

**LX** - Manage your LaTeX notes like a pro
