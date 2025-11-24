# Publishing air

This document describes how to publish and release the **air** library and CLI tools.

## Prerequisites

1. **GitHub Repository**: Create repository at `https://github.com/raja-aiml/air`
2. **Go Module**: Already configured as `github.com/raja-aiml/air`
3. **GoReleaser**: Installed for automated releases
4. **Git Tags**: For version management

## Initial Setup

### 1. Initialize Git Repository

```bash
cd /Users/rajasoun/workspace/dev/in-progress/ai-runtime
git init
git add .
git commit -m "Initial commit: air - AI Runtime Infrastructure"
```

### 2. Create GitHub Repository

```bash
# Create repository on GitHub at raja-aiml/air
# Then add remote and push
git remote add origin https://github.com/raja-aiml/air.git
git branch -M main
git push -u origin main
```

### 3. Enable GitHub Actions

GitHub Actions workflows are already configured in `.github/workflows/`:
- `ci.yml` - Runs on every push/PR (test, lint, build)
- `release.yml` - Runs on version tags (creates releases)

These will run automatically once you push to GitHub.

## Publishing the Go Library

### Publishing to pkg.go.dev

The library will automatically appear on pkg.go.dev once you:

1. **Push to GitHub**
2. **Create a version tag**

```bash
# Create and push a tag
git tag v0.1.0
git push origin v0.1.0
```

Within a few minutes, the package will be available at:
- `https://pkg.go.dev/github.com/raja-aiml/air`

Users can then install it:

```bash
go get github.com/raja-aiml/air@latest
```

### Using in Projects

```go
import "github.com/raja-aiml/air"

func main() {
    // Use air package
    shutdown := air.InitTracer("my-agent")
    defer shutdown()
    
    pool := air.NewDatabasePool(ctx, dbURL)
    defer pool.Close()
}
```

## Publishing CLI Tools

### Automated Releases with GoReleaser

When you push a version tag, GitHub Actions automatically:

1. **Runs tests** and linting
2. **Builds binaries** for:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)  
   - Windows (amd64, arm64)
3. **Creates GitHub Release** with:
   - Binaries
   - Archives (tar.gz, zip)
   - Checksums
   - Changelog
4. **Publishes to Homebrew** (optional, requires homebrew-tap repo)

### Release Process

```bash
# 1. Ensure all changes are committed
git status

# 2. Update CHANGELOG or version references if needed
# (Optional) Update version in cmd files

# 3. Create and push version tag
git tag -a v0.1.0 -m "Release v0.1.0 - Initial release"
git push origin v0.1.0

# 4. GitHub Actions will automatically:
#    - Run CI checks
#    - Build for all platforms  
#    - Create GitHub release
#    - Upload binaries
```

### Manual Testing of GoReleaser (Optional)

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Test release locally (doesn't publish)
goreleaser release --snapshot --clean

# Check dist/ folder for builds
ls -la dist/
```

## Installation Methods for Users

Once published, users can install via:

### 1. Go Install (Recommended for Go developers)

```bash
# Install CLI tools
go install github.com/raja-aiml/air/cmd/air-dev@latest
go install github.com/raja-aiml/air/cmd/air-verify@latest

# Or specific version
go install github.com/raja-aiml/air/cmd/air-dev@v0.1.0
```

### 2. Homebrew (macOS/Linux)

```bash
# After setting up homebrew-tap
brew tap raja-aiml/tap
brew install air
```

### 3. Direct Binary Download

Users can download from GitHub releases:
- Go to: `https://github.com/raja-aiml/air/releases`
- Download the appropriate archive for their platform
- Extract and add to PATH

### 4. As a Go Library

```bash
# Add to go.mod
go get github.com/raja-aiml/air@latest
```

## Version Management

### Semantic Versioning

Follow semantic versioning (semver):
- `v0.1.0` - Initial release
- `v0.1.1` - Patch (bug fixes)
- `v0.2.0` - Minor (new features, backwards compatible)
- `v1.0.0` - Major (stable API, or breaking changes)

### Pre-releases

For beta/alpha releases:

```bash
git tag v0.1.0-beta.1
git push origin v0.1.0-beta.1
```

Mark as pre-release in GitHub.

## Setting Up Homebrew (Optional)

### 1. Create Homebrew Tap Repository

```bash
# Create a new repo: raja-aiml/homebrew-tap on GitHub
mkdir homebrew-tap
cd homebrew-tap
mkdir Formula
git init
git remote add origin https://github.com/raja-aiml/homebrew-tap.git
```

### 2. GoReleaser Will Auto-Update

When you release, GoReleaser will automatically:
- Generate Homebrew formula
- Push to your tap repository
- Keep it updated with each release

## Continuous Integration

### CI Workflow (on every push/PR)

```yaml
# .github/workflows/ci.yml
- Runs go test
- Runs golangci-lint
- Builds binaries
- Uploads coverage to Codecov
```

### Release Workflow (on version tags)

```yaml
# .github/workflows/release.yml  
- Runs GoReleaser
- Creates GitHub release
- Publishes binaries
- Updates Homebrew tap
```

## Badges for README

Add these badges to README.md:

```markdown
[![Go Version](https://img.shields.io/github/go-mod/go-version/raja-aiml/air)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/raja-aiml/air)](https://goreportcard.com/report/github.com/raja-aiml/air)
[![codecov](https://codecov.io/gh/raja-aiml/air/branch/main/graph/badge.svg)](https://codecov.io/gh/raja-aiml/air)
[![Release](https://img.shields.io/github/v/release/raja-aiml/air)](https://github.com/raja-aiml/air/releases)
```

## Post-Release Checklist

After each release:

- [ ] Verify release appears on GitHub
- [ ] Check binaries download correctly
- [ ] Test installation via `go install`
- [ ] Verify pkg.go.dev updates (may take a few minutes)
- [ ] Update documentation if needed
- [ ] Announce release (Twitter, Reddit, etc.)
- [ ] Monitor issues for bug reports

## Troubleshooting

### Release Failed

```bash
# Check GitHub Actions logs
# Go to: Actions tab in GitHub repo

# Re-run failed jobs if needed
# Or delete tag and recreate:
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0
git tag v0.1.0
git push origin v0.1.0
```

### Package Not Showing on pkg.go.dev

```bash
# Request indexing manually
# Visit: https://pkg.go.dev/github.com/raja-aiml/air
# Click "Request" if needed

# Or wait 10-15 minutes for automatic indexing
```

## Summary

1. **Push to GitHub** → Library available via `go get`
2. **Create version tag** → GoReleaser builds and publishes CLI tools
3. **GitHub Actions** → Automates testing, building, and releasing
4. **Users install** via go install, Homebrew, or direct download

Your library and tools are now production-ready and publicly available!
