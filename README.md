# LX - LaTeX Notes Manager

A high-performance, opinionated CLI for managing LaTeX notes. Treat your notes as data while abstracting away file management complexity.

## Features

- **Note Management** - Create, edit, search, and organize LaTeX notes with ease
- **Template System** - Reusable LaTeX style templates for consistent formatting
- **Automated Building** - Compile individual notes or entire vault with latexmk
- **Full-Text Search** - Quick grep-based search across all notes
- **Smart Organization** - Tag-based filtering and date-based organization
- **Knowledge Graph** - Interactive graph visualization of note connections
- **Link Management** - Track backlinks and outgoing connections between notes
- **Statistics** - Insights into your note-taking patterns
- **Beautiful UI** - Clean terminal interface with syntax highlighting
- **Git Integration** - Built-in version control support
- **Sync Support** - Push and pull notes to remote repositories
- **Health Checks** - Doctor command for vault diagnostics

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

5. **Open a note:**
    ```bash
    lx open "my first note"
    ```

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

# Automatic reindexing after note modifications
auto_reindex: true

# Default sort order for list command
default_sort: "date"
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

## Environment Variables

- `EDITOR` - Default text editor for `lx edit`
- `XDG_DATA_HOME` - Override vault location (default: `~/.local/share`)
- `XDG_CONFIG_HOME` - Override config location (default: `~/.config`)

## Important Files

- `LICENSE` - MIT License file
- `README.md` - This file
- `config.sample.yaml` - Sample configuration file with all available options
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

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [go-fuzzyfinder](https://github.com/ktr0731/go-fuzzyfinder) - Fuzzy finding

---

**LX** - Manage your LaTeX notes like a pro
