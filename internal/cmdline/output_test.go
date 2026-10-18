package cmdline_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/cmdline"
)

// item is a sample record for the printer tests.
type item struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

func TestAddOutputFlag(t *testing.T) {
	t.Parallel()

	t.Run("keeps the pre run of the command", func(t *testing.T) {
		t.Parallel()

		root, stdout, _ := newTestRoot(t)
		cmd := &cobra.Command{
			Use: "list",
		}
		preRan := false
		cmd.PreRunE = func(*cobra.Command, []string) error {
			preRan = true
			return nil
		}
		format := cmdline.AddOutputFlag(cmd, cmdline.OutputTable, cmdline.OutputJSON)
		cmd.RunE = func(cmd *cobra.Command, _ []string) error {
			return cmdline.NewPrinter(cmd, *format).Print([]item{
				{
					Name: "x",
				},
			}, table([]item{
				{
					Name: "x",
				},
			}))
		}
		root.AddCommand(cmd)

		code := cmdline.Execute(root, []string{"list", "-o", "json"})

		require.Equal(t, 0, code)
		assert.True(t, preRan)
		assert.JSONEq(t, `[{"name":"x","count":0}]`, stdout.String())
	})
}

func TestPrinter_Print(t *testing.T) {
	t.Parallel()

	items := []item{
		{
			Name:  "alpha",
			Count: 1,
		},
		{
			Name:  "beta",
			Count: 22,
		},
	}

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{
			name:   "prints json",
			format: cmdline.OutputJSON,
			want:   "[\n  {\n    \"name\": \"alpha\",\n    \"count\": 1\n  },\n  {\n    \"name\": \"beta\",\n    \"count\": 22\n  }\n]\n",
		},
		{
			name:   "prints yaml",
			format: cmdline.OutputYAML,
			want:   "- name: alpha\n  count: 1\n- name: beta\n  count: 22\n",
		},
		{
			name:   "prints a table",
			format: cmdline.OutputTable,
			want:   "NAME   COUNT\nalpha  1\nbeta   22\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer

			err := cmdline.Printer{
				Out:    &out,
				Format: tt.format,
			}.Print(items, table(items))

			require.NoError(t, err)
			assert.Equal(t, tt.want, out.String())
		})
	}

	t.Run("rejects a table without a table function", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := cmdline.Printer{
			Out:    &out,
			Format: cmdline.OutputTable,
		}.Print(1, nil)

		assert.Error(t, err)
	})

	t.Run("returns the error of the table function", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		failing := func(io.Writer) error { return errors.New("broken") }

		err := cmdline.Printer{
			Out:    &out,
			Format: cmdline.OutputTable,
		}.Print(1, failing)

		assert.EqualError(t, err, "broken")
	})

	t.Run("rejects a value that json cannot encode", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := cmdline.Printer{
			Out:    &out,
			Format: cmdline.OutputJSON,
		}.Print(make(chan int), nil)

		assert.Error(t, err)
	})

	t.Run("panics for a value that yaml cannot encode", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		encode := func() {
			_ = cmdline.Printer{
				Out:    &out,
				Format: cmdline.OutputYAML,
			}.Print(make(chan int), nil)
		}

		assert.Panics(t, encode)
	})
}

// table returns a function that writes the items as a tab-separated table.
func table(items []item) func(w io.Writer) error {
	return func(w io.Writer) error {
		_, _ = fmt.Fprintln(w, "NAME\tCOUNT")
		for _, i := range items {
			_, _ = fmt.Fprintf(w, "%s\t%d\n", i.Name, i.Count)
		}
		return nil
	}
}
