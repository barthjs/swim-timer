package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/service"
)

// newRecordingCommand groups the commands that process the recordings of the timing data.
func newRecordingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recording",
		Short: "Process recordings of the timing data",
	}
	cmd.AddCommand(newRecordingCSVCommand())
	return cmd
}

// newRecordingCSVCommand converts a recording into a CSV table with the decoded frames.
func newRecordingCSVCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "csv <recording> [file]",
		Short: "Convert a recording into a CSV table",
		Long: "Decodes the frames of a recording and writes one row per frame, discarded block, gap, session, and meet line.\n" +
			"Without a file, it prints the table to the standard output.",
		Example: fmt.Sprintf("  %[1]s recording csv recording_2026-10-18.jsonl recording.csv\n"+
			"  %[1]s recording csv recording_2026-10-18.jsonl > recording.csv", cmdline.BinaryName()),
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return exportRecordingCSV(args[0], cmd.OutOrStdout())
			}
			return writeRecordingCSV(args[0], args[1])
		},
	}
	return cmdline.SkipSetup(cmd)
}

// writeRecordingCSV converts the recording at path into a CSV file at target.
// A conversion that fails removes the incomplete file.
func writeRecordingCSV(path, target string) (err error) {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, file.Close())
		if err != nil {
			_ = os.Remove(target)
		}
	}()

	return exportRecordingCSV(path, file)
}

// exportRecordingCSV converts the recording at path into a CSV table written to w.
func exportRecordingCSV(path string, w io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := service.NewRecordingService(service.RecordingOptions{}, nil, nil).ExportCSV(file, w); err != nil {
		return fmt.Errorf("converting %s: %w", path, err)
	}
	return nil
}
