package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/oak/pkg/api"
	"github.com/go-go-golems/oak/pkg/tree-sitter"
	"github.com/pkg/errors"
)

// Ensure TSParser implements the Parser interface
var _ Parser = &TSParser{}

// TSParser implements the Parser interface using Tree-sitter
type TSParser struct {
	config parserConfig
}

// NewParser creates a new TypeScript/JavaScript parser with the given options
func NewParser(opts ...ParserOption) *TSParser {
	config := parserConfig{
		IncludePrivate:    false,
		IncludeUnexported: false,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &TSParser{
		config: config,
	}
}

// ParseFiles parses the given files and returns function information
func (p *TSParser) ParseFiles(files []string) ([]FunctionInfo, error) {
	return p.parse(api.WithFiles(files))
}

// ParseGlob parses files matching the given glob pattern and returns function information
func (p *TSParser) ParseGlob(pattern string) ([]FunctionInfo, error) {
	return p.parse(api.WithGlob(pattern))
}

// parse is the internal implementation for parsing files using the given run option
func (p *TSParser) parse(runOption api.RunOption) ([]FunctionInfo, error) {
	// Create a query builder for TypeScript/JavaScript
	query := api.NewQueryBuilder(
		api.WithLanguage("typescript"), // Works for both TS and JS

		// Capture regular function declarations
		api.WithQuery("functionDeclarations", `
			(function_declaration
				name: (identifier) @name
				parameters: (formal_parameters) @parameters
				return_type: (_)? @returnType
			) @function
		`),

		// Capture exported function declarations - note the statement structure is different
		api.WithQuery("exportedFunctionDeclarations", `
			(export_statement 
				declaration: (function_declaration
					name: (identifier) @name
					parameters: (formal_parameters) @parameters
					return_type: (_)? @returnType
				)
			) @function
		`),

		// Capture arrow functions as variables
		api.WithQuery("arrowFunctions", `
			(lexical_declaration
				(variable_declarator
				name: (identifier) @name
				value: (arrow_function
					parameters: (formal_parameters) @parameters
					return_type: (_)? @returnType
				)) @function)
		`),

		// Capture exported arrow functions
		api.WithQuery("exportedArrowFunctions", `
			(export_statement
				declaration: (lexical_declaration
					(variable_declarator
						name: (identifier) @name
						value: (arrow_function
							parameters: (formal_parameters) @parameters
							return_type: (_)? @returnType
						)
					)
				)
			) @function
		`),

		// Capture class declarations
		api.WithQuery("classDeclarations", `
			(class_declaration
				name: (type_identifier) @name
			) @class
		`),

		// Capture exported class declarations
		api.WithQuery("exportedClassDeclarations", `
			(export_statement
				declaration: (class_declaration
					name: (type_identifier) @name
				)
			) @class
		`),

		// Capture method definitions in classes
		api.WithQuery("methodDefinitions", `
			(method_definition
				name: (property_identifier) @name
				parameters: (formal_parameters) @parameters
				return_type: (_)? @returnType
			) @function
		`),

		// Capture constructor definitions
		api.WithQuery("constructorDefinitions", `
			(method_definition
				name: (property_identifier) @name
				parameters: (formal_parameters) @parameters
			) @function
		`),

		// Capture comments for documentation
		api.WithQuery("comments", `
			(comment) @comment
		`),

		// Debug query to see all export statements
		api.WithQuery("allExports", `
			(export_statement) @export
		`),
	)

	// Process the results
	functionsResult, err := query.RunWithProcessor(
		context.Background(),
		func(results api.QueryResults) (any, error) {
			var allFunctions []FunctionInfo
			fileComments := make(map[string][]tree_sitter.Capture)

			// Debug: Print the raw exports found in each file
			for fileName, fileResults := range results {
				if exportResults, ok := fileResults["allExports"]; ok {
					fmt.Printf("DEBUG: Found %d exports in %s\n", len(exportResults.Matches), fileName)
					for i, match := range exportResults.Matches {
						fmt.Printf("DEBUG: Export %d: %s\n", i, match["export"].Text)
					}
				}
			}

			// First, collect all comments to match with functions
			for fileName, fileResults := range results {
				if commentResults, ok := fileResults["comments"]; ok {
					for _, match := range commentResults.Matches {
						fileComments[fileName] = append(fileComments[fileName], match["comment"])
					}
				}
			}

			// Create a map to track seen functions by location to avoid duplicates
			seenFunctions := make(map[string]bool)

			// Process all function types
			for fileName, fileResults := range results {
				// Process function declarations
				processFunctions := func(queryName string) {
					if funcResults, ok := fileResults[queryName]; ok {
						fmt.Printf("DEBUG: Found %d matches for query %s in %s\n", len(funcResults.Matches), queryName, fileName)

						for _, match := range funcResults.Matches {
							fnName := match["name"].Text
							fnStartRow := match["function"].StartPoint.Row
							fnStartCol := match["function"].StartPoint.Column

							// Skip invalid function names
							if strings.TrimSpace(fnName) == "" {
								continue
							}

							// Create a unique ID for this function based on its location
							// to avoid duplicates from multiple queries
							functionID := fmt.Sprintf("%s:%d:%d", fileName, fnStartRow, fnStartCol)
							if _, exists := seenFunctions[functionID]; exists {
								// Skip this function as we've already processed it
								continue
							}
							seenFunctions[functionID] = true

							// Find docstring comment
							docstring := p.findDocComment(fileComments, fileName, fnStartRow)

							// Extract return type if available
							returnType := ""
							if rtCapture, ok := match["returnType"]; ok && rtCapture.Text != "" {
								returnType = rtCapture.Text
								// Clean up return type (remove colon prefix if present)
								returnType = strings.TrimPrefix(returnType, ":")
								returnType = strings.TrimSpace(returnType)
							}

							// Extract parameters
							params := p.extractParams(match["parameters"].Text)

							// Determine if exported based on query type or name capitalization
							isExported := p.isExported(fnName) ||
								strings.HasPrefix(queryName, "exported") ||
								queryName == "exportedFunctionDeclarations" ||
								queryName == "exportedArrowFunctions" ||
								queryName == "exportedClassDeclarations"

							// Filter based on config
							if !isExported && !p.config.IncludeUnexported {
								continue
							}

							fmt.Printf("DEBUG: Adding function %s (exported: %v) from query %s\n", fnName, isExported, queryName)

							allFunctions = append(allFunctions, FunctionInfo{
								Name:       fnName,
								Docstring:  docstring,
								Params:     params,
								ReturnType: returnType,
								SourceFile: fileName,
								LineNumber: int(fnStartRow) + 1,
								IsExported: isExported,
							})
						}
					}
				}

				// Process all function types - order is important for correctly detecting exports
				processFunctions("exportedFunctionDeclarations") // Process exported functions first
				processFunctions("exportedArrowFunctions")
				processFunctions("functionDeclarations")
				processFunctions("arrowFunctions")
				processFunctions("methodDefinitions")
				processFunctions("constructorDefinitions")

				// For now, manually process class declarations
				// Map to keep track of classes and their methods
				classMap := make(map[string]*ClassInfo)

				processClassDeclarations := func(queryName string) {
					if classResults, ok := fileResults[queryName]; ok {
						fmt.Printf("DEBUG: Found %d classes for query %s in %s\n", len(classResults.Matches), queryName, fileName)
						for _, match := range classResults.Matches {
							className := match["name"].Text
							classStartRow := match["class"].StartPoint.Row

							// Find docstring comment for the class
							docstring := p.findDocComment(fileComments, fileName, classStartRow)

							// Determine if exported based on query name or class name capitalization
							isExported := p.isExported(className) ||
								strings.HasPrefix(queryName, "exported") ||
								queryName == "exportedClassDeclarations"

							// Create class info
							classInfo := &ClassInfo{
								Name:       className,
								Docstring:  docstring,
								SourceFile: fileName,
								LineNumber: int(classStartRow) + 1,
								IsExported: isExported,
							}

							// Store class info for later use when processing methods
							classMap[className] = classInfo

							fmt.Printf("DEBUG: Added class %s (exported: %v)\n", className, isExported)

							// If class methods should be included, add the class methods to the function list
							if p.config.IncludeClassMethods {
								// Find methods related to this class
								// This is a simplistic approach - in a more complete implementation,
								// we would parse the class body and extract methods directly
								for _, fn := range allFunctions {
									// Check if function name matches format: className.methodName
									if strings.HasPrefix(fn.Name, className+".") {
										methodName := strings.TrimPrefix(fn.Name, className+".")
										fn.Name = methodName
										fn.ClassName = className
										fn.IsMethod = true

										classInfo.Methods = append(classInfo.Methods, fn)

										fmt.Printf("DEBUG: Added method %s to class %s\n", methodName, className)
									}
								}
							}
						}
					}
				}

				processClassDeclarations("classDeclarations")
				processClassDeclarations("exportedClassDeclarations")

				// Process methods now that we have the class information
				processClassMethods := func() {
					if p.config.IncludeClassMethods && len(classMap) > 0 {
						// Scan for method definitions
						for _, results := range []string{"methodDefinitions", "constructorDefinitions"} {
							if methodResults, ok := fileResults[results]; ok {
								for _, match := range methodResults.Matches {
									methodName := match["name"].Text
									methodStartRow := match["function"].StartPoint.Row
									methodStartCol := match["function"].StartPoint.Column

									// Create a unique ID for this method
									methodID := fmt.Sprintf("%s:%d:%d", fileName, methodStartRow, methodStartCol)
									if _, exists := seenFunctions[methodID]; exists {
										// Skip this method as we've already processed it
										continue
									}
									seenFunctions[methodID] = true

									// Find docstring comment
									docstring := p.findDocComment(fileComments, fileName, methodStartRow)

									// Extract return type if available
									returnType := ""
									if rtCapture, ok := match["returnType"]; ok && rtCapture.Text != "" {
										returnType = rtCapture.Text
										// Clean up return type
										returnType = strings.TrimPrefix(returnType, ":")
										returnType = strings.TrimSpace(returnType)
									}

									// Extract parameters
									params := p.extractParams(match["parameters"].Text)

									// Try to determine which class this method belongs to
									// This is a simplistic approach - in a real implementation,
									// we would analyze the class body to find methods
									var className string
									for cName := range classMap {
										// The method definition is after the class declaration
										if methodStartRow > uint32(classMap[cName].LineNumber) {
											// And before the next class declaration or end of file
											// (this is a simplification)
											className = cName
											break
										}
									}

									if className != "" {
										// Add method to function list
										methodFn := FunctionInfo{
											Name:       methodName,
											Docstring:  docstring,
											Params:     params,
											ReturnType: returnType,
											SourceFile: fileName,
											LineNumber: int(methodStartRow) + 1,
											IsExported: classMap[className].IsExported, // Methods inherit export status from class
											IsMethod:   true,
											ClassName:  className,
										}

										allFunctions = append(allFunctions, methodFn)
										fmt.Printf("DEBUG: Added method %s to class %s\n", methodName, className)
									}
								}
							}
						}
					}
				}

				processClassMethods()
			}

			return allFunctions, nil
		},
		runOption,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query code")
	}

	// Type assertion for the result
	functions, ok := functionsResult.([]FunctionInfo)
	if !ok {
		return nil, errors.New("could not convert result to []FunctionInfo")
	}

	// Final deduplication by location
	seenLocations := make(map[string]int)
	var uniqueFunctions []FunctionInfo
	for _, fn := range functions {
		// Create a location key from file + line number + name
		locationKey := fmt.Sprintf("%s:%d:%s", fn.SourceFile, fn.LineNumber, fn.Name)
		if _, exists := seenLocations[locationKey]; !exists {
			uniqueFunctions = append(uniqueFunctions, fn)
			seenLocations[locationKey] = len(uniqueFunctions) - 1
		}
	}

	return uniqueFunctions, nil
}

// Helper to find the nearest comment above a function
func (p *TSParser) findDocComment(fileComments map[string][]tree_sitter.Capture, fileName string, fnStartRow uint32) string {
	var nearestComment tree_sitter.Capture
	nearestDistance := uint32(10) // Only look for comments within 10 lines

	for _, comment := range fileComments[fileName] {
		if comment.EndPoint.Row < fnStartRow &&
			fnStartRow-comment.EndPoint.Row <= nearestDistance {
			nearestDistance = fnStartRow - comment.EndPoint.Row
			nearestComment = comment
		}
	}

	if nearestDistance <= 3 { // Only use comments within 3 lines
		// Clean the comment text
		text := nearestComment.Text
		text = strings.TrimSpace(text)

		// Remove comment markers
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")

		// Process JSDoc style comments
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
			lines[i] = strings.TrimPrefix(lines[i], "*")
			lines[i] = strings.TrimSpace(lines[i])
		}

		// Convert JSDoc @param and @returns tags to markdown
		text = strings.Join(lines, "\n")
		text = strings.TrimSpace(text)

		// Convert @param tags to markdown list items
		text = strings.ReplaceAll(text, "@param ", "- **")
		text = strings.ReplaceAll(text, "@returns ", "**Returns:** ")

		// Add closing bold tags for parameter names
		lines = strings.Split(text, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "- **") {
				// Extract parameter name (first word after the prefix)
				parts := strings.SplitN(line[4:], " ", 2) // Skip the "- **" prefix
				if len(parts) == 2 {
					lines[i] = "- **" + parts[0] + ":** " + parts[1]
				}
			}
		}

		return strings.Join(lines, "\n")
	}
	return ""
}

