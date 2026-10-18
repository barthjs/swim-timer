package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/store"
)

// tenLaneFiles describes event 7, a 100 m backstroke event in phase 2 with a single heat in a pool numbered 1 to 10.
var tenLaneFiles = map[string]string{
	"LSTROUND.TXT": "idRound;TITLE;RoundAbrev;roundtext;sequence\n2;\"Semifinal\";\"SEM\";\"Heat\";\"1\"\n",
	"LSTCAT.TXT":   "Category;AbrevCat\nOpen;\"O\"\n",
	"LSTLONG.TXT":  "idLength;Longueur;Mlongueur;Relais\n1;\"100 m\";100;1\n",
	"LSTSTYLE.TXT": "idStyle;Style;StyleAbrév\n3;\"Backstroke\";\"BA\"\n",
	"LSTRACE.TXT":  "event;round;nbHeat;idLen;idStyle;abCat;date;time\n7;2;1;1;3;\"O\";\"26/05/01\";\"10:00\"\n",
	"LSTCONC.TXT":  "id;bib;lastname;firstname;birthyear;abNat;abCat\n1;;\"One\";\"Erika\";\"2001\";\"SC A\";\"O\";\n10;;\"Young\";\"Zora\";\"2002\";\"SC B\";\"O\";\n",
	"LSTSTART.TXT": "event;round;heat;lane;relais;idBib\n7;2;0;1;0;1;\n7;2;0;10;0;10;\n7;1;0;5;0;1;\n",
}

// testReplayName and testReplayPath are the name and the storage path writeRecording stores a recording under.
const (
	testReplayName = "recording.jsonl"
	testReplayPath = "recordings/" + testReplayName
)

// writeRecording stores entries as a recording at testReplayPath and returns the store that holds it.
func writeRecording(t *testing.T, entries ...recording.Entry) *store.RecordingStore {
	t.Helper()

	return writeRawRecording(t, mustMarshalLines(t, entries...))
}

// writeRawRecording stores content as a recording at testReplayPath and returns the store that holds it.
func writeRawRecording(t *testing.T, content string) *store.RecordingStore {
	t.Helper()

	recordings := store.NewRecordingStore(newTestStorage(t))
	_, err := recordings.Create(t.Context(), testReplayName, []byte(content))
	require.NoError(t, err)
	return recordings
}

// newTestStorage returns a local storage in a temporary directory of the test.
func newTestStorage(t *testing.T) media.Storage {
	t.Helper()

	storage, err := media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: t.TempDir(),
	})
	require.NoError(t, err)
	return storage
}

// newTestMeetStore loads meet files from a temporary storage,
// encoded as Windows-1252 like the exports of the timing software.
func newTestMeetStore(t *testing.T, files map[string]string) *store.MeetStore {
	t.Helper()

	meetStore, _ := newTestMeetStoreWithRoot(t, files)
	return meetStore
}

// newTestMeetStoreWithRoot is newTestMeetStore that also returns the root of the storage, to change the files later.
func newTestMeetStoreWithRoot(t *testing.T, files map[string]string) (*store.MeetStore, string) {
	t.Helper()

	root := t.TempDir()
	writeMeetFiles(t, root, files)

	storage, err := media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: root,
	})
	require.NoError(t, err)
	meetStore := store.NewMeetStore(storage)
	require.NoError(t, meetStore.LoadAll(t.Context()))
	return meetStore, root
}

// writeMeetFiles writes meet files below root, encoded as Windows-1252.
func writeMeetFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()

	dir := filepath.Join(root, "meet")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for name, content := range files {
		encoded, err := charmap.Windows1252.NewEncoder().String(content)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(encoded), 0o644))
	}
}

// readRecording returns the entries of the only recording file in dir.
func readRecording(t *testing.T, dir string) []recording.Entry {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, "recording_*.jsonl"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	file, err := os.Open(files[0])
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	reader := recording.NewReader(file)
	var entries []recording.Entry
	for {
		entry, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return entries
		}
		require.NoError(t, err)
		entries = append(entries, entry)
	}
}

// newTestRecorder returns a recorder that writes into dir.
func newTestRecorder(dir string) *recording.Writer {
	return recording.NewWriter(zerolog.Nop(), recording.WriterOptions{
		Dir: dir,
		Session: recording.Session{
			Source: recording.SourceDevice,
		},
	})
}

// meetFileNames returns the name of every recorded meet file.
func meetFileNames(files []recording.MeetFile) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name)
	}
	return names
}

// mustMarshalLines encodes entries as the lines of a recording.
func mustMarshalLines(t *testing.T, entries ...recording.Entry) string {
	t.Helper()

	var lines bytes.Buffer
	encoder := json.NewEncoder(&lines)
	for _, entry := range entries {
		require.NoError(t, encoder.Encode(entry))
	}
	return lines.String()
}

// sessionEntry returns the line that starts a session at.
func sessionEntry(at time.Time) recording.Entry {
	return recording.Entry{
		Type: recording.TypeSession,
		At:   at,
		Session: &recording.Session{
			Format: recording.FormatVersion,
			Source: recording.SourceDevice,
		},
	}
}

// dataEntry returns a data line with the bytes data that arrived at.
func dataEntry(at time.Time, data ...byte) recording.Entry {
	return recording.Entry{
		Type: recording.TypeData,
		At:   at,
		Data: data,
	}
}

// gapEntry returns a gap line at.
func gapEntry(at time.Time) recording.Entry {
	return recording.Entry{
		Type:         recording.TypeGap,
		At:           at,
		DroppedBytes: 1,
	}
}

// assertValidation checks that err is a validation error with the tag on the field.
func assertValidation(t *testing.T, err error, field, tag string) {
	t.Helper()

	validation, ok := errors.AsType[*apperr.ValidationError](err)
	require.True(t, ok)
	require.Contains(t, validation.Fields, field)
	tags := make([]string, 0, len(validation.Fields[field]))
	for _, fieldErr := range validation.Fields[field] {
		tags = append(tags, fieldErr.Tag)
	}
	assert.Contains(t, tags, tag)
}
