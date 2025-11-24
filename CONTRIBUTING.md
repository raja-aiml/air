# Contributing to air

Thank you for your interest in contributing to air! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and professional in all interactions.

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Docker and Docker Compose
- Git

### Development Setup

```bash
# Clone the repository
git clone https://github.com/raja-aiml/air.git
cd air

# Install dependencies
make deps

# Install development tools
make tools

# Start infrastructure
make dev-up

# Run tests
make test-all
```

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/raja-aiml/air/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Go version, OS, and any relevant environment details
   - Code samples or error messages

### Suggesting Features

1. Check existing [Issues](https://github.com/raja-aiml/air/issues) and [Discussions](https://github.com/raja-aiml/air/discussions)
2. Create a new issue or discussion with:
   - Clear description of the feature
   - Use cases and benefits
   - Possible implementation approach (optional)

### Pull Requests

1. **Fork the repository** and create a new branch from `main`

```bash
git checkout -b feature/your-feature-name
```

2. **Make your changes**
   - Write clear, documented code
   - Follow existing code style
   - Add tests for new functionality
   - Update documentation as needed

3. **Ensure all checks pass**

```bash
# Format code
make ci-format

# Run linters
make ci-lint

# Run tests
make test-all

# Build binaries
make ci-build
```

4. **Commit your changes**
   - Use clear, descriptive commit messages
   - Follow conventional commits format:
     - `feat: add new feature`
     - `fix: resolve bug`
     - `docs: update documentation`
     - `test: add tests`
     - `refactor: restructure code`
     - `chore: update dependencies`

5. **Push to your fork**

```bash
git push origin feature/your-feature-name
```

6. **Create a Pull Request**
   - Provide a clear description of the changes
   - Reference any related issues
   - Ensure CI checks pass

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Write clear comments and documentation
- Keep functions focused and testable

## Testing

- Write unit tests for new functionality
- Ensure integration tests pass
- Aim for high code coverage
- Test edge cases and error conditions

```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Generate coverage report
make test-coverage
```

## Documentation

- Update README.md for user-facing changes
- Add GoDoc comments for exported functions
- Update examples if APIs change
- Document configuration options

## Release Process

Releases are automated via GitHub Actions:

1. Update version in code if needed
2. Create a new tag: `git tag -a v0.1.0 -m "Release v0.1.0"`
3. Push the tag: `git push origin v0.1.0`
4. GitHub Actions will:
   - Run all tests
   - Build binaries for all platforms
   - Create a GitHub release
   - Publish to package managers

## Questions?

- Open a [Discussion](https://github.com/raja-aiml/air/discussions)
- Ask in issues with the `question` label
- Check existing documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
