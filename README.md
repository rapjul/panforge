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
# Generate a sample Markdown file with YAML frontmatter
panforge init --markdown

# OR generate a default config file
panforge init --config

# Generate a sample Markdown file to convert to specific output formats
panforge init -m -t pdf,docx
```

### Running Conversions

```bash
panforge [flags] <file>
```

This acts as a transparent wrapper around `pandoc`, reading configuration from the YAML header of `input.md` to determine how to process it.

### Passing Arguments to Pandoc

`panforge` generally passes unknown arguments through to `pandoc`. However, since `panforge` uses some flags (like `-f`/`--force`) that conflict with `pandoc`'s flags (e.g., `-f`/`--from`), strict flag parsing may consume them.

To safely pass flags directly to `pandoc` without interference, use the `--` separator:

```bash
# Correctly pass -f/--from to pandoc
panforge input.md -- -f markdown

# Pass arbitrary flags
panforge input.md -- --toc --toc-depth=2
```

### Command Line Flags

- `-t, --target <format>`: Specifically target one or more output formats defined in the YAML header. Can be used multiple times.
- `-o, --output <file>`: Override the output filename.
- `-a, --all`: Process all formats defined in the YAML header (this is also the default behavior if no targets are specified).
- `-f, --force`: Force overwrite of existing output files without prompting.
- `-d, --dry-run`: Print the `pandoc` commands that would be executed without running them.
- `-v, --verbose`: Enable verbose logging.
- `-q, --quiet`: Suppress standard output messages.
- `-w, --watch`: Watch input file for changes and automatically re-run.
- `--log <file>`: Append logs to the specified file.

To pass arguments directly to the underlying `pandoc` command (not recommended, use the YAML header instead), you can append them after the input file or arguments.

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

To load completions for every session, correct the above commands to write to your shell's completion directory or config file (e.g., `~/.bashrc` or `~/.zshrc`).



## Configuration

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

Options at the root of the YAML header are treated as variables or metadata by `panforge` if they match known configuration keys, otherwise they are passed to pandoc as metadata.

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

