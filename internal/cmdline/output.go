package cmdline

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

// The output formats a command can offer with AddOutputFlag.
const (
	OutputText  = "text"
	OutputTable = "table"
	OutputJSON  = "json"
	OutputYAML  = "yaml"
)

// AddOutputFlag adds --output (-o) to cmd with the allowed formats, the first being the default,
// and returns the selected format after parsing. An unknown format fails the command before it runs.
func AddOutputFlag(cmd *cobra.Command, formats ...string) *string {
	format := cmd.Flags().StringP("output", "o", formats[0], "Output format ("+strings.Join(formats, ", ")+")")
	_ = cmd.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(formats, cobra.ShellCompDirectiveNoFileComp))

	previous := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if !slices.Contains(formats, *format) {
			return fmt.Errorf("unknown output format %q, use one of: %s", *format, strings.Join(formats, ", "))
		}
		if previous != nil {
			return previous(cmd, args)
		}
		return nil
	}

	return format
}

// Printer writes a command's result in the selected format.
type Printer struct {
	Out    io.Writer
	Format string
}

// NewPrinter creates a Printer that writes to the command's output.
func NewPrinter(cmd *cobra.Command, format string) Printer {
	return Printer{
		Out:    cmd.OutOrStdout(),
		Format: format,
	}
}

// Print writes v as indented JSON or as YAML. For the table and text formats it calls table with a tab-aligned writer,
// so the command decides which columns a person sees, while JSON and YAML always contain the whole value.
func (p Printer) Print(v any, table func(w io.Writer) error) error {
	switch p.Format {
	case OutputJSON:
		// The encoder ends the value with a newline, and sorted map keys keep the output stable
		return json.MarshalEncode(jsontext.NewEncoder(p.Out, jsontext.WithIndent("  ")), v, json.Deterministic(true), jsonv1.FormatDurationAsNano(true))
	case OutputYAML:
		encoder := yaml.NewEncoder(p.Out)
		encoder.SetIndent(2)
		if err := encoder.Encode(v); err != nil {
			return err
		}
		return encoder.Close()
	default:
		if table == nil {
			return fmt.Errorf("output format %q is not supported", p.Format)
		}
		w := tabwriter.NewWriter(p.Out, 0, 4, 2, ' ', 0)
		if err := table(w); err != nil {
			return err
		}
		return w.Flush()
	}
}
