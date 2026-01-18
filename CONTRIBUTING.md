# Contributing to Panforge

Thank you for your interest in contributing to Panforge! We welcome contributions from everyone.

## Getting Started

1.  **Fork the repository** on GitHub.
2.  **Clone your fork** locally:

    ```bash
    git clone https://github.com/YOUR_USERNAME/panforge.git
    cd panforge
    ```

3.  **Create a branch** for your feature or bugfix:

    ```bash
    git checkout -b feature/my-new-feature
    ```

## Prerequisites

- [Go](https://go.dev/) (1.25 or later)
- [Pandoc](https://pandoc.org/) (for running integration tests)
- [pre-commit](https://pre-commit.com/) (recommended for local quality checks)

## Development Workflow

### Dependency Management

We use Go modules and vendor our dependencies.

- To add/update dependencies: `go get ...`
- To tidy up: `go mod tidy`
- To update the vendor directory: `go mod vendor`

### Running Tests

Run all unit tests with:

```bash
go test ./...
```

### Local Quality Checks

We use `pre-commit` to ensure code quality.

1.  **Install**: `brew install pre-commit` (or via pip).
2.  **Setup**: Run `pre-commit install` in the repo root.
3.  **Run Manually**: `pre-commit run --all-files`.

## Pull Requests

1.  Ensure your code builds and passes all tests.
2.  Update documentation if you're changing behavior.
3.  Push your branch to GitHub.
4.  Open a Pull Request against the `main` branch.

## Coding Standards

- **Formatting**: We use `gofmt` (handled by `pre-commit`).
- **Linting**: We use `golangci-lint` (handled by `pre-commit`).
- **Documentation**: All exported functions and types must have GoDoc-style comments.

## License

By contributing, you agree that your contributions will be licensed under the project's [LICENSE](LICENSE).