// Helper function to extract parameter info
func (p *TSParser) extractParams(paramsText string) []ParameterInfo {
	var params []ParameterInfo

	// Remove parentheses and split by comma
	cleanParams := strings.Trim(paramsText, "()")
	if cleanParams == "" {
		return params
	}

	// Handle object types with nested commas
	// This is a simplified approach - a proper parser would be better
	var paramItems []string
	braceLevel := 0
	currentParam := ""

	for i := 0; i < len(cleanParams); i++ {
		ch := cleanParams[i]
		switch ch {
		case '{', '(':
			braceLevel++
			currentParam += string(ch)
		case '}', ')':
			braceLevel--
			currentParam += string(ch)
		case ',':
			if braceLevel > 0 {
				currentParam += string(ch)
			} else {
				paramItems = append(paramItems, currentParam)
				currentParam = ""
			}
		default:
			currentParam += string(ch)
		}
	}

	// Add the last parameter
	if currentParam != "" {
		paramItems = append(paramItems, currentParam)
	}

	for _, param := range paramItems {
		param = strings.TrimSpace(param)

		// Split on the first colon to separate name and type
		parts := strings.SplitN(param, ":", 2)

		pInfo := ParameterInfo{Name: parts[0]}
		if len(parts) > 1 {
			pInfo.Type = strings.TrimSpace(parts[1])
		}
		params = append(params, pInfo)
	}
	return params
}

// isExported checks if a function/method name is exported (starts with uppercase)
func (p *TSParser) isExported(name string) bool {
	if len(name) == 0 {
		return false
	}

	firstChar := name[0]
	return firstChar >= 'A' && firstChar <= 'Z'
}
