package docgen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/oak/pkg/tsdocs/parser"
	"github.com/pkg/errors"
)

// Ensure MarkdownFormatter implements the Formatter interface
var _ Formatter = &MarkdownFormatter{}

// MarkdownFormatter implements the Formatter interface for Markdown output
type MarkdownFormatter struct {
	config formatterConfig
}

// NewMarkdownFormatter creates a new Markdown formatter with the given options
func NewMarkdownFormatter(opts ...FormatterOption) *MarkdownFormatter {
	config := formatterConfig{
		IncludeTableOfContents: true,
		IncludeSourceLocation:  true,
		GroupByFile:            true,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &MarkdownFormatter{
		config: config,
	}
}

// FormatFunctions formats a slice of FunctionInfo into Markdown
func (f *MarkdownFormatter) FormatFunctions(functions []parser.FunctionInfo, title string) (string, error) {
	if len(functions) == 0 {
		return "", errors.New("no functions to document")
	}

	output := &strings.Builder{}
	// Write header
	fmt.Fprintf(output, "# %s API Reference\n\n", title)

	// Group functions by file if configured
	if f.config.GroupByFile {
		// Create a map of file to functions
		fileMap := make(map[string][]parser.FunctionInfo)
		for _, fn := range functions {
			fileMap[fn.SourceFile] = append(fileMap[fn.SourceFile], fn)
		}

		// Generate table of contents if configured
		if f.config.IncludeTableOfContents {
			f.writeTableOfContents(output, fileMap, title)
		}

		// Generate function documentation for each file
		for file, fileFunctions := range fileMap {
			relPath := filepath.Base(file)

			// File header
			fmt.Fprintf(output, "## %s\n\n", relPath)

			// Document each function
			for _, fn := range fileFunctions {
				f.writeFunction(output, fn, relPath)
			}
		}
	} else {
		// Not grouped by file, just list all functions

		// Generate table of contents if configured
		if f.config.IncludeTableOfContents {
			f.writeSimpleTableOfContents(output, functions)
		}

		// Document each function
		for _, fn := range functions {
			relPath := ""
			if f.config.IncludeSourceLocation {
				relPath = filepath.Base(fn.SourceFile)
			}
			f.writeFunction(output, fn, relPath)
		}
	}

	return output.String(), nil
}

// writeTableOfContents writes a table of contents grouped by file
func (f *MarkdownFormatter) writeTableOfContents(output *strings.Builder, fileMap map[string][]parser.FunctionInfo, basePath string) {
	fmt.Fprintf(output, "## Table of Contents\n\n")

	for file, fileFunctions := range fileMap {
		relPath := filepath.Base(file)

		// Create section heading
		fmt.Fprintf(output, "- [%s](#%s)\n", relPath, strings.ReplaceAll(relPath, ".", ""))

		// Add function links
		for _, fn := range fileFunctions {
			anchor := strings.ToLower(fn.Name)
			anchor = strings.ReplaceAll(anchor, " ", "-")
			fmt.Fprintf(output, "  - [%s](#%s)\n", fn.Name, anchor)
		}
	}

	fmt.Fprintf(output, "\n")
}

// writeSimpleTableOfContents writes a simple table of contents for all functions
func (f *MarkdownFormatter) writeSimpleTableOfContents(output *strings.Builder, functions []parser.FunctionInfo) {
	fmt.Fprintf(output, "## Table of Contents\n\n")

	for _, fn := range functions {
		anchor := strings.ToLower(fn.Name)
		anchor = strings.ReplaceAll(anchor, " ", "-")
		fmt.Fprintf(output, "- [%s](#%s)\n", fn.Name, anchor)
	}

	fmt.Fprintf(output, "\n")
}

// writeFunction writes documentation for a single function
func (f *MarkdownFormatter) writeFunction(output *strings.Builder, fn parser.FunctionInfo, relPath string) {
	// Function heading - if it's a method, format it as className.methodName
	if fn.IsMethod && fn.ClassName != "" {
		fmt.Fprintf(output, "### %s.%s\n\n", fn.ClassName, fn.Name)
	} else {
		fmt.Fprintf(output, "### %s\n\n", fn.Name)
	}

	// Export status
	if fn.IsExported {
		fmt.Fprintf(output, "*Exported*\n\n")
	}

	// Method type
	if fn.IsMethod {
		fmt.Fprintf(output, "*Method of class %s*\n\n", fn.ClassName)
	}

	// Description from docstring
	if fn.Docstring != "" {
		fmt.Fprintf(output, "%s\n\n", fn.Docstring)
	}

	// Function signature
	fmt.Fprintf(output, "```typescript\n")
	signature := fn.Name + "("

	// Format parameters
	for i, param := range fn.Params {
		if i > 0 {
			signature += ", "
		}
		signature += param.Name
		if param.Type != "" {
			signature += ": " + param.Type
		}
	}
	signature += ")"

	// Add return type if available
	if fn.ReturnType != "" {
		signature += ": " + fn.ReturnType
	}

	fmt.Fprintf(output, "%s\n```\n\n", signature)

	// Parameters section if we have any
	if len(fn.Params) > 0 {
		fmt.Fprintf(output, "**Parameters:**\n\n")
		for _, param := range fn.Params {
			typeInfo := ""
			if param.Type != "" {
				typeInfo = fmt.Sprintf(" - *%s*", param.Type)
			}
			fmt.Fprintf(output, "- `%s`%s\n", param.Name, typeInfo)
		}
		fmt.Fprintf(output, "\n")
	}

	// Return type section if available
	if fn.ReturnType != "" {
		fmt.Fprintf(output, "**Returns:** *%s*\n\n", fn.ReturnType)
	}

	// Source location if configured
	if f.config.IncludeSourceLocation && relPath != "" {
		fmt.Fprintf(output, "*Defined in [%s:%d]*\n\n", relPath, fn.LineNumber)
	}
}
