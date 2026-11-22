package tabular

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/apitest"
)

// testItem is the target type of the tests.
type testItem struct {
	Name   string
	Count  int
	Kind   string
	Active bool
}

// testColumns returns the columns of testItem: a required name with an alias, a count that must not be negative,
// a kind with choices, and a flag.
func testColumns() []Column[testItem] {
	return []Column[testItem]{
		Field(FieldOptions[testItem, string]{
			Key:    "name",
			Parse:  ParseString,
			Set:    func(item *testItem, value string) { item.Name = value },
			Get:    func(item *testItem) string { return item.Name },
			Format: func(value string) string { return value },
		}, Required(), Aliases("title")),
		Field(FieldOptions[testItem, int]{
			Key:    "count",
			Parse:  ParseInt,
			Set:    func(item *testItem, value int) { item.Count = value },
			Get:    func(item *testItem) int { return item.Count },
			Format: strconv.Itoa,
		}, Rules("min=0")),
		Field(FieldOptions[testItem, string]{
			Key:    "kind",
			Parse:  ParseString,
			Set:    func(item *testItem, value string) { item.Kind = value },
			Get:    func(item *testItem) string { return item.Kind },
			Format: func(value string) string { return value },
		}, Choices("small", "large")),
		Field(FieldOptions[testItem, bool]{
			Key:   "active",
			Parse: ParseBool,
			Set:   func(item *testItem, value bool) { item.Active = value },
		}, Choices("true", "false")),
	}
}

// testName returns a name for a test item.
func testName(t *testing.T) string {
	t.Helper()

	name := apitest.Faker(t).LetterN(8)
	require.NotEmpty(t, name)
	return name
}

// encodeWindows1252 returns text encoded in Windows-1252.
func encodeWindows1252(t *testing.T, text string) []byte {
	t.Helper()

	encoded, err := charmap.Windows1252.NewEncoder().String(text)
	require.NoError(t, err)
	return []byte(encoded)
}
