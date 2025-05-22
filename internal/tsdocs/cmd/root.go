package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/oak/pkg/tsdocs/docgen"
	"github.com/go-go-golems/oak/pkg/tsdocs/parser"
	"github.com/spf13/cobra"
)

var (
	includePrivate      bool
	includeUnexported   bool
	includeClassMethods bool
	noTableOfContents   bool
	noSourceLocation    bool
	noGroupByFile       bool
	outputFormat        string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ts-docs [file/directory]",
	Short: "Generate API documentation for TypeScript/JavaScript files",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		// Create a parser with the configured options
		parserOpts := []parser.ParserOption{}
		if includeUnexported {
			parserOpts = append(parserOpts, parser.WithUnexported(true))
		}
		if includeClassMethods {
			parserOpts = append(parserOpts, parser.WithClassMethods(true))
		}

		tsParser := parser.NewParser(parserOpts...)

		// Parse the source files
		var functions []parser.FunctionInfo
		var err error

		// Check if the path is a file or directory
		fileInfo, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		if fileInfo.IsDir() {
			// Use glob for directory
			functions, err = tsParser.ParseGlob(filepath.Join(path, "**/*.{js,ts,jsx,tsx}"))
		} else {
			// Use specific file
			functions, err = tsParser.ParseFiles([]string{path})
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing files: %s\n", err)
			os.Exit(1)
		}

		// Get the title from the source path
		base := filepath.Base(path)
		extension := filepath.Ext(path)
		title := base
		if extension != "" {
			title = strings.TrimSuffix(base, extension)
		}

		// Create formatter options
		formatterOpts := []docgen.FormatterOption{}
		if noTableOfContents {
			formatterOpts = append(formatterOpts, docgen.WithTableOfContents(false))
		}
		if noSourceLocation {
			formatterOpts = append(formatterOpts, docgen.WithSourceLocation(false))
		}
		if noGroupByFile {
			formatterOpts = append(formatterOpts, docgen.WithGroupByFile(false))
		}

		// Create the appropriate formatter based on the output format
		var formatter docgen.Formatter
		switch outputFormat {
		case "markdown", "md":
			formatter = docgen.NewMarkdownFormatter(formatterOpts...)
		default:
			fmt.Fprintf(os.Stderr, "Unsupported output format: %s\n", outputFormat)
			os.Exit(1)
		}

		// Format the documentation
		output, err := formatter.FormatFunctions(functions, title)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting documentation: %s\n", err)
			os.Exit(1)
		}

		// Output the documentation
		fmt.Println(output)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	// Define flags
	rootCmd.Flags().BoolVar(&includePrivate, "include-private", false, "Include private functions/methods")
	rootCmd.Flags().BoolVar(&includeUnexported, "include-unexported", false, "Include unexported functions")
	rootCmd.Flags().BoolVar(&includeClassMethods, "include-class-methods", true, "Include class methods")
	rootCmd.Flags().BoolVar(&noTableOfContents, "no-toc", false, "Don't include table of contents")
	rootCmd.Flags().BoolVar(&noSourceLocation, "no-source-location", false, "Don't include source file locations")
	rootCmd.Flags().BoolVar(&noGroupByFile, "no-group-by-file", false, "Don't group functions by file")
	rootCmd.Flags().StringVar(&outputFormat, "format", "markdown", "Output format (markdown)")
}
