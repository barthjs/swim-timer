package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/app"
	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/update"
)

// updateCheck is the output of the update check command.
type updateCheck struct {
	Current     string `json:"current" yaml:"current"`
	Latest      string `json:"latest" yaml:"latest"`
	URL         string `json:"url" yaml:"url"`
	Newer       bool   `json:"newer" yaml:"newer"`
	Unsupported string `json:"unsupported,omitempty" yaml:"unsupported,omitempty"`
}

// newUpdateCommand groups the commands that check for and install a newer release.
func newUpdateCommand(build cmdline.Build) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install a newer release",
	}
	cmd.AddCommand(newUpdateCheckCommand(build), newUpdateApplyCommand(build))
	return cmd
}

// newUpdateCheckCommand compares the running version with the latest release.
func newUpdateCheckCommand(build cmdline.Build) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check whether a newer release exists",
		Args:  cobra.NoArgs,
	}
	output := cmdline.AddOutputFlag(cmd, cmdline.OutputText, cmdline.OutputJSON, cmdline.OutputYAML)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		updater := app.NewUpdater(build.Version)
		release, err := updater.Check(cmd.Context())
		if err != nil {
			return err
		}

		check := updateCheck{
			Current: build.Version,
			Latest:  release.Version,
			URL:     release.URL,
			Newer:   release.Newer,
		}
		if err := updater.Supported(); err != nil {
			check.Unsupported = err.Error()
		}

		return cmdline.NewPrinter(cmd, *output).Print(check, func(w io.Writer) error { return printUpdateCheck(w, check) })
	}
	return cmdline.SkipSetup(cmd)
}

// printUpdateCheck writes the result of a check for a person.
func printUpdateCheck(w io.Writer, check updateCheck) error {
	if !check.Newer {
		_, err := fmt.Fprintf(w, "%s is the latest version.\n", check.Current)
		return err
	}
	if _, err := fmt.Fprintf(w, "%s is available, %s is installed.\n%s\n", check.Latest, check.Current, check.URL); err != nil {
		return err
	}
	if check.Unsupported != "" {
		_, err := fmt.Fprintf(w, "This installation cannot update itself: %s.\n", check.Unsupported)
		return err
	}
	_, err := fmt.Fprintf(w, "Install it with: %s update apply\n", cmdline.BinaryName())
	return err
}

// newUpdateApplyCommand installs the latest release.
func newUpdateApplyCommand(build cmdline.Build) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Install the latest release",
		Long:  "Replaces this binary with the latest release.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			updater := app.NewUpdater(build.Version)
			if err := updater.Supported(); err != nil {
				return fmt.Errorf("this installation cannot update itself: %w", err)
			}

			release, err := updater.Check(cmd.Context())
			if err != nil {
				return err
			}
			if !release.Newer {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s is the latest version.\n", build.Version)
				return err
			}

			if err := updater.Apply(cmd.Context(), release); err != nil {
				if errors.Is(err, update.ErrNoAsset) {
					return fmt.Errorf("%s has no download for this platform: %w", release.Version, err)
				}
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Installed %s. Restart the application to use it.\n", release.Version)
			return err
		},
	}
	return cmdline.SkipSetup(cmd)
}
