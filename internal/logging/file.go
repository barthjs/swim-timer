package logging

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Defaults for the options that are left at zero.
const (
	DefaultMaxSizeMB  = 100
	DefaultMaxAgeDays = 30
)

// rotateRetryInterval is how long Write waits after a failed rotation before it tries again,
// so a full disk does not cost a rotation attempt on every log line.
const rotateRetryInterval = time.Minute

// ErrNoFilename is returned when OpenFile gets no file name.
var ErrNoFilename = errors.New("logging: a file name is required")

// FileOptions holds the settings of a log file.
type FileOptions struct {
	// Filename is the path of the active log file. Rotated files are stored next to it.
	Filename string
	// MaxSizeMB rotates the file early once it reaches this size, DefaultMaxSizeMB if zero.
	MaxSizeMB int
	// MaxAgeDays removes rotated files older than this many days, DefaultMaxAgeDays if zero.
	// A negative value keeps them forever.
	MaxAgeDays int
	// Compress stores rotated files gzip-compressed.
	Compress bool
	// Clock returns the current time and decides when a new day begins, time.Now if nil.
	Clock func() time.Time
	// Fallback receives the lines that cannot be written to the file, os.Stdout if nil.
	Fallback io.Writer
	// Errors receives the report of a failed rotation or write, os.Stderr if nil.
	Errors io.Writer
}

// File is an io.WriteCloser that rotates once a day and when it grows too large.
// It is safe for concurrent use. Rotated files are named after the time of the rotation,
// optionally compressed, and removed once they are older than the retention period.
//
// The daily rotation happens on the first write after local midnight instead of on a timer,
// so no goroutine runs and a quiet process keeps the last day's file until it logs again.
// The size limit is a safety net for days with unusually many logs.
//
// A failed rotation, such as on a full disk, keeps the current file and is retried after rotateRetryInterval.
// A line the file does not take goes to the fallback, so no line is lost. Each kind of failure is reported once
// until the file works again, since a report per line would flood the error output.
type File struct {
	mu          sync.Mutex
	writer      *lumberjack.Logger
	clock       func() time.Time
	fallback    io.Writer
	errorOutput io.Writer
	day         time.Time
	// retryAt delays the next rotation after a failed one.
	retryAt time.Time
	// rotateFailed and writeFailed mark a reported failure that has not recovered yet.
	rotateFailed bool
	writeFailed  bool
}

// OpenFile prepares the log file. The file itself is created on the first write.
// An existing file from an earlier day is rotated before the first write of today.
func OpenFile(options FileOptions) (*File, error) {
	if options.Filename == "" {
		return nil, ErrNoFilename
	}

	clock := options.Clock
	if clock == nil {
		clock = time.Now
	}

	maxSize := options.MaxSizeMB
	if maxSize <= 0 {
		maxSize = DefaultMaxSizeMB
	}

	maxAge := options.MaxAgeDays
	switch {
	case maxAge == 0:
		maxAge = DefaultMaxAgeDays
	case maxAge < 0:
		// lumberjack keeps rotated files forever with a zero age
		maxAge = 0
	}

	fallback := options.Fallback
	if fallback == nil {
		fallback = os.Stdout
	}

	errorOutput := options.Errors
	if errorOutput == nil {
		errorOutput = os.Stderr
	}

	day := startOfDay(clock())
	if info, err := os.Stat(options.Filename); err == nil {
		day = startOfDay(info.ModTime())
	}

	return &File{
		writer: &lumberjack.Logger{
			Filename:  options.Filename,
			MaxSize:   maxSize,
			MaxAge:    maxAge,
			Compress:  options.Compress,
			LocalTime: true,
		},
		clock:       clock,
		fallback:    fallback,
		errorOutput: errorOutput,
		day:         day,
	}, nil
}

// Write appends p to the log file, rotating it first if a new day has begun since the last write.
// If the file does not take p, Write passes it to the fallback instead.
func (f *File) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.rotateIfDue(f.clock())

	n, err := f.writer.Write(p)
	if err == nil {
		f.writeFailed = false
		return n, nil
	}

	if !f.writeFailed {
		f.writeFailed = true
		f.report("writing the log file failed, writing to the fallback until it works again", err)
	}
	return f.fallback.Write(p)
}

// Close closes the active log file.
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.writer.Close()
}

// rotateIfDue rotates the file once a new day has begun, unless a failed rotation waits for its retry.
func (f *File) rotateIfDue(now time.Time) {
	today := startOfDay(now)
	if !today.After(f.day) || now.Before(f.retryAt) {
		return
	}

	if err := f.writer.Rotate(); err != nil {
		f.retryAt = now.Add(rotateRetryInterval)
		if !f.rotateFailed {
			f.rotateFailed = true
			f.report("rotating the log file failed, retrying every "+rotateRetryInterval.String(), err)
		}
		return
	}

	f.day = today
	f.retryAt = time.Time{}
	f.rotateFailed = false
}

// report writes a failure of the log file to the error output, which is the only place left to report it.
func (f *File) report(message string, err error) {
	_, _ = fmt.Fprintf(f.errorOutput, "logging: %s: %v\n", message, err)
}

// startOfDay returns local midnight of the day t falls on.
func startOfDay(t time.Time) time.Time {
	year, month, day := t.Local().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}
