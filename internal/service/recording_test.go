package service

import (
	"bytes"
	"encoding/csv"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/store"
)

func TestRecordingService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the recordings and marks the one being written", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		recordings, root := newTestRecordingStore(t)
		stored, err := recordings.Create(ctx, "0"+apitest.Faker(t).LetterN(8)+".jsonl", []byte(testRecording(t)))
		require.NoError(t, err)
		recorder, dir := newStorageRecorder(t, root)
		svc := NewRecordingService(RecordingOptions{}, recorder, recordings)

		got, err := svc.List(ctx)

		require.NoError(t, err)
		current := recorder.FileName()
		require.FileExists(t, filepath.Join(dir, current))
		assert.Equal(t, []domain.Recording{
			stored,
			{
				Name:    current,
				Path:    "recordings/" + current,
				Current: true,
			},
		}, got, "sorted by name")
	})
}

func TestRecordingService_Create(t *testing.T) {
	t.Parallel()

	t.Run("stores the recording under the sanitized name", func(t *testing.T) {
		t.Parallel()

		svc := NewRecordingService(RecordingOptions{}, nil, store.NewRecordingStore(newTestStorage(t)))

		got, err := svc.Create(t.Context(), "../day one.JSONL", []byte(testRecording(t)))

		require.NoError(t, err)
		assert.Equal(t, domain.Recording{
			Name: "day-one.jsonl",
			Path: "recordings/day-one.jsonl",
		}, got)
	})

	for _, name := range []string{"   ", "..", ".jsonl", "..jsonl"} {
		t.Run("names a recording without a usable name "+name, func(t *testing.T) {
			t.Parallel()

			svc := NewRecordingService(RecordingOptions{}, nil, store.NewRecordingStore(newTestStorage(t)))

			got, err := svc.Create(t.Context(), name, []byte(testRecording(t)))

			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(got.Path, "recordings/recording-"), got.Path)
			assert.True(t, strings.HasSuffix(got.Path, ".jsonl"), got.Path)
		})
	}

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "rejects an empty file",
			content: "",
		},
		{
			name:    "rejects a file that is no recording",
			content: "Timestamp,Data\n",
		},
		{
			name:    "rejects a recording without a session",
			content: mustMarshalLines(t, dataEntry(time.Now(), 0x01)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewRecordingService(RecordingOptions{}, nil, store.NewRecordingStore(newTestStorage(t)))

			_, err := svc.Create(t.Context(), "recording.jsonl", []byte(tt.content))

			assertValidation(t, err, "file", "invalid")
		})
	}

	t.Run("rejects a name that is too long", func(t *testing.T) {
		t.Parallel()

		svc := NewRecordingService(RecordingOptions{}, nil, store.NewRecordingStore(newTestStorage(t)))

		_, err := svc.Create(t.Context(), strings.Repeat("a", 300)+".jsonl", []byte(testRecording(t)))

		assertValidation(t, err, "file", "invalid")
	})

	t.Run("rejects the name of the recording being written", func(t *testing.T) {
		t.Parallel()

		recordings, root := newTestRecordingStore(t)
		recorder, _ := newStorageRecorder(t, root)
		svc := NewRecordingService(RecordingOptions{}, recorder, recordings)

		_, err := svc.Create(t.Context(), recorder.FileName(), []byte(testRecording(t)))

		assert.ErrorIs(t, err, ErrRecordingInUse)
	})
}

func TestRecordingService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the recording", func(t *testing.T) {
		t.Parallel()

		recordings := writeRawRecording(t, testRecording(t))
		svc := NewRecordingService(RecordingOptions{}, nil, recordings)

		err := svc.Delete(t.Context(), testReplayName)

		require.NoError(t, err)
		got, err := recordings.List(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	for _, name := range []string{"missing.jsonl", "../secret.jsonl", "recording.csv"} {
		t.Run("returns not found for "+name, func(t *testing.T) {
			t.Parallel()

			svc := NewRecordingService(RecordingOptions{}, nil, store.NewRecordingStore(newTestStorage(t)))

			err := svc.Delete(t.Context(), name)

			assert.ErrorIs(t, err, apperr.ErrNotFound)
		})
	}

	t.Run("keeps the replayed recording", func(t *testing.T) {
		t.Parallel()

		svc := NewRecordingService(RecordingOptions{
			ReplayPath: testReplayPath,
		}, nil, writeRawRecording(t, testRecording(t)))

		err := svc.Delete(t.Context(), testReplayName)

		assert.ErrorIs(t, err, ErrRecordingInUse)
	})

	t.Run("keeps the recording being written", func(t *testing.T) {
		t.Parallel()

		recordings, root := newTestRecordingStore(t)
		recorder, _ := newStorageRecorder(t, root)
		svc := NewRecordingService(RecordingOptions{}, recorder, recordings)

		err := svc.Delete(t.Context(), recorder.FileName())

		assert.ErrorIs(t, err, ErrRecordingInUse)
	})
}

