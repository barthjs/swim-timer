// Package cli is the command-line interface of the application. Without a subcommand it starts the application.
// Commands stay thin. They read their arguments, build what they need, and call the application, see internal/app.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// Help groups of the root command.
const (
	groupMain    = "main"
	groupUtility = "utility"
)

// Execute runs the command line and returns the exit code.
func Execute(version string) int {
	return cmdline.Execute(newRoot(cmdline.Build{
		Title:   config.AppTitle,
		Version: version,
	}), nil)
}

// newRoot builds the command tree. Without a subcommand, the root command starts the application.
// Every configuration setting is a persistent flag, so it can be passed to any command.
func newRoot(build cmdline.Build) *cobra.Command {
	s := &state{
		build: build,
	}

	root := cmdline.NewRoot(build, "Overlays for swimming competitions from Swiss Timing devices")
	root.Long = "Overlays for swimming competitions from Swiss Timing devices.\n" +
		"Without a command, it starts the application. Every setting can also be passed as a flag."
	root.Example = fmt.Sprintf("  %[1]s\n  %[1]s --server-port 9000\n  %[1]s config show --origin", cmdline.BinaryName())
	root.Args = cobra.NoArgs
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		return runServe(cmd, s)
	}

	root.PersistentFlags().String(structconf.FileFlag, "", "Path to a config file (yaml, json, or toml)")
	if err := structconf.RegisterFlags(root.PersistentFlags(), &config.Config{}); err != nil {
		// The flags come from the struct tags of the config, so this only fails during development
		panic(fmt.Sprintf("registering the config flags: %v", err))
	}
	registerValueCompletions(root)

	cmdline.PersistentSetup(root, s.setup)

	root.AddGroup(
		&cobra.Group{
			ID:    groupMain,
			Title: "Commands:",
		},
		&cobra.Group{
			ID:    groupUtility,
			Title: "Additional Commands:",
		},
	)
	root.SetHelpCommandGroupID(groupUtility)
	root.SetCompletionCommandGroupID(groupUtility)

	root.AddCommand(
		inGroup(groupMain, newConfigCommand(s)),
		inGroup(groupMain, newHealthcheckCommand()),
		inGroup(groupMain, newRecordingCommand()),
		inGroup(groupMain, newUpdateCommand(build)),
		inGroup(groupUtility, cmdline.VersionCommand(build)),
		inGroup(groupUtility, newGenCommand(root)),
	)

	return root
}

// newGenCommand groups the commands that generate files for packaging.
func newGenCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate files for packaging",
	}
	cmd.AddCommand(cmdline.ManCommand(root))
	return cmd
}

// inGroup puts cmd into a help group of the root command.
func inGroup(groupID string, cmd *cobra.Command) *cobra.Command {
	cmd.GroupID = groupID
	return cmd
}

// registerValueCompletions lets the shell complete the flags of settings with a fixed set of values.
func registerValueCompletions(root *cobra.Command) {
	for _, leaf := range structconf.Leaves(&config.Config{}) {
		if len(leaf.Values) == 0 {
			continue
		}
		_ = root.RegisterFlagCompletionFunc(leaf.Flag, cobra.FixedCompletions(leaf.Values, cobra.ShellCompDirectiveNoFileComp))
	}
}
