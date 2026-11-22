package tabular

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_Read(t *testing.T) {
	t.Parallel()

	t.Run("reads a file whose mapping fits", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns())

		imp, err := schema.Read(strings.NewReader("name;count\r\n"), Mapping{
			Columns: map[string]string{
				"name":  "name",
				"count": "count",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, ";", imp.Format().Separator)
	})

	t.Run("rejects a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		_, err := New(testColumns()).Read(strings.NewReader(""), Mapping{})

		_, ok := errors.AsType[*FileError](err)
		assert.True(t, ok)
	})

	t.Run("rejects a mapping that does not fit", func(t *testing.T) {
		t.Parallel()

		_, err := New(testColumns()).Read(strings.NewReader("count;kind\r\n"), Mapping{
			Columns: map[string]string{
				"count":   "other",
				"kind":    "kind",
				"unknown": "count",
			},
			Values: map[string]map[string]string{
				"kind": {
					"S": "tiny",
					"L": "",
				},
			},
		})

		mappingErr, ok := errors.AsType[*MappingError](err)
		require.True(t, ok)
		assert.Equal(t, FieldErrors{
			"name": {
				{
					Tag: "required",
				},
			},
			"count": {
				{
					Tag:   "missing",
					Value: "other",
				},
			},
			"unknown": {
				{
					Tag: "invalid",
				},
			},
		}, mappingErr.Columns)
		assert.Equal(t, FieldErrors{
			"kind": {
				{
					Tag:   "oneof",
					Param: "small large",
					Value: "S",
				},
			},
		}, mappingErr.Values, "a translation into an empty value is allowed")
	})
}

func TestSchema_ReadByHeader(t *testing.T) {
	t.Parallel()

	t.Run("maps the columns the header names by key or alias", func(t *testing.T) {
		t.Parallel()

		name := testName(t)
		imp, err := New(testColumns()).ReadByHeader(strings.NewReader("Title;COUNT\r\n"+name+";2\r\n"), Format{})
		require.NoError(t, err)

		rows := collectRows(t, imp)

		assert.Equal(t, []Row[testItem]{
			{
				Line: 2,
				Value: &testItem{
					Name:  name,
					Count: 2,
				},
			},
		}, rows)
	})

	t.Run("rejects a header without a required column", func(t *testing.T) {
		t.Parallel()

		_, err := New(testColumns()).ReadByHeader(strings.NewReader("count\r\n"), Format{})

		mappingErr, ok := errors.AsType[*MappingError](err)
		require.True(t, ok)
		assert.Equal(t, FieldErrors{
			"name": {
				{
					Tag: "required",
				},
			},
		}, mappingErr.Columns)
	})

	t.Run("rejects a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		_, err := New(testColumns()).ReadByHeader(strings.NewReader(""), Format{})

		_, ok := errors.AsType[*FileError](err)
		assert.True(t, ok)
	})
}

func TestImport_Header(t *testing.T) {
	t.Parallel()

	t.Run("returns the names of the columns of the file", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("title;size\r\nx;S\r\n"), Mapping{
			Columns: map[string]string{
				"name": "title",
			},
		})
		require.NoError(t, err)

		header := imp.Header()

		assert.Equal(t, []string{"title", "size"}, header)
	})
}

func TestImport_Format(t *testing.T) {
	t.Parallel()

	t.Run("returns the detected format", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("title\tx\r\n"), Mapping{
			Columns: map[string]string{
				"name": "title",
			},
		})
		require.NoError(t, err)

		format := imp.Format()

		assert.Equal(t, "\t", format.Separator)
	})
}

func TestImport_Offset(t *testing.T) {
	t.Parallel()

	t.Run("counts the consumed bytes", func(t *testing.T) {
		t.Parallel()

		content := "name\r\n" + testName(t) + "\r\n"
		imp, err := New(testColumns()).Read(strings.NewReader(content), Mapping{
			Columns: map[string]string{
				"name": "name",
			},
		})
		require.NoError(t, err)

		for _, err := range imp.Rows(t.Context()) {
			require.NoError(t, err)
		}

		assert.Equal(t, int64(len(content)), imp.Offset())
	})
}

