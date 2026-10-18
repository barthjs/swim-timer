package cmdline

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Build describes the binary and what the build stamped into it.
type Build struct {
	// Title is the display name in the version output.
	Title string
	// Version is set at build time, "dev" for local builds.
	Version string
	// Commit is the short VCS revision. Empty means it is read from the build information.
	Commit string
}

// versionInfo is the version output in JSON or YAML.
type versionInfo struct {
	Name      string `json:"name" yaml:"name"`
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit,omitempty" yaml:"commit,omitempty"`
	Arch      string `json:"arch,omitempty" yaml:"arch,omitempty"`
	GoVersion string `json:"goVersion" yaml:"goVersion"`
	Platform  string `json:"platform" yaml:"platform"`
}

// ResolveCommit returns the short VCS revision the binary was built from,
// or an empty string when the build carries none, for example in tests.
func ResolveCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
			return setting.Value[:7]
		}
	}
	return ""
}

// ResolveArch returns the architecture the binary was built for, with the ARM version for 32-bit ARM,
// such as "amd64", "arm64", or "armv7". Release archives are named after it.
func ResolveArch() string {
	arch := runtime.GOARCH
	if arch != "arm" {
		return arch
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return arch
	}
	for _, setting := range info.Settings {
		if setting.Key == "GOARM" {
			// The setting may carry the floating point mode, such as "7,softfloat"
			version, _, _ := strings.Cut(setting.Value, ",")
			return arch + "v" + version
		}
	}
	return arch
}

// VersionText returns the version information as aligned text lines.
func VersionText(b Build) string {
	info := b.info()

	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "%s\t%s\n", info.Name, info.Version)
	if info.Commit != "" {
		_, _ = fmt.Fprintf(&sb, "Commit:\t\t%s\n", info.Commit)
	}
	_, _ = fmt.Fprintf(&sb, "Go Version:\t%s\n", info.GoVersion)
	_, _ = fmt.Fprintf(&sb, "OS/Arch:\t%s\n", info.Platform)
	return sb.String()
}

// VersionCommand prints the version information, as text or, with --output, as JSON or YAML.
func VersionCommand(b Build) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Args:  cobra.NoArgs,
	}
	output := AddOutputFlag(cmd, OutputText, OutputJSON, OutputYAML)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if *output == OutputText {
			_, err := fmt.Fprint(cmd.OutOrStdout(), VersionText(b))
			return err
		}
		return NewPrinter(cmd, *output).Print(b.info(), nil)
	}

	return SkipSetup(cmd)
}

// info collects the build details shown by the version command.
func (b Build) info() versionInfo {
	commit := b.Commit
	if commit == "" {
		commit = ResolveCommit()
	}
	return versionInfo{
		Name:      b.Title,
		Version:   b.Version,
		Commit:    commit,
		Arch:      ResolveArch(),
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}
