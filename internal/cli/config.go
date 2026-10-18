package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// newConfigCommand groups the commands that create and inspect the configuration.
func newConfigCommand(s *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Create and inspect the configuration",
	}
	cmd.AddCommand(newConfigInitCommand(), newConfigPathCommand(), newConfigShowCommand(s), newConfigValidateCommand())
	return cmd
}

// newConfigInitCommand writes a config file with every setting at its default, each explained by a comment.
func newConfigInitCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "init [file]",
		Short:   "Write a commented config file with the defaults",
		Long:    "Writes a config file with every setting at its default, each explained by a comment.\nWithout a file, it prints the config to the standard output.",
		Example: fmt.Sprintf("  %[1]s config init /etc/%[2]s/config.yaml\n  %[1]s config init > config.yaml", cmdline.BinaryName(), config.AppName),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return structconf.Template(cmd.OutOrStdout(), &config.Config{})
			}
			return writeTemplate(args[0], force)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite an existing file")
	return cmdline.SkipSetup(cmd)
}

// writeTemplate writes the commented config to path. An existing file is only replaced with force.
func writeTemplate(path string, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists, use --force to overwrite it", path)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	var buf bytes.Buffer
	if err := structconf.Template(&buf, &config.Config{}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}

// configPaths is the output of the path command.
type configPaths struct {
	File   string   `json:"file" yaml:"file"`
	Search []string `json:"search" yaml:"search"`
}

// newConfigPathCommand shows which config file is read and where one is searched for.
func newConfigPathCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show which config file is read and where it is searched for",
		Long: fmt.Sprintf("Shows the config file in use and the locations searched for one, in order.\n"+
			"--%s or %s select a file instead of the search.", structconf.FileFlag, structconf.FileEnv),
		Args: cobra.NoArgs,
	}
	output := cmdline.AddOutputFlag(cmd, cmdline.OutputText, cmdline.OutputJSON, cmdline.OutputYAML)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		paths := configPaths{
			File:   structconf.FindFile(config.AppName, cmd.Flags()),
			Search: structconf.SearchPaths(config.AppName),
		}
		return cmdline.NewPrinter(cmd, *output).Print(paths, func(w io.Writer) error {
			file := paths.File
			if file == "" {
				file = "none, only defaults, environment, and flags apply"
			}
			if _, err := fmt.Fprintf(w, "Config file:\t%s\n\nSearch order:\n", file); err != nil {
				return err
			}
			for _, path := range paths.Search {
				if _, err := fmt.Fprintf(w, "  %s\n", path); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return cmdline.SkipSetup(cmd)
}

// setting is one row of the show command with --origin.
type setting struct {
	Key    string `json:"key" yaml:"key"`
	Value  any    `json:"value" yaml:"value"`
	Source string `json:"source" yaml:"source"`
}

// newConfigShowCommand prints the effective configuration with every secret masked.
func newConfigShowCommand(s *state) *cobra.Command {
	var origin bool
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Print the effective configuration with secrets masked",
		Long:    "Prints the configuration as it results from the defaults, the config file, the environment, and the flags.\nSecrets are masked.",
		Example: fmt.Sprintf("  %[1]s config show\n  %[1]s config show --origin -o table", cmdline.BinaryName()),
		Args:    cobra.NoArgs,
	}
	cmd.Flags().BoolVar(&origin, "origin", false, "Show where each value comes from")
	output := cmdline.AddOutputFlag(cmd, cmdline.OutputYAML, cmdline.OutputJSON, cmdline.OutputTable)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		redacted := structconf.Redacted(s.cfg)
		values := flatten(structconf.ToMap(&redacted), "")
		printer := cmdline.NewPrinter(cmd, *output)

		if !origin {
			return printer.Print(structconf.ToMap(&redacted), func(w io.Writer) error {
				if _, err := fmt.Fprintln(w, "KEY\tVALUE"); err != nil {
					return err
				}
				for _, key := range slices.Sorted(maps.Keys(values)) {
					if _, err := fmt.Fprintf(w, "%s\t%v\n", key, values[key]); err != nil {
						return err
					}
				}
				return nil
			})
		}

		sources, err := structconf.Sources(config.AppName, cmd.Flags(), &config.Config{})
		if err != nil {
			return err
		}
		settings := make([]setting, 0, len(values))
		for _, key := range slices.Sorted(maps.Keys(values)) {
			settings = append(settings, setting{
				Key:    key,
				Value:  values[key],
				Source: sources[key].String(),
			})
		}
		return printer.Print(settings, func(w io.Writer) error {
			if _, err := fmt.Fprintln(w, "KEY\tVALUE\tSOURCE"); err != nil {
				return err
			}
			for _, row := range settings {
				if _, err := fmt.Fprintf(w, "%s\t%v\t%s\n", row.Key, row.Value, row.Source); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return cmd
}

// flatten turns nested maps into one map keyed by dotted keys.
func flatten(nested map[string]any, prefix string) map[string]any {
	flat := map[string]any{}
	for key, value := range nested {
		if prefix != "" {
			key = prefix + "." + key
		}
		if inner, ok := value.(map[string]any); ok {
			maps.Copy(flat, flatten(inner, key))
			continue
		}
		flat[key] = value
	}
	return flat
}

// newConfigValidateCommand checks the configuration and reports every problem. Without a file,
// it checks the configuration the application would start with. A file is checked on its own,
// without the environment and the flags, for example before it is deployed. It runs without the setup,
// since the setup itself fails on an invalid configuration.
func newConfigValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [file]",
		Short: "Check the configuration and report every problem",
		Long: "Without a file, checks the configuration the application would start with: defaults, config file, environment, and flags.\n" +
			"With a file, checks only that file on top of the defaults.",
		Example: fmt.Sprintf("  %[1]s config validate\n  %[1]s config validate ./config.yaml", cmdline.BinaryName()),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return validateFile(cmd, args[0])
			}

			if _, err := config.Load(cmd.Flags()); err != nil {
				return cmdline.Exit(1, fmt.Errorf("the configuration is invalid:\n%w", err))
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "The configuration is valid (%s).\n", configSource(cmd))
			return err
		},
	}
	return cmdline.SkipSetup(cmd)
}

// validateFile checks a single config file on top of the defaults.
func validateFile(cmd *cobra.Command, path string) error {
	var cfg config.Config
	if err := structconf.LoadFile(path, &cfg); err != nil {
		return cmdline.Exit(1, err)
	}
	if err := config.Validate(&cfg); err != nil {
		return cmdline.Exit(1, fmt.Errorf("%s is invalid:\n%w", path, err))
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s is valid.\n", path)
	return err
}

// configSource describes where the configuration came from: the config file if one was read,
// otherwise only the defaults, the environment, and the flags.
func configSource(cmd *cobra.Command) string {
	path := structconf.FindFile(config.AppName, cmd.Flags())
	if path == "" {
		return "defaults, environment, and flags"
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Sprintf("defaults, environment, and flags, %s does not exist", path)
	}
	return "defaults, config file " + path + ", environment, and flags"
}
