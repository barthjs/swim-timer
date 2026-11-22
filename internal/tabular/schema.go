// Package tabular imports and exports tables in delimited text files, such as the CSV files of a spreadsheet.
// A Schema describes the columns of a target type. A Mapping assigns the columns and values of a file to them,
// so a file of any layout can be read, and an export writes the columns of the schema back.
// Rows are read one by one, so a file of any size is processed with constant memory.
package tabular

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
)

// defaultValidator checks the values of the columns against their rules, unless WithValidator replaces it.
var defaultValidator = validator.New()

// Column is one column of a schema, built by Field.
type Column[T any] struct {
	key      string
	required bool
	rules    string
	aliases  []string
	choices  []string
	// suggestions are values a mapping may translate into, without restricting the column to them.
	suggestions []string
	// valueAliases are further values of a file the guess translates into a choice or suggestion, by the target.
	valueAliases map[string][]string
	normalize    func(string) string
	// set parses the raw text, checks the value with the validator of the schema, and assigns it to the target.
	set func(ctx context.Context, rules *validator.Validate, target *T, raw string) *FieldError
	// format returns the text of the value of the target, nil for a column that is not exported.
	format func(target *T) string
}

// FieldOptions defines a column whose values have the type V.
type FieldOptions[T, V any] struct {
	// Key names the column in a mapping and in an export.
	Key string
	// Parse turns the text of a value into V. A *FieldError keeps its tag, any other error fails with "invalid".
	Parse func(string) (V, error)
	// ParseContext replaces Parse for a value that needs the context of the import.
	ParseContext func(ctx context.Context, raw string) (V, error)
	Set          func(target *T, value V)
	// Validate checks a parsed value beyond the rules of the column, nil to check only the rules.
	Validate func(ctx context.Context, value V) *FieldError
	// Get and Format return the text of the value for an export, nil for a column that is not exported.
	Get    func(target *T) V
	Format func(value V) string
}

// ColumnOption changes the behavior of a column.
type ColumnOption func(*columnSettings)

// columnSettings holds the behavior the ColumnOption values select.
type columnSettings struct {
	required     bool
	rules        string
	aliases      []string
	choices      []string
	suggestions  []string
	valueAliases map[string][]string
	normalize    func(string) string
}

// Required rejects a row without a value in the column, and a mapping that leaves the column out.
func Required() ColumnOption {
	return func(s *columnSettings) {
		s.required = true
	}
}

// Rules checks the parsed value with the tags of go-playground/validator.
func Rules(tags string) ColumnOption {
	return func(s *columnSettings) {
		s.rules = tags
	}
}

// Aliases names further headers the column is guessed from, besides its key, compared without regard to case.
func Aliases(names ...string) ColumnOption {
	return func(s *columnSettings) {
		s.aliases = names
	}
}

// Choices restricts the column to the given values and lets a mapping translate the values of a file into them.
func Choices(values ...string) ColumnOption {
	return func(s *columnSettings) {
		s.choices = values
	}
}

// Suggestions lets a mapping translate the values of a file into the given values like Choices,
// but accepts any other value as well.
func Suggestions(values ...string) ColumnOption {
	return func(s *columnSettings) {
		s.suggestions = values
	}
}

// ValueAliases names further values of a file, compared without regard to case,
// which the guess of a mapping translates into the choice or suggestion they are listed under.
func ValueAliases(aliases map[string][]string) ColumnOption {
	return func(s *columnSettings) {
		s.valueAliases = aliases
	}
}

// Normalize prepares the text of a value before it is checked and parsed.
func Normalize(f func(string) string) ColumnOption {
	return func(s *columnSettings) {
		s.normalize = f
	}
}

// Field builds a column of the target type T whose values have the type V.
func Field[T, V any](opts FieldOptions[T, V], options ...ColumnOption) Column[T] {
	var settings columnSettings
	for _, option := range options {
		option(&settings)
	}

	column := Column[T]{
		key:          opts.Key,
		required:     settings.required,
		rules:        settings.rules,
		aliases:      settings.aliases,
		choices:      settings.choices,
		suggestions:  settings.suggestions,
		valueAliases: settings.valueAliases,
		normalize:    settings.normalize,
	}
	column.set = func(ctx context.Context, rules *validator.Validate, target *T, raw string) *FieldError {
		value, err := parseValue(ctx, opts, raw)
		if err != nil {
			return toFieldError(err, raw)
		}
		if column.rules != "" {
			if failed := checkRules(rules, value, column.rules); failed != nil {
				failed.Value = raw
				return failed
			}
		}
		if opts.Validate != nil {
			if failed := opts.Validate(ctx, value); failed != nil {
				failed.Value = raw
				return failed
			}
		}
		opts.Set(target, value)
		return nil
	}
	if opts.Get != nil && opts.Format != nil {
		column.format = func(target *T) string {
			return opts.Format(opts.Get(target))
		}
	}
	return column
}

// parseValue parses the text of a value with ParseContext, or with Parse without one.
func parseValue[T, V any](ctx context.Context, opts FieldOptions[T, V], raw string) (V, error) {
	if opts.ParseContext != nil {
		return opts.ParseContext(ctx, raw)
	}
	return opts.Parse(raw)
}

// toFieldError keeps the tag of a *FieldError and reports any other parse error as "invalid".
func toFieldError(err error, raw string) *FieldError {
	if failed, ok := errors.AsType[*FieldError](err); ok {
		result := *failed
		result.Value = raw
		return &result
	}
	return &FieldError{
		Tag:   "invalid",
		Value: raw,
	}
}

