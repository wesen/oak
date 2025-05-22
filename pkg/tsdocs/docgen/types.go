package docgen

import (
	"github.com/go-go-golems/oak/pkg/tsdocs/parser"
)

// Formatter defines an interface for formatting function documentation
type Formatter interface {
	// FormatFunctions formats a slice of FunctionInfo into a documentation string
	FormatFunctions(functions []parser.FunctionInfo, title string) (string, error)
}

// FormatterOption defines functional options for configuring formatters
type FormatterOption func(*formatterConfig)

// formatterConfig holds configuration for documentation formatters
type formatterConfig struct {
	IncludeTableOfContents bool
	IncludeSourceLocation  bool
	GroupByFile            bool
}

// WithTableOfContents configures the formatter to include a table of contents
func WithTableOfContents(include bool) FormatterOption {
	return func(c *formatterConfig) {
		c.IncludeTableOfContents = include
	}
}

// WithSourceLocation configures the formatter to include source file locations
func WithSourceLocation(include bool) FormatterOption {
	return func(c *formatterConfig) {
		c.IncludeSourceLocation = include
	}
}

// WithGroupByFile configures the formatter to group functions by their source file
func WithGroupByFile(group bool) FormatterOption {
	return func(c *formatterConfig) {
		c.GroupByFile = group
	}
}
