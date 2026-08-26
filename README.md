# Panforge (`panforge`)

`panforge` is a from scratch Go port of the [Ruby `panrun` script](https://github.com/mb21/panrun). It is a wrapper around `pandoc` that allows you to specify compile commands (like output formats and Pandoc arguments) directly in the YAML header of your Markdown files.



## Installation

### Prerequisites

- [Pandoc](https://pandoc.org/) must be installed and available in your `PATH`.
- [Go](https://go.dev/) (1.25+ recommended) for building from source.

### Install with `go install`

```bash
go install github.com/rapjul/panforge/cmd/panforge@latest
```

### Build from Source

```bash
git clone https://github.com/rapjul/panforge.git
cd panforge
go install ./cmd/panforge
```



## Usage

### Quick Start (`init`)

To quickly get started with a new file or configuration:

```bash
# Generate a sample Markdown file with YAML frontmatter (default: input.md)
panforge init --markdown

# OR
panforge init -m

# Generate a sample Markdown file with a custom filename
panforge init -m chapter1.md

# OR generate a default config file in the current directory (default: .panforge.yaml)
panforge init --config

# Generate a sample Markdown file configured for specific output formats
panforge init -m -t pdf,docx
```

### Running Conversions

```bash
# Convert a single file
panforge input.md

# Batch convert multiple files
panforge doc1.md doc2.md
panforge *.md

# Specify input files explicitly via flags
panforge -i doc1.md -i doc2.md
```

This reads configuration from the YAML front-matter of each Markdown file to determine how to process it.

### Passing Arguments to Pandoc

To safely pass arbitrary flags directly to `pandoc` without interference, use the POSIX `--` separator:

```bash
# Pass input format directly to pandoc
panforge input.md -- -f markdown

# Pass arbitrary flags directly to pandoc
panforge input.md -- --toc --toc-depth=2 --shift-heading-level-by=1

# Batch convert multiple files with passthrough flags
panforge *.md -- --toc
```

### Command Line Flags

- `-t, --to <format>`, `--target <format>`: Specifically target one or more output formats defined in the YAML header. Can be used multiple times.
- `-i, --input <file>`: Explicitly specify one or more input files.
- `-o, --output <file>`: Override the output filename (supports templates such as `"{title}.{ext}"`).
- `-a, --all`: Process all formats defined in the YAML header (this is also the default behavior if no targets are specified).
- `-F, --force`: Force overwrite of existing output files without prompting.
- `-n, --dry-run`: Print the `pandoc` commands that would be executed without running them.
- `-v, --verbose`: Enable verbose logging.
- `-q, --quiet`: Suppress standard output messages.
- `-w, --watch`: Watch input file and configuration for changes and automatically re-run (implies `--force`).
- `-c, --concurrency <num>`: Limit number of concurrent Pandoc processes (default: number of CPUs).
- `-r, --relative-output`: Resolve relative output paths against CWD instead of input file directory.
- `--log <file>`: Append logs to the specified file.
- `--json`: Output logs in JSON format.

### Shell Completion

`panforge` supports shell completion for Bash, Zsh, Fish, and PowerShell. This includes dynamic completion for output formats and input files.

To generate the completion script:

#### Bash

```sh
source <(panforge completion bash)
```

#### Zsh

```sh
source <(panforge completion zsh)
```

#### Fish

```sh
panforge completion fish | source
```

To load completions for every session, write the output to your shell's completion directory or config file (e.g., `~/.bashrc` or `~/.zshrc`).



## Configuration

### Configuration Paths

`panforge` looks for configuration files in the following order:

1.  **Project Level**: `./.panforge.yaml`, `./panforge.yaml`, `./.panforge.yml`, or `./panforge.yml` in the current working directory.
2.  **XDG Specification**: `$XDG_CONFIG_HOME/panforge/default.yaml` (e.g., `~/.config/panforge/default.yaml` on Linux/macOS).
3.  **Windows**: `%APPDATA%/panforge/default.yaml`.
4.  **Default**: `~/.config/panforge/default.yaml` (if `XDG_CONFIG_HOME` is unset).

You can place your default configuration file in any of these locations to customize fallback templates and Pandoc options across documents.

### Metadata Configuration

`panforge` looks for strictly structured metadata in the YAML header of your Markdown file.

### Multiple Outputs

You can define a list of formats to generate using the `outputs` key, or a map of configurations using the `output` key.

#### Using `output` Map (Recommended)

Allows specifying per-format options.

```yaml
---
title: My Document
output:
  html:
    to: html5
    standalone: true
    css: style.css
  pdf:
    pdf-engine: xelatex
    variable:
      geometry: margin=2cm
---
```

Running `panforge file.md` on the above will generate both an HTML and a PDF file.

#### Using `outputs` List

Simple list of formats.

```yaml
---
outputs:
  - html
  - docx
---
```

### Pandoc Arguments

Any key inside an output block is translated to a Pandoc argument.

The following rules apply:

- `key: value` -> `--key=value`
- `key: true` -> `--key`
- `key: [list]` -> `--key=item1 --key=item2 ...`
- `key: {map}` -> (varies, usually not directly mapped to simple flags, but `variables` and `metadata` are special cases)

### Global Options

Options at the root of the YAML header are treated as variables or metadata by `panforge` if they match known configuration keys, otherwise they are passed to Pandoc as metadata.

Special keys processed by `panforge`:

- `output` / `outputs`: Defines targets.
- `filename-template`: (Optional) Template for output filenames (e.g., `"{title}_{date}.{ext}"`).
    - Supported template variables include:
        - `{date}` and `{time}` (formatted as `YYYY-MM-DD` and `HH:MM:SS`, respectively)
        - `{title}` and `{title-slug}` (if `title` is a string)
        - `{author}` and `{author-slug}` (if `author` is a string)
        - `{ext}` (file extension)
    - If `slugify-filename` is enabled, `{title}` and `{author}` will be slugified any time they are used (e.g., `my-title` instead of `my title`)
- `slugify-filename`: (Optional) Boolean to enable/disable filename slugification (default: `false`).



## Maintenance

This project is designed for **Minimal Maintenance**.

- **Dependencies**: All Go dependencies are vendored in the `vendor/` directory. This ensures the project can always be built even if upstream repositories disappear or make incompatible changes.
- **CI/CD**: Workflows are pinned to specific versions (Go 1.25, GoReleaser v2) to prevent "bit rot" where CI breaks simply because a tool updated.
- **Versioning**: The project uses semantic versioning (e.g., `v1.2.3`).
- **Branches**: The project uses a single branch (`main`) for development and releases.

### Updating Dependencies

If you need to update dependencies:

1.  Run `go get -u ./...` (or update specific packages).
2.  Run `go mod tidy`.
3.  Run `go mod vendor` to update the `vendor/` directory.
4.  Commit changes.

### Local Quality Checks (Lefthook)

This project uses [Lefthook](https://github.com/evilmartians/lefthook) for fast, Go-native git hooks.

1.  **Install**:

    ```bash
    go install github.com/evilmartians/lefthook@latest
    go install github.com/google/yamlfmt/cmd/yamlfmt@latest
    ```

2.  **Setup**: Run `lefthook install` in the repo root.
3.  **Run Manually**: `lefthook run pre-commit`.



## Contributing

For development instructions, please see [CONTRIBUTING.md](CONTRIBUTING.md).
