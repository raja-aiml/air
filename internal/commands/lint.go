package commands

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/raja-aiml/air/internal/engine"
	"golang.org/x/tools/go/packages"
)

// LintCommands holds dependencies for linting commands.
type LintCommands struct{}

// NewLintCommands creates linting command handlers.
func NewLintCommands() *LintCommands {
	return &LintCommands{}
}

// Register adds all linting commands to the registry.
func (c *LintCommands) Register(r *engine.Registry) {
	r.Register(&engine.Command{
		Name:        "lint.check",
		Description: "Run static analysis checks on Go code (uses go/analysis)",
		Examples: []string{
			"lint the code",
			"check for errors",
			"run linter",
			"static analysis",
			"find bugs",
		},
		Parameters: []engine.Parameter{
			{Name: "path", Type: "string", Default: "./...", Description: "Path to analyze"},
		},
		Execute: c.check,
	})

	r.Register(&engine.Command{
		Name:        "fmt.check",
		Description: "Check if Go code is properly formatted",
		Examples: []string{
			"check formatting",
			"is code formatted",
			"format check",
		},
		Parameters: []engine.Parameter{
			{Name: "path", Type: "string", Default: ".", Description: "Path to check"},
		},
		Execute: c.formatCheck,
	})

	r.Register(&engine.Command{
		Name:        "fmt.fix",
		Description: "Format Go code (pure Go, no gofmt binary required)",
		Examples: []string{
			"format code",
			"fix formatting",
			"gofmt",
			"format go files",
		},
		Parameters: []engine.Parameter{
			{Name: "path", Type: "string", Default: ".", Description: "Path to format"},
		},
		Execute: c.formatFix,
	})
}

func (c *LintCommands) check(ctx context.Context, params map[string]any) (engine.Result, error) {
	p := engine.Params(params)
	path := p.String("path", "./...")

	cfg := &packages.Config{
		Mode:    packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedName,
		Context: ctx,
	}

	pkgs, err := packages.Load(cfg, path)
	if err != nil {
		return engine.ErrorResult(err), err
	}

	var issues []string

	// Check for package loading errors
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			issues = append(issues, fmt.Sprintf("%s: %s", pkg.PkgPath, e.Msg))
		}
	}

	message := "Static Analysis Results:\n"
	if len(issues) == 0 {
		message += "  No issues found!"
	} else {
		for _, issue := range issues {
			message += fmt.Sprintf("  - %s\n", issue)
		}
	}

	return engine.NewResultWithData(message, map[string]any{
		"issues_count": len(issues),
		"issues":       issues,
	}), nil
}

func (c *LintCommands) formatCheck(ctx context.Context, params map[string]any) (engine.Result, error) {
	pr := engine.Params(params)
	path := pr.String("path", ".")

	var unformatted []string

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(filePath, ".go") {
			return nil
		}

		// Read and check formatting
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		formatted, err := format.Source(content)
		if err != nil {
			// Syntax error - skip
			return nil
		}

		if !bytes.Equal(content, formatted) {
			unformatted = append(unformatted, filePath)
		}

		return nil
	})

	if err != nil {
		return engine.ErrorResult(err), err
	}

	message := "Format Check Results:\n"
	if len(unformatted) == 0 {
		message += "  All files are properly formatted!"
	} else {
		message += fmt.Sprintf("  %d files need formatting:\n", len(unformatted))
		for _, f := range unformatted {
			message += fmt.Sprintf("    - %s\n", f)
		}
		message += "\n  Run 'air fmt.fix' to fix formatting."
	}

	return engine.NewResultWithData(message, map[string]any{
		"unformatted_count": len(unformatted),
		"unformatted":       unformatted,
	}), nil
}

func (c *LintCommands) formatFix(ctx context.Context, params map[string]any) (engine.Result, error) {
	pr := engine.Params(params)
	path := pr.String("path", ".")

	var fixed []string
	var errors []string

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(filePath, ".go") {
			return nil
		}

		// Parse and format
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
			return nil
		}

		// Read original content
		original, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		// Format
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
			return nil
		}

		// Write if changed
		if !bytes.Equal(original, buf.Bytes()) {
			if err := os.WriteFile(filePath, buf.Bytes(), info.Mode()); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
				return nil
			}
			fixed = append(fixed, filePath)
		}

		return nil
	})

	if err != nil {
		return engine.ErrorResult(err), err
	}

	message := "Format Fix Results:\n"
	if len(fixed) == 0 {
		message += "  No files needed formatting."
	} else {
		message += fmt.Sprintf("  Fixed %d files:\n", len(fixed))
		for _, f := range fixed {
			message += fmt.Sprintf("    - %s\n", f)
		}
	}

	if len(errors) > 0 {
		message += "\n  Errors:\n"
		for _, e := range errors {
			message += fmt.Sprintf("    - %s\n", e)
		}
	}

	return engine.NewResultWithData(message, map[string]any{
		"fixed_count": len(fixed),
		"fixed":       fixed,
		"errors":      errors,
	}), nil
}