func TestImport_Rows(t *testing.T) {
	t.Parallel()

	t.Run("reads the rows through the mapping", func(t *testing.T) {
		t.Parallel()

		name := testName(t)
		imp, err := New(testColumns()).Read(strings.NewReader("Title;Size;Count;On\r\n"+name+";S;4;yes\r\n"), Mapping{
			Columns: map[string]string{
				"name":   "Title",
				"count":  "Count",
				"kind":   "Size",
				"active": "On",
			},
			Values: map[string]map[string]string{
				"kind": {
					"S": "small",
				},
				"active": {
					"yes": "true",
				},
			},
		})
		require.NoError(t, err)

		rows := collectRows(t, imp)

		assert.Equal(t, []Row[testItem]{
			{
				Line: 2,
				Value: &testItem{
					Name:   name,
					Count:  4,
					Kind:   "small",
					Active: true,
				},
			},
		}, rows)
	})

	t.Run("leaves an optional column empty without a value or a mapping", func(t *testing.T) {
		t.Parallel()

		name := testName(t)
		imp, err := New(testColumns()).Read(strings.NewReader("name;count\r\n"+name+";\r\n"), Mapping{
			Columns: map[string]string{
				"name":  "name",
				"count": "count",
			},
		})
		require.NoError(t, err)

		rows := collectRows(t, imp)

		require.Len(t, rows, 1)
		assert.Equal(t, &testItem{
			Name: name,
		}, rows[0].Value)
	})

	t.Run("accepts a value outside the suggestions of a column", func(t *testing.T) {
		t.Parallel()

		columns := testColumns()
		columns[2] = Field(FieldOptions[testItem, string]{
			Key:   "kind",
			Parse: ParseString,
			Set:   func(item *testItem, value string) { item.Kind = value },
		}, Suggestions("small"))
		imp, err := New(columns).Read(strings.NewReader("name;kind\r\nx;S\r\ny;huge\r\n"), Mapping{
			Columns: map[string]string{
				"name": "name",
				"kind": "kind",
			},
			Values: map[string]map[string]string{
				"kind": {
					"S": "small",
				},
			},
		})
		require.NoError(t, err)

		rows := collectRows(t, imp)

		require.Len(t, rows, 2)
		assert.Equal(t, "small", rows[0].Value.Kind)
		assert.Equal(t, "huge", rows[1].Value.Kind)
	})

	t.Run("normalizes a value before it is checked", func(t *testing.T) {
		t.Parallel()

		columns := testColumns()
		columns[1] = Field(FieldOptions[testItem, int]{
			Key:   "count",
			Parse: ParseInt,
			Set:   func(item *testItem, value int) { item.Count = value },
		}, Normalize(func(raw string) string { return strings.TrimPrefix(raw, "#") }))
		imp, err := New(columns).Read(strings.NewReader("name;count\r\nx;#3\r\n"), Mapping{
			Columns: map[string]string{
				"name":  "name",
				"count": "count",
			},
		})
		require.NoError(t, err)

		rows := collectRows(t, imp)

		require.Len(t, rows, 1)
		assert.Equal(t, 3, rows[0].Value.Count)
	})

	t.Run("runs the hooks of the schema", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns(),
			WithPrepare[testItem](func(_ context.Context, values map[string]string) FieldErrors {
				values["name"] = strings.ToUpper(values["name"])
				return nil
			}),
			WithResolve(func(_ context.Context, values map[string]string) (*testItem, error) {
				return &testItem{
					Count: 9,
				}, nil
			}),
			WithValidate(func(_ context.Context, item *testItem) FieldErrors {
				if item.Count > 5 && item.Kind == "small" {
					return FieldErrors{
						"kind": {
							{
								Tag: "small",
							},
						},
					}
				}
				return nil
			}),
		)
		imp, err := schema.Read(strings.NewReader("name;kind\r\nx;large\r\ny;small\r\n"), Mapping{
			Columns: map[string]string{
				"name": "name",
				"kind": "kind",
			},
		})
		require.NoError(t, err)

		rows, failed := collectAll(t, imp)

		require.Len(t, rows, 1)
		assert.Equal(t, "X", rows[0].Value.Name, "the prepared value is parsed")
		assert.Equal(t, 9, rows[0].Value.Count, "the resolved target keeps the value of a column without one")
		require.Len(t, failed, 1)
		assert.Equal(t, 3, failed[0].Line)
		assert.Equal(t, "small", failed[0].Fields["kind"][0].Tag)
	})

	t.Run("skips a row the prepare hook rejects", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns(), WithPrepare[testItem](func(_ context.Context, values map[string]string) FieldErrors {
			if values["name"] == "skip" {
				return FieldErrors{
					RowKey: {
						{
							Tag: "skip",
						},
					},
				}
			}
			return nil
		}))
		imp, err := schema.Read(strings.NewReader("name\r\nskip\r\nkeep\r\n"), Mapping{
			Columns: map[string]string{
				"name": "name",
			},
		})
		require.NoError(t, err)

		rows, failed := collectAll(t, imp)

		require.Len(t, rows, 1)
		assert.Equal(t, "keep", rows[0].Value.Name)
		require.Len(t, failed, 1)
		assert.Equal(t, "skip", failed[0].Fields[RowKey][0].Tag)
	})

	t.Run("reports every failure of a row and reads the rows after it", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns())
		content := "name;count;kind;active\r\n;x;tiny;maybe\r\nok;1;small;true\r\n"
		imp, err := schema.Read(strings.NewReader(content), schema.Identity(Format{}))
		require.NoError(t, err)

		rows, failed := collectAll(t, imp)

		require.Len(t, rows, 1)
		require.Len(t, failed, 1)
		assert.Equal(t, []string{"", "x", "tiny", "maybe"}, failed[0].Values, "the row keeps its values as in the file")
		assert.Equal(t, FieldErrors{
			"name": {
				{
					Tag: "required",
				},
			},
			"count": {
				{
					Tag:   "number",
					Value: "x",
				},
			},
			"kind": {
				{
					Tag:   "oneof",
					Param: "small large",
					Value: "tiny",
				},
			},
			"active": {
				{
					Tag:   "oneof",
					Param: "true false",
					Value: "maybe",
				},
			},
		}, failed[0].Fields)
	})

	t.Run("passes a row that ends early as a failure", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name;count\r\nx\r\n"), Mapping{
			Columns: map[string]string{
				"name": "name",
			},
		})
		require.NoError(t, err)

		_, failed := collectAll(t, imp)

		require.Len(t, failed, 1)
		assert.Equal(t, "short_row", failed[0].Fields[RowKey][0].Tag)
	})

	t.Run("ends with an error of the resolve hook", func(t *testing.T) {
		t.Parallel()

		resolveErr := errors.New("unavailable")
		schema := New(testColumns(), WithResolve(func(context.Context, map[string]string) (*testItem, error) {
			return nil, resolveErr
		}))
		imp, err := schema.Read(strings.NewReader("name\r\nx\r\ny\r\n"), Mapping{
			Columns: map[string]string{
				"name": "name",
			},
		})
		require.NoError(t, err)

		var errs []error
		for _, err := range imp.Rows(t.Context()) {
			errs = append(errs, err)
		}

		assert.Equal(t, []error{resolveErr}, errs)
	})

	t.Run("ends with the context", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name\r\nx\r\n"), Mapping{
			Columns: map[string]string{
				"name": "name",
			},
		})
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		var errs []error
		for _, err := range imp.Rows(ctx) {
			errs = append(errs, err)
		}

		require.Len(t, errs, 1)
		assert.ErrorIs(t, errs[0], context.Canceled)
	})
}