// checkRules checks a value against the tags of go-playground/validator and returns the first failed rule.
func checkRules(rules *validator.Validate, value any, tags string) *FieldError {
	err := rules.Var(value, tags)
	if err == nil {
		return nil
	}
	if failures, ok := errors.AsType[validator.ValidationErrors](err); ok && len(failures) > 0 {
		return &FieldError{
			Tag:   failures[0].Tag(),
			Param: failures[0].Param(),
		}
	}
	return &FieldError{
		Tag: "invalid",
	}
}

// ColumnInfo describes a column for the mapping of a file.
type ColumnInfo struct {
	Key      string
	Required bool
	// Choices are the values the column accepts, empty for a column that accepts any text.
	Choices []string
	// Suggestions are values a mapping may translate into, while the column accepts any text.
	Suggestions []string
}

// Schema describes the columns of the target type T and the hooks a row passes through.
type Schema[T any] struct {
	columns  []Column[T]
	rules    *validator.Validate
	prepare  func(ctx context.Context, values map[string]string) FieldErrors
	resolve  func(ctx context.Context, values map[string]string) (*T, error)
	validate func(ctx context.Context, target *T) FieldErrors
}

// SchemaOption adds a hook or a setting to a schema.
type SchemaOption[T any] func(*Schema[T])

// WithPrepare changes the values of a row by key after the mapping and before the columns parse them,
// and the failures it returns skip the row.
func WithPrepare[T any](f func(ctx context.Context, values map[string]string) FieldErrors) SchemaOption[T] {
	return func(s *Schema[T]) {
		s.prepare = f
	}
}

// WithResolve returns the target a row is read into instead of a new one, and an error ends the rows.
func WithResolve[T any](f func(ctx context.Context, values map[string]string) (*T, error)) SchemaOption[T] {
	return func(s *Schema[T]) {
		s.resolve = f
	}
}

// WithValidate checks a target after every column is set, and the failures it returns skip the row.
func WithValidate[T any](f func(ctx context.Context, target *T) FieldErrors) SchemaOption[T] {
	return func(s *Schema[T]) {
		s.validate = f
	}
}

// WithValidator checks the Rules of the columns with the given validator instead of one with the built-in rules only.
func WithValidator[T any](rules *validator.Validate) SchemaOption[T] {
	return func(s *Schema[T]) {
		s.rules = rules
	}
}

// New creates a schema of the given columns.
func New[T any](columns []Column[T], options ...SchemaOption[T]) *Schema[T] {
	schema := &Schema[T]{
		columns: columns,
		rules:   defaultValidator,
	}
	for _, option := range options {
		option(schema)
	}
	return schema
}

// Columns describes every column of the schema in its order.
func (s *Schema[T]) Columns() []ColumnInfo {
	infos := make([]ColumnInfo, 0, len(s.columns))
	for _, column := range s.columns {
		infos = append(infos, ColumnInfo{
			Key:         column.key,
			Required:    column.required,
			Choices:     slices.Clone(column.choices),
			Suggestions: slices.Clone(column.suggestions),
		})
	}
	return infos
}

// Mapping assigns the columns and values of a file to the columns of a schema.
type Mapping struct {
	Format Format
	// Columns holds the header of the file a column is read from by the key of the column.
	// A column without an entry stays empty.
	Columns map[string]string
	// Values translates the values of a file into the choices of a column, by the key of the column
	// and the value in the file. A value without an entry is kept.
	Values map[string]map[string]string
}

// Identity returns the mapping of a file whose header names every column by its key.
func (s *Schema[T]) Identity(format Format) Mapping {
	columns := make(map[string]string, len(s.columns))
	for _, column := range s.columns {
		columns[column.key] = column.key
	}
	return Mapping{
		Format:  format,
		Columns: columns,
	}
}

// guessMapping assigns every column the first header that equals its key or one of its aliases without regard to case,
// and translates every value of a column that equals one of its choices or suggestions,
// or one of their value aliases, without regard to case.
func (s *Schema[T]) guessMapping(format Format, header []string, values map[string][]string) Mapping {
	mapping := Mapping{
		Format:  format,
		Columns: make(map[string]string),
		Values:  make(map[string]map[string]string),
	}
	for _, column := range s.columns {
		names := append([]string{column.key}, column.aliases...)
		source, ok := findHeader(header, names)
		if !ok {
			continue
		}
		mapping.Columns[column.key] = source
		targets := slices.Concat(column.choices, column.suggestions)
		if len(targets) == 0 {
			continue
		}

		translated := make(map[string]string)
		for _, value := range values[source] {
			if target, ok := guessValue(value, targets, column.valueAliases); ok {
				translated[value] = target
			}
		}
		if len(translated) > 0 {
			mapping.Values[column.key] = translated
		}
	}
	return mapping
}

// guessValue returns the target a value of a file equals, or one whose value aliases it equals, without regard to case.
func guessValue(value string, targets []string, aliases map[string][]string) (string, bool) {
	for _, target := range targets {
		if strings.EqualFold(value, target) {
			return target, true
		}
	}
	for _, target := range targets {
		for _, alias := range aliases[target] {
			if strings.EqualFold(value, alias) {
				return target, true
			}
		}
	}
	return "", false
}

// findHeader returns the first name of the header that equals one of names without regard to case.
func findHeader(header, names []string) (string, bool) {
	for _, name := range names {
		for _, source := range header {
			if source != "" && strings.EqualFold(strings.TrimSpace(source), name) {
				return source, true
			}
		}
	}
	return "", false
}
