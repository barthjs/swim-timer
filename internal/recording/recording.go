// Package recording writes and reads recordings of the raw byte stream of a timing device.
// A recording is a JSON Lines file. Every start of the application appends a session line with its metadata,
// followed by the meet files in use, the data as it arrived, and the gaps where data had to be dropped.
package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// FormatVersion is the version of the recording format the package writes and reads.
const FormatVersion = 1

// EntryType is the kind of a line in a recording.
type EntryType string

const (
	// TypeSession starts a session, written once per start of the application.
	TypeSession EntryType = "session"
	// TypeMeet holds every meet file in use from this point on.
	TypeMeet EntryType = "meet"
	// TypeData holds the bytes of one read from the device, at the time they arrived.
	TypeData EntryType = "data"
	// TypeGap reports bytes that were dropped before this point, since the recording could not keep up.
	TypeGap EntryType = "gap"
)

// Source is where the recorded data came from.
type Source string

const (
	// SourceDevice is a timing device.
	SourceDevice Source = "device"
	// SourceSimulator is the simulator with generated times.
	SourceSimulator Source = "simulator"
)

// Session describes the application that wrote a session.
type Session struct {
	// Format is the version of the recording format.
	Format int `json:"format"`
	// Version is the version of the application.
	Version string `json:"version"`
	// Source is where the data came from.
	Source Source `json:"source"`
	// Port is the configured port, empty if every port found was tried.
	Port string `json:"port"`
	// BaudRate is the configured baud rate.
	BaudRate int `json:"baudRate"`
}

// MeetFile is one meet file with its content as it was loaded.
type MeetFile struct {
	// Name is the name of the file without its extension, such as LSTSTART.
	Name string `json:"name"`
	// SHA256 is the hex encoded hash of the content.
	SHA256 string `json:"sha256"`
	// Content is the content of the file in its original encoding.
	Content []byte `json:"content"`
}

// NewMeetFile returns a meet file with the hash of its content.
func NewMeetFile(name string, content []byte) MeetFile {
	return MeetFile{
		Name:    name,
		SHA256:  Hash(content),
		Content: content,
	}
}

// Hash returns the hex encoded SHA-256 hash of content, as a MeetFile stores it.
func Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// Entry is one line of a recording. Which fields are set depends on its type.
type Entry struct {
	// Type decides which of the other fields are set.
	Type EntryType `json:"type"`
	// At is when the line was written, for data when the bytes arrived, with the UTC offset of the host.
	At time.Time `json:"at"`

	// Session is set for a session line.
	*Session

	// Files is set for a meet line.
	Files []MeetFile `json:"files,omitempty"`
	// Data is set for a data line.
	Data HexBytes `json:"data,omitempty"`
	// DroppedBytes is set for a gap line.
	DroppedBytes int64 `json:"droppedBytes,omitempty"`
}

// HexBytes is a byte slice that is written as space-separated upper-case hex bytes, such as "01 02 1E",
// so the raw data of a recording stays readable.
type HexBytes []byte

// MarshalText returns the bytes as space-separated hex bytes.
func (b HexBytes) MarshalText() ([]byte, error) {
	var text strings.Builder
	for i, value := range b {
		if i > 0 {
			_ = text.WriteByte(' ')
		}
		_, _ = fmt.Fprintf(&text, "%02X", value)
	}
	return []byte(text.String()), nil
}

// UnmarshalText parses hex bytes, with or without spaces between them.
func (b *HexBytes) UnmarshalText(text []byte) error {
	decoded, err := hex.DecodeString(strings.ReplaceAll(string(text), " ", ""))
	if err != nil {
		return fmt.Errorf("recording: invalid hex data: %w", err)
	}
	*b = decoded
	return nil
}