func TestRecordingService_Download(t *testing.T) {
	t.Parallel()

	t.Run("returns the recording", func(t *testing.T) {
		t.Parallel()

		content := testRecording(t)
		svc := NewRecordingService(RecordingOptions{}, nil, writeRawRecording(t, content))

		reader, err := svc.Download(t.Context(), testReplayName)

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	for _, name := range []string{"missing.jsonl", "../secret.jsonl", "recording.csv"} {
		t.Run("returns not found for "+name, func(t *testing.T) {
			t.Parallel()

			svc := NewRecordingService(RecordingOptions{}, nil, store.NewRecordingStore(newTestStorage(t)))

			_, err := svc.Download(t.Context(), name)

			assert.ErrorIs(t, err, apperr.ErrNotFound)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "keeps a plain name",
			input: "recording.jsonl",
			want:  "recording.jsonl",
		},
		{
			name:  "drops the path of a traversal",
			input: "../../etc/passwd",
			want:  "passwd",
		},
		{
			name:  "replaces spaces and symbols",
			input: "  my recording (1).jsonl ",
			want:  "my-recording-1-.jsonl",
		},
		{
			name:  "replaces non-ASCII letters",
			input: "Café-Résumé.jsonl",
			want:  "Caf--R-sum-.jsonl",
		},
		{
			name:  "returns nothing for blanks",
			input: "   ",
			want:  "",
		},
		{
			name:  "returns nothing for the parent directory",
			input: "..",
			want:  "",
		},
		{
			name:  "returns nothing for only slashes",
			input: "///",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name := sanitizeFilename(tt.input)

			assert.Equal(t, tt.want, name)
		})
	}
}

func TestRecordingService_ExportCSV(t *testing.T) {
	t.Parallel()

	// The recording was made on a host two hours ahead of UTC
	zone := time.FixedZone("", 2*60*60)
	at := time.Date(2026, 10, 18, 9, 30, 0, 0, zone)
	load := omega.Encode(omega.FrameData{
		Kind:      omega.KindLoad,
		UsedLanes: []int{3, 4},
		LapCount:  4,
		Event:     101,
		Heat:      2,
	})
	split := omega.Encode(omega.FrameData{
		Kind:      omega.KindIntermediate,
		UsedLanes: []int{3, 4},
		LapCount:  4,
		Event:     101,
		Heat:      2,
		Rank:      1,
		Lane:      4,
		Lap:       1,
		Time:      30100 * time.Millisecond,
	})

	t.Run("writes a row per line and frame in the order of the recording", func(t *testing.T) {
		t.Parallel()

		content := mustMarshalLines(t,
			sessionEntry(at),
			recording.Entry{
				Type: recording.TypeMeet,
				At:   at,
				Files: []recording.MeetFile{
					recording.NewMeetFile("LSTRACE", []byte("event")),
					recording.NewMeetFile("LSTSTART", []byte("start")),
				},
			},
			dataEntry(at.Add(time.Second), load[:20]...),
			dataEntry(at.Add(2*time.Second), load[20:]...),
			dataEntry(at.Add(3*time.Second), 'x', 'y'),
			gapEntry(at.Add(4*time.Second)),
			dataEntry(at.Add(5*time.Second), split...),
		)

		rows := exportCSV(t, content)

		assert.Equal(t, []string{
			"session",
			"meet",
			"frame",
			"gap",
			"discarded",
			"frame",
		}, rows.column("Type"), "noise is recognized with the header that follows it")
		assert.Equal(t, "2026-10-18T09:30:02.000+02:00", rows.value(2, "Time"), "a frame is complete with its last byte")
		assert.Equal(t, "load", rows.value(2, "Kind"))
		assert.Equal(t, "3 4", rows.value(2, "UsedLanes"))
		assert.Equal(t, "LSTRACE LSTSTART", rows.value(1, "Details"))
		assert.Equal(t, "2", rows.value(4, "DroppedBytes"))
		assert.Equal(t, "78 79", rows.value(4, "Hex"))
		assert.Equal(t, "intermediate", rows.value(5, "Kind"))
		assert.Equal(t, "101", rows.value(5, "Event"))
		assert.Equal(t, "2", rows.value(5, "Heat"))
		assert.Equal(t, "4", rows.value(5, "Lane"))
		assert.Equal(t, "1", rows.value(5, "Lap"))
		assert.Equal(t, "1", rows.value(5, "Rank"))
		assert.Equal(t, "4", rows.value(5, "LapCount"))
		assert.Equal(t, "30100", rows.value(5, "TimeMS"))
		assert.Empty(t, rows.value(5, "DelayMS"), "a split has no clock values")
	})

	t.Run("corrects the start times like the timing service", func(t *testing.T) {
		t.Parallel()

		content := mustMarshalLines(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Minute), recordedStart(at.Add(time.Minute), time.Minute)...),
			dataEntry(at.Add(5*time.Minute), recordedStart(at.Add(5*time.Minute-2*time.Second), time.Minute)...),
		)

		rows := exportCSV(t, content)

		assert.Equal(t, "start", rows.value(1, "Kind"))
		assert.Equal(t, "0", rows.value(1, "DelayMS"), "the first start sets the baseline")
		assert.Equal(t, "-60000", rows.value(1, "ClockOffsetMS"), "the device runs a minute ahead")
		assert.Equal(t, "2000", rows.value(2, "DelayMS"), "the second start signal arrived late")
		assert.Equal(t, "-60000", rows.value(2, "BaselineMS"))
		assert.Equal(t, "2026-10-18T09:34:58.000+02:00", rows.value(2, "StartTime"))
	})

	t.Run("measures every session against a new baseline", func(t *testing.T) {
		t.Parallel()

		content := mustMarshalLines(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Minute), recordedStart(at.Add(time.Minute), time.Minute)...),
			sessionEntry(at.Add(time.Hour)),
			dataEntry(at.Add(2*time.Hour), recordedStart(at.Add(2*time.Hour), time.Hour)...),
		)

		rows := exportCSV(t, content)

		assert.Equal(t, []string{
			"1",
			"1",
			"2",
			"2",
		}, rows.column("Session"))
		assert.Equal(t, "0", rows.value(3, "DelayMS"), "the other clock offset after the restart is no delay")
	})

	t.Run("starts every session with a new decoder", func(t *testing.T) {
		t.Parallel()

		content := mustMarshalLines(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Second), split[:20]...),
			sessionEntry(at.Add(time.Hour)),
			dataEntry(at.Add(time.Hour+time.Second), split[20:]...),
		)

		rows := exportCSV(t, content)

		assert.NotContains(t, rows.column("Type"), "frame", "parts of different sessions are never combined")
	})

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "rejects an empty recording",
			content: "",
		},
		{
			name:    "rejects a recording that does not start with a session",
			content: mustMarshalLines(t, dataEntry(at, load...)),
		},
		{
			name:    "rejects a broken line within a session",
			content: mustMarshalLines(t, sessionEntry(at)) + "{\n" + mustMarshalLines(t, dataEntry(at, load...)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			err := NewRecordingService(RecordingOptions{}, nil, nil).ExportCSV(strings.NewReader(tt.content), &out)

			assert.Error(t, err)
		})
	}
}