func TestBatches(t *testing.T) {
	t.Parallel()

	t.Run("groups the rows and passes the errors through", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name\r\na\r\nb\r\nc\r\n"), Mapping{
			Columns: map[string]string{
				"name": "name",
			},
		})
		require.NoError(t, err)

		var sizes []int
		for batch, err := range Batches(imp.Rows(t.Context()), 2) {
			require.NoError(t, err)
			sizes = append(sizes, len(batch))
		}

		assert.Equal(t, []int{2, 1}, sizes)
	})

	t.Run("yields an error between the batches", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name;count\r\na;1\r\nb;x\r\nc;2\r\n"), Mapping{
			Columns: map[string]string{
				"name":  "name",
				"count": "count",
			},
		})
		require.NoError(t, err)

		var steps []string
		for batch, err := range Batches(imp.Rows(t.Context()), 5) {
			if err != nil {
				steps = append(steps, "error")
				continue
			}
			steps = append(steps, strings.Repeat("row", len(batch)))
		}

		assert.Equal(t, []string{"error", "rowrow"}, steps)
	})
}

func TestCollect(t *testing.T) {
	t.Parallel()

	t.Run("returns the valid rows and the failed ones", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name;count\r\na;1\r\nb;x\r\n"), Mapping{
			Columns: map[string]string{
				"name":  "name",
				"count": "count",
			},
		})
		require.NoError(t, err)

		rows, failed, err := Collect(imp.Rows(t.Context()), 10)

		require.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Len(t, failed, 1)
	})

	t.Run("stops after the maximum of failed rows", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name;count\r\na;x\r\nb;x\r\nc;1\r\n"), Mapping{
			Columns: map[string]string{
				"name":  "name",
				"count": "count",
			},
		})
		require.NoError(t, err)

		rows, failed, err := Collect(imp.Rows(t.Context()), 2)

		require.NoError(t, err)
		assert.Empty(t, rows)
		assert.Len(t, failed, 2)
	})

	t.Run("returns an error that ends the rows", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name\r\na\r\n\"b\r\n"), Mapping{
			Format: Format{
				Encoding: EncodingUTF8,
			},
			Columns: map[string]string{
				"name": "name",
			},
		}, Lenient())
		require.NoError(t, err)

		_, _, err = Collect(imp.Rows(t.Context()), 10)

		assert.ErrorIs(t, err, ErrUnclosedQuote)
	})
}

// collectRows reads every row of an import and fails the test on an error.
func collectRows(t *testing.T, imp *Import[testItem]) []Row[testItem] {
	t.Helper()

	rows, failed := collectAll(t, imp)
	require.Empty(t, failed)
	return rows
}

// collectAll reads every row of an import with the failed rows and fails the test on any other error.
func collectAll(t *testing.T, imp *Import[testItem]) ([]Row[testItem], []*RowError) {
	t.Helper()

	rows, failed, err := Collect(imp.Rows(t.Context()), 100)
	require.NoError(t, err)
	return slices.Clip(rows), failed
}
