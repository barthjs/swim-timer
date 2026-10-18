// Package logging builds the logger, writing to the standard output, a rotating log file, or both.
package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Channel selects where the logger writes.
type Channel string

const (
	ChannelStdout Channel = "stdout"
	ChannelFile   Channel = "file"
	ChannelBoth   Channel = "both"
)

// Format selects how log lines look.
type Format string

const (
	FormatJSON    Format = "json"
	FormatConsole Format = "console"
)

// consoleTimeFormat writes the time of a console log line as a sortable local date and time.
const consoleTimeFormat = time.DateTime

// Options holds the settings of the logger.
type Options struct {
	Level   zerolog.Level
	Channel Channel
	Format  Format
	// File configures the log file for the file and both channels, see File.
	File FileOptions
	// Stdout replaces the standard output for tests, os.Stdout if nil.
	Stdout io.Writer
}

// noopCloser is returned when the logger writes to no file,
// so callers can always close the logger without checking the channel.
type noopCloser struct{}

// Close does nothing, since there is no file to close.
func (noopCloser) Close() error { return nil }

// New creates a logger and returns a closer that releases the log file.
// A file channel writes through a log file that rotates daily and at a size limit.
func New(options Options) (zerolog.Logger, io.Closer, error) {
	stdout := options.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	var file io.WriteCloser
	if options.Channel == ChannelFile || options.Channel == ChannelBoth {
		fileOptions := options.File
		if fileOptions.Fallback == nil {
			fileOptions.Fallback = fileFallback(options.Channel, stdout)
		}
		opened, err := OpenFile(fileOptions)
		if err != nil {
			return zerolog.Nop(), noopCloser{}, fmt.Errorf("opening the log file: %w", err)
		}
		file = opened
	}

	logger := zerolog.New(logWriter(options, stdout, file)).Level(options.Level).With().Timestamp().Logger()
	if file == nil {
		return logger, noopCloser{}, nil
	}
	return logger, file, nil
}

// fileFallback returns where the lines go that the log file does not take. With both channels,
// the line already reaches the standard output, so writing it there again would only duplicate it.
func fileFallback(channel Channel, stdout io.Writer) io.Writer {
	if channel == ChannelBoth {
		return io.Discard
	}
	return stdout
}

// logWriter assembles the output for the channel and format. The console format's color codes only make sense
// on a terminal, so a file never gets them, otherwise the log file fills up with raw escape sequences.
func logWriter(options Options, stdout io.Writer, file io.Writer) io.Writer {
	format := func(w io.Writer, noColor bool) io.Writer {
		if options.Format != FormatConsole {
			return w
		}
		return zerolog.ConsoleWriter{
			Out:        w,
			NoColor:    noColor,
			TimeFormat: consoleTimeFormat,
		}
	}

	switch options.Channel {
	case ChannelFile:
		return format(file, true)
	case ChannelBoth:
		return io.MultiWriter(format(stdout, false), format(file, true))
	default:
		return format(stdout, false)
	}
}
