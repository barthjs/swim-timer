package recording

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Reader reads the entries of a recording in order.
type Reader struct {
	reader *bufio.Reader
	// line is the number of the last line read, counted from 1.
	line int
	// sessionStarted reports whether a session line was read, which every recording starts with.
	sessionStarted bool
}

// readLine is a line of the recording with the result of parsing it.
type readLine struct {
	number int
	entry  Entry
	err    error
}

// NewReader creates a Reader that reads a recording from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		reader: bufio.NewReader(r),
	}
}

// Next returns the next entry and io.EOF after the last one. The application can stop while writing a line,
// so a broken line directly before a session line or at the end of the recording is skipped.
// Any other broken line, a recording that does not start with a session, and an unknown format fail with an error.
func (r *Reader) Next() (Entry, error) {
	current, err := r.readNext()
	if err != nil {
		return Entry{}, err
	}

	if current.err != nil {
		following, err := r.readNext()
		if errors.Is(err, io.EOF) {
			return Entry{}, io.EOF
		}
		if err != nil {
			return Entry{}, err
		}
		if following.err != nil || following.entry.Type != TypeSession {
			return Entry{}, fmt.Errorf("recording: line %d: %w", current.number, current.err)
		}
		current = following
	}

	if err := r.check(current); err != nil {
		return Entry{}, err
	}
	return current.entry, nil
}

// readNext returns the next non-empty line, parsed. It returns io.EOF at the end.
func (r *Reader) readNext() (readLine, error) {
	for {
		text, err := r.reader.ReadBytes('\n')
		if len(text) == 0 && err != nil {
			return readLine{}, err
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return readLine{}, err
		}
		r.line++

		text = bytes.TrimSpace(text)
		if len(text) == 0 {
			continue
		}

		var entry Entry
		parseErr := json.Unmarshal(text, &entry)
		if parseErr == nil {
			parseErr = validate(entry)
		}
		return readLine{
			number: r.line,
			entry:  entry,
			err:    parseErr,
		}, nil
	}
}

// validate checks that an entry has a known type and the fields of its type.
func validate(entry Entry) error {
	switch entry.Type {
	case TypeSession:
		if entry.Session == nil {
			return errors.New("session line without session data")
		}
	case TypeMeet:
		// A meet line may list no files, when no meet data is loaded
	case TypeData:
		if len(entry.Data) == 0 {
			return errors.New("data line without data")
		}
	case TypeGap:
		if entry.DroppedBytes <= 0 {
			return errors.New("gap line without dropped bytes")
		}
	default:
		return fmt.Errorf("unknown line type %q", entry.Type)
	}
	if entry.At.IsZero() {
		return errors.New("line without time")
	}
	return nil
}

// check rejects a recording that does not start with a session and a session of an unknown format.
func (r *Reader) check(line readLine) error {
	if line.entry.Type != TypeSession {
		if !r.sessionStarted {
			return fmt.Errorf("recording: line %d: the recording does not start with a session", line.number)
		}
		return nil
	}

	if line.entry.Format != FormatVersion {
		return fmt.Errorf("recording: line %d: unsupported format %d", line.number, line.entry.Format)
	}
	r.sessionStarted = true
	return nil
}
