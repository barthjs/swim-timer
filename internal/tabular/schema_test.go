package tabular

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField(t *testing.T) {
	t.Parallel()

	t.Run("parses and sets the value", func(t *testing.T) {
		t.Parallel()

		column := testColumns()[1]
		var item testItem

		err := column.set(t.Context(), defaultValidator, &item, "7")

		assert.Nil(t, err)
		assert.Equal(t, 7, item.Count)
	})

	t.Run("formats the value of an exported column", func(t *testing.T) {
		t.Parallel()

		column := testColumns()[1]
		item := testItem{
			Count: 12,
		}

		text := column.format(&item)

		assert.Equal(t, "12", text)
	})

	t.Run("keeps the tag of a parse error", func(t *testing.T) {
		t.Parallel()

		column := testColumns()[1]
		var item testItem

		err := column.set(t.Context(), defaultValidator, &item, "x")

		assert.Equal(t, &FieldError{
			Tag:   "number",
			Value: "x",
		}, err)
	})

	t.Run("reports another parse error as invalid", func(t *testing.T) {
		t.Parallel()

		column := Field(FieldOptions[testItem, string]{
			Key:   "name",
			Parse: func(string) (string, error) { return "", strconv.ErrSyntax },
			Set:   func(item *testItem, value string) { item.Name = value },
		})
		var item testItem

		err := column.set(t.Context(), defaultValidator, &item, "x")

		assert.Equal(t, "invalid", err.Tag)
	})

	t.Run("parses the value with the context of the import", func(t *testing.T) {
		t.Parallel()

		type key struct{}
		name := testName(t)
		column := Field(FieldOptions[testItem, string]{
			Key: "name",
			ParseContext: func(ctx context.Context, raw string) (string, error) {
				prefix, _ := ctx.Value(key{}).(string)
				return prefix + raw, nil
			},
			Set: func(item *testItem, value string) { item.Name = value },
		})
		var item testItem

		err := column.set(context.WithValue(t.Context(), key{}, name), defaultValidator, &item, "x")

		assert.Nil(t, err)
		assert.Equal(t, name+"x", item.Name)
	})

	t.Run("checks the rules of the column", func(t *testing.T) {
		t.Parallel()

		column := testColumns()[1]
		var item testItem

		err := column.set(t.Context(), defaultValidator, &item, "-1")

		assert.Equal(t, &FieldError{
			Tag:   "min",
			Param: "0",
			Value: "-1",
		}, err)
		assert.Zero(t, item.Count, "a failed value is not set")
	})

	t.Run("checks the rules with the given validator", func(t *testing.T) {
		t.Parallel()

		rules := validator.New()
		require.NoError(t, rules.RegisterValidation("even", func(fl validator.FieldLevel) bool {
			return fl.Field().Int()%2 == 0
		}))
		column := Field(FieldOptions[testItem, int]{
			Key:   "count",
			Parse: ParseInt,
			Set:   func(item *testItem, value int) { item.Count = value },
		}, Rules("even"))
		var item testItem

		err := column.set(t.Context(), rules, &item, "3")

		assert.Equal(t, &FieldError{
			Tag:   "even",
			Value: "3",
		}, err)
	})

	t.Run("checks the value with the validate hook of the column", func(t *testing.T) {
		t.Parallel()

		column := Field(FieldOptions[testItem, string]{
			Key:   "name",
			Parse: ParseString,
			Set:   func(item *testItem, value string) { item.Name = value },
			Validate: func(_ context.Context, value string) *FieldError {
				if strings.ToUpper(value) != value {
					return &FieldError{
						Tag: "uppercase",
					}
				}
				return nil
			},
		})
		var item testItem

		err := column.set(t.Context(), defaultValidator, &item, "abc")

		assert.Equal(t, &FieldError{
			Tag:   "uppercase",
			Value: "abc",
		}, err)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("keeps the columns and the hooks", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns(), WithValidate(func(context.Context, *testItem) FieldErrors { return nil }))

		assert.Len(t, schema.columns, len(testColumns()))
		assert.NotNil(t, schema.validate)
		assert.Same(t, defaultValidator, schema.rules)
	})

	t.Run("keeps the given validator", func(t *testing.T) {
		t.Parallel()

		rules := validator.New()

		schema := New(testColumns(), WithValidator[testItem](rules))

		assert.Same(t, rules, schema.rules)
	})
}

func TestSchema_Columns(t *testing.T) {
	t.Parallel()

	t.Run("describes every column in its order", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns())

		columns := schema.Columns()

		assert.Equal(t, []ColumnInfo{
			{
				Key:      "name",
				Required: true,
			},
			{
				Key: "count",
			},
			{
				Key: "kind",
				Choices: []string{
					"small",
					"large",
				},
			},
			{
				Key: "active",
				Choices: []string{
					"true",
					"false",
				},
			},
		}, columns)
	})
}

func TestSchema_Identity(t *testing.T) {
	t.Parallel()

	t.Run("maps every column to the header of its key", func(t *testing.T) {
		t.Parallel()

		format := Format{
			Encoding:  EncodingUTF8,
			Separator: ";",
		}

		mapping := New(testColumns()).Identity(format)

		assert.Equal(t, Mapping{
			Format: format,
			Columns: map[string]string{
				"name":   "name",
				"count":  "count",
				"kind":   "kind",
				"active": "active",
			},
		}, mapping)
	})
}

func TestSchema_guessMapping(t *testing.T) {
	t.Parallel()

	t.Run("finds the columns by key and alias and the values by their choices", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns())

		mapping := schema.guessMapping(Format{}, []string{"TITLE", "Kind", "other"}, map[string][]string{
			"Kind": {
				"Small",
				"tiny",
			},
		})

		assert.Equal(t, map[string]string{
			"name": "TITLE",
			"kind": "Kind",
		}, mapping.Columns)
		assert.Equal(t, map[string]map[string]string{
			"kind": {
				"Small": "small",
			},
		}, mapping.Values)
	})

	t.Run("translates the values into the suggestions of a column", func(t *testing.T) {
		t.Parallel()

		schema := New([]Column[testItem]{
			Field(FieldOptions[testItem, string]{
				Key:   "kind",
				Parse: ParseString,
				Set:   func(item *testItem, value string) { item.Kind = value },
			}, Suggestions("small")),
		})

		mapping := schema.guessMapping(Format{}, []string{"kind"}, map[string][]string{
			"kind": {
				"SMALL",
				"huge",
			},
		})

		assert.Equal(t, map[string]map[string]string{
			"kind": {
				"SMALL": "small",
			},
		}, mapping.Values)
	})

	t.Run("translates the value aliases of a choice", func(t *testing.T) {
		t.Parallel()

		schema := New([]Column[testItem]{
			Field(FieldOptions[testItem, string]{
				Key:   "kind",
				Parse: ParseString,
				Set:   func(item *testItem, value string) { item.Kind = value },
			}, Choices("small", "large"), ValueAliases(map[string][]string{
				"small": {
					"S",
					"klein",
				},
			})),
		})

		mapping := schema.guessMapping(Format{}, []string{"kind"}, map[string][]string{
			"kind": {
				"Klein",
				"LARGE",
				"huge",
			},
		})

		assert.Equal(t, map[string]map[string]string{
			"kind": {
				"Klein": "small",
				"LARGE": "large",
			},
		}, mapping.Values)
	})

	t.Run("prefers the key over an alias", func(t *testing.T) {
		t.Parallel()

		mapping := New(testColumns()).guessMapping(Format{}, []string{"title", "name"}, nil)

		assert.Equal(t, "name", mapping.Columns["name"])
	})
}
