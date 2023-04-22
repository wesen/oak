package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
	"io"
	"text/template"
)

type OakCommand struct {
	Language       string
	Queries        []Query
	Template       string
	SitterLanguage *sitter.Language
}

type Query struct {
	// Name of the resulting variable after parsing
	Name string
	// Query contains the tree-sitter query that will be applied to the source code
	Query string
}

type Capture struct {
	// Name if the capture name from the query
	Name string
	// Text is the actual text that was captured
	Text string
}

type Match map[string]Capture

type Result struct {
	// Name is the name of the query
	Name string
	// Matches are the matches for the query
	Matches []Match
}

type QueryResults map[string]*Result

type OakCommandOption func(*OakCommand)

func WithLanguage(lang string) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.Language = lang
	}
}

func WithSitterLanguage(lang *sitter.Language) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.SitterLanguage = lang
	}
}

func WithQueries(queries ...Query) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.Queries = append(cmd.Queries, queries...)
	}
}

func WithTemplate(template string) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.Template = template
	}
}

func NewOakCommandFromReader(r io.Reader, options ...OakCommandOption) (*OakCommand, error) {
	var cmd OakCommand
	err := yaml.NewDecoder(r).Decode(&cmd)
	if err != nil {
		return nil, err
	}

	for _, option := range options {
		option(&cmd)
	}
	return &cmd, nil
}

func NewOakCommand(options ...OakCommandOption) *OakCommand {
	cmd := OakCommand{}
	for _, option := range options {
		option(&cmd)
	}
	return &cmd
}

func (cmd *OakCommand) ExecuteQueries(tree *sitter.Node, sourceCode []byte) (QueryResults, error) {
	if cmd.SitterLanguage == nil {
		lang, err := LanguageNameToSitterLanguage(cmd.Language)
		if err != nil {
			return nil, err
		}
		cmd.SitterLanguage = lang
	}
	results := make(map[string]*Result)
	for _, query := range cmd.Queries {
		matches := []Match{}

		// could parse queries up front and return an error if necessary
		q, err := sitter.NewQuery([]byte(query.Query), cmd.SitterLanguage)
		if err != nil {
			switch err := err.(type) {
			case *sitter.QueryError:
				line := 1
				for i := uint32(0); i < err.Offset; i++ {
					if query.Query[i] == '\n' {
						line++
					}
				}

				return nil, errors.Errorf("error parsing query: %v at line %d", err.Type, line)
			}
			return nil, err
		}
		qc := sitter.NewQueryCursor()
		qc.Exec(q, tree)
		for {
			m, ok := qc.NextMatch()
			if !ok {
				break
			}
			m = qc.FilterPredicates(m, sourceCode)

			match := Match{}
			for _, c := range m.Captures {
				match[q.CaptureNameForId(c.Index)] = Capture{
					Name: q.CaptureNameForId(c.Index),
					Text: c.Node.Content(sourceCode),
				}
			}
			matches = append(matches, match)
		}

		results[query.Name] = &Result{
			Name:    query.Name,
			Matches: matches,
		}
	}

	return results, nil
}

func (cmd *OakCommand) Render(results QueryResults) (string, error) {
	tmpl, err := templating.CreateTemplate("oak").Parse(cmd.Template)
	if err != nil {
		return "", err
	}

	return cmd.RenderWithTemplate(results, err, tmpl)
}

func (cmd *OakCommand) RenderWithTemplate(results QueryResults, err error, tmpl *template.Template) (string, error) {
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, results)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (cmd *OakCommand) RenderWithTemplateFile(results QueryResults, file string) (string, error) {
	tmpl, err := templating.CreateTemplate("oak").ParseFiles(file)
	if err != nil {
		return "", err
	}

	return cmd.RenderWithTemplate(results, err, tmpl)
}

func (cmd *OakCommand) ResultsToJSON(results QueryResults, f io.Writer) error {
	enc := json.NewEncoder(f)
	return enc.Encode(results)
}

func (cmd *OakCommand) ResultsToYAML(results QueryResults, f io.Writer) error {
	enc := yaml.NewEncoder(f)
	return enc.Encode(results)
}

func (cmd *OakCommand) Parse(ctx context.Context, code []byte) (*sitter.Tree, error) {
	if cmd.SitterLanguage == nil {
		lang, err := LanguageNameToSitterLanguage(cmd.Language)
		if err != nil {
			return nil, err
		}

		cmd.SitterLanguage = lang
	}

	parser := sitter.NewParser()
	parser.SetLanguage(cmd.SitterLanguage)
	tree, err := parser.ParseCtx(ctx, nil, code)
	if err != nil {
		return nil, err
	}

	return tree, nil
}
