package parser

// FunctionInfo represents information about a function or method in TypeScript/JavaScript
type FunctionInfo struct {
	Name       string
	Docstring  string
	Params     []ParameterInfo
	ReturnType string
	SourceFile string
	LineNumber int
	IsExported bool
	IsMethod   bool
	ClassName  string // If this is a method, what class it belongs to
}

// ParameterInfo represents information about a function parameter
type ParameterInfo struct {
	Name string
	Type string
}

// ClassInfo represents information about a class
type ClassInfo struct {
	Name       string
	Docstring  string
	Methods    []FunctionInfo
	Properties []PropertyInfo
	SourceFile string
	LineNumber int
	IsExported bool
}

// PropertyInfo represents information about a class property
type PropertyInfo struct {
	Name       string
	Type       string
	Docstring  string
	IsPrivate  bool
	IsReadonly bool
}

// Parser defines the interface for parsing TypeScript/JavaScript files
type Parser interface {
	// ParseFiles parses the given files and returns a collection of function information
	ParseFiles(files []string) ([]FunctionInfo, error)

	// ParseGlob parses files matching the given glob pattern and returns function information
	ParseGlob(pattern string) ([]FunctionInfo, error)
}

// ParserOption defines functional options for configuring the parser
type ParserOption func(*parserConfig)

// parserConfig holds configuration for the parser
type parserConfig struct {
	IncludePrivate      bool
	IncludeUnexported   bool
	IncludeClassMethods bool
}

// WithPrivate configures the parser to include private functions/methods
func WithPrivate(include bool) ParserOption {
	return func(c *parserConfig) {
		c.IncludePrivate = include
	}
}

// WithUnexported configures the parser to include unexported functions/methods
func WithUnexported(include bool) ParserOption {
	return func(c *parserConfig) {
		c.IncludeUnexported = include
	}
}

// WithClassMethods configures the parser to include class methods
func WithClassMethods(include bool) ParserOption {
	return func(c *parserConfig) {
		c.IncludeClassMethods = include
	}
}