// recordedStart encodes a start frame whose device clock, ahead of the host by ahead, shows the start at.
func recordedStart(at time.Time, ahead time.Duration) []byte {
	device := at.Add(ahead)
	midnight := time.Date(device.Year(), device.Month(), device.Day(), 0, 0, 0, 0, device.Location())
	return omega.Encode(omega.FrameData{
		Kind:      omega.KindStart,
		UsedLanes: []int{4},
		LapCount:  4,
		Event:     101,
		Heat:      2,
		Time:      device.Sub(midnight),
	})
}

// csvRows are the rows of a CSV export without its header.
type csvRows struct {
	header []string
	rows   [][]string
}

// exportCSV exports a recording and returns its rows.
func exportCSV(t *testing.T, content string) csvRows {
	t.Helper()

	var out bytes.Buffer
	require.NoError(t, NewRecordingService(RecordingOptions{}, nil, nil).ExportCSV(strings.NewReader(content), &out))
	records, err := csv.NewReader(&out).ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, records)
	return csvRows{
		header: records[0],
		rows:   records[1:],
	}
}

// value returns the value of a column in a row.
func (r csvRows) value(row int, column string) string {
	return r.rows[row][slices.Index(r.header, column)]
}

// column returns the values of a column in every row.
func (r csvRows) column(column string) []string {
	index := slices.Index(r.header, column)
	values := make([]string, 0, len(r.rows))
	for _, row := range r.rows {
		values = append(values, row[index])
	}
	return values
}

// testRecording returns a recording with one session and one load frame.
func testRecording(t *testing.T) string {
	t.Helper()

	at := time.Now()
	return mustMarshalLines(t, sessionEntry(at), dataEntry(at.Add(time.Second), loadFrame(101, 1)...))
}

// newTestRecordingStore returns a recording store on a local storage in a temporary directory and that directory.
func newTestRecordingStore(t *testing.T) (*store.RecordingStore, string) {
	t.Helper()

	root := t.TempDir()
	storage, err := media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: root,
	})
	require.NoError(t, err)
	return store.NewRecordingStore(storage), root
}

// newStorageRecorder starts a recorder that writes into the recordings folder of the local storage at root
// and waits until its file is open. It returns the recorder and the directory it writes into.
func newStorageRecorder(t *testing.T, root string) (*recording.Writer, string) {
	t.Helper()

	dir := filepath.Join(root, store.RecordingsFolder)
	recorder := newTestRecorder(dir)
	recorder.Start()
	t.Cleanup(recorder.Close)
	require.Eventually(t, func() bool { return recorder.FileName() != "" }, 5*time.Second, time.Millisecond)
	return recorder, dir
}
