package i18n_test

import (
	"bytes"
	htmltemplate "html/template"
	"testing"
	texttemplate "text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/i18n"
)

// blankSource exercises output actions at every level the rewrite walks into.
const blankSource = `[{{.Name}}][{{.User.Name}}][{{if .On}}on{{else}}{{.Off}}{{end}}]` +
	`[{{range .Items}}{{.}}{{else}}{{.Empty}}{{end}}][{{with .User}}{{.Name}}{{else}}{{.None}}{{end}}]` +
	`[{{$name := .Name}}{{$name}}][{{.Pointer}}][{{.Set}}]`

// blankData is the template data for the missing value tests.
type blankData struct {
	Name    any
	User    any
	On      bool
	Off     any
	Items   []string
	Empty   any
	None    any
	Pointer *int
	Set     string
}

func TestBlankMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data any
		want string
	}{
		{
			name: "blanks missing map keys",
			data: map[string]any{"Set": "x"},
			want: "[][][][][][][][x]",
		},
		// A field of a nil struct field still fails, as text/template cannot tell it from a typo,
		// so User is an empty map here.
		{
			name: "blanks nil values and pointers",
			data: blankData{
				User: map[string]any{},
				Set:  "x",
			},
			want: "[][][][][][][][x]",
		},
		{
			name: "blanks nil data",
			data: nil,
			want: "[][][][][][][][]",
		},
		{
			name: "keeps present values",
			data: map[string]any{
				"Name":  "Ada",
				"User":  map[string]string{"Name": "Bob"},
				"Items": []int{1, 2},
				"Set":   "x",
			},
			want: "[Ada][Bob][][12][Bob][Ada][][x]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl := texttemplate.Must(texttemplate.New("t").Funcs(i18n.TemplateFuncs()).Parse(blankSource))

			for _, each := range tmpl.Templates() {
				i18n.BlankMissing(each.Tree)
			}

			var buf bytes.Buffer
			require.NoError(t, tmpl.Execute(&buf, tt.data))
			assert.Equal(t, tt.want, buf.String())
		})
	}

	t.Run("keeps the escaping of html templates", func(t *testing.T) {
		t.Parallel()

		tmpl := htmltemplate.Must(
			htmltemplate.New("t").Funcs(i18n.TemplateFuncs()).Parse(blankSource + `[{{.Markup}}]`),
		)

		for _, each := range tmpl.Templates() {
			i18n.BlankMissing(each.Tree)
		}

		var buf bytes.Buffer
		data := map[string]any{"Name": "<b>", "Markup": htmltemplate.HTML("<i>ok</i>"), "Set": "x"}
		require.NoError(t, tmpl.Execute(&buf, data))
		want := "[&lt;b&gt;][][][][][&lt;b&gt;][][x][<i>ok</i>]"
		assert.Equal(t, want, buf.String(), "values stay escaped, trusted HTML stays raw")
	})

	t.Run("blanks missing values in defined templates", func(t *testing.T) {
		t.Parallel()

		source := `{{define "inner"}}<{{.Name}}>{{end}}{{template "inner" .}}`
		tmpl := texttemplate.Must(texttemplate.New("t").Funcs(i18n.TemplateFuncs()).Parse(source))

		for _, each := range tmpl.Templates() {
			i18n.BlankMissing(each.Tree)
		}

		var buf bytes.Buffer
		require.NoError(t, tmpl.Execute(&buf, map[string]any{}))
		assert.Equal(t, "<>", buf.String())
	})

	t.Run("ignores a nil tree", func(t *testing.T) {
		t.Parallel()

		blank := func() { i18n.BlankMissing(nil) }

		assert.NotPanics(t, blank)
	})
}
