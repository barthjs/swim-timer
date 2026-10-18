package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/store"
)

// recordingExtension is the extension a stored recording ends with.
const recordingExtension = ".jsonl"

// recordingTimeLayout formats the times of the CSV export as ISO 8601 with milliseconds and the UTC offset.
const recordingTimeLayout = "2006-01-02T15:04:05.000Z07:00"

// Row types of the CSV export.
const (
	recordingRowSession   = "session"
	recordingRowMeet      = "meet"
	recordingRowFrame     = "frame"
	recordingRowDiscarded = "discarded"
	recordingRowGap       = "gap"
)

// recordingColumns are the columns of the CSV export.
var recordingColumns = []string{
	"Time", "Session", "Type", "Kind", "MsgType", "MsgKind", "TimeType", "Event", "Heat", "Lane", "Lap", "Rank",
	"LapCount", "UsedLanes", "TimeMS", "ClockOffsetMS", "BaselineMS", "DelayMS", "StartTime", "DroppedBytes", "Hex",
	"Details",
}

// ErrRecordingInUse is returned when the recording to change is replayed or being written.
var ErrRecordingInUse = errors.New("service: the recording is in use")

// RecordingOptions holds the settings of a RecordingService.
type RecordingOptions struct {
	// ReplayPath is the storage path of the recording the simulator replays, empty for none.
	ReplayPath string
}

// RecordingService manages the stored recordings of the timing data and turns them into tables for analysis.
type RecordingService struct {
	options RecordingOptions
	// recorder is the writer of the running recording, nil without recording.
	recorder *recording.Writer
	store    *store.RecordingStore
}

// NewRecordingService creates a RecordingService. The recorder may be nil, and so may the store
// for a service that only converts recordings, see ExportCSV.
func NewRecordingService(options RecordingOptions, recorder *recording.Writer, store *store.RecordingStore) *RecordingService {
	return &RecordingService{
		options:  options,
		recorder: recorder,
		store:    store,
	}
}

// List returns all recordings, sorted by name, and marks the one being written.
func (s *RecordingService) List(ctx context.Context) ([]domain.Recording, error) {
	recordings, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	current := s.currentName()
	for i := range recordings {
		recordings[i].Current = recordings[i].Name == current
	}
	return recordings, nil
}

// Create stores data as a recording under the sanitized filename of the upload with the .jsonl extension,
// replacing a recording of the same name. Data that is not a readable recording and a name that cannot be stored
// fail with a validation error of the file, the name of the recording being written fails with ErrRecordingInUse.
func (s *RecordingService) Create(ctx context.Context, filename string, data []byte) (domain.Recording, error) {
	if !readable(data) {
		return domain.Recording{}, apperr.NewValidationError("file", "invalid")
	}

	name := sanitizeFilename(filename)
	if strings.EqualFold(path.Ext(name), recordingExtension) {
		name = name[:len(name)-len(recordingExtension)]
	}
	if name == "" || name == "." || name == ".." {
		name = fmt.Sprintf("recording-%d", time.Now().UnixNano())
	}
	name += recordingExtension

	if name == s.currentName() {
		return domain.Recording{}, ErrRecordingInUse
	}

	recording, err := s.store.Create(ctx, name, data)
	if errors.Is(err, store.ErrInvalidName) {
		// A name that is still too long after sanitizing cannot be stored
		return domain.Recording{}, apperr.NewValidationError("file", "invalid")
	}
	return recording, err
}

// Delete removes the recording with the given name, or returns apperr.ErrNotFound if it does not exist
// or name cannot address a recording file. The replayed recording and the one being written
// fail with ErrRecordingInUse.
func (s *RecordingService) Delete(ctx context.Context, name string) error {
	recordingPath, err := s.store.Path(name)
	if errors.Is(err, store.ErrInvalidName) {
		return apperr.ErrNotFound
	}
	if err != nil {
		return err
	}

	if recordingPath == path.Clean(s.options.ReplayPath) || name == s.currentName() {
		return ErrRecordingInUse
	}

	return s.store.Delete(ctx, name)
}

// Download opens the recording with the given name for reading, or returns apperr.ErrNotFound if it does not exist
// or name cannot address a recording file. The caller must close it.
func (s *RecordingService) Download(ctx context.Context, name string) (io.ReadCloser, error) {
	recordingPath, err := s.store.Path(name)
	if errors.Is(err, store.ErrInvalidName) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return s.store.Get(ctx, recordingPath)
}

// readable reports whether data is a recording the reader accepts with at least one session.
func readable(data []byte) bool {
	reader := recording.NewReader(bytes.NewReader(data))
	sessions := 0
	for {
		entry, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return sessions > 0
		}
		if err != nil {
			return false
		}
		if entry.Type == recording.TypeSession {
			sessions++
		}
	}
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// sanitizeFilename strips path separators and unsafe characters from a client-supplied name.
// A name that leaves nothing usable sanitizes to "", so callers can fall back or reject it.
func sanitizeFilename(name string) string {
	name = path.Base(strings.TrimSpace(name))
	if name == "." || name == ".." || name == "/" {
		return ""
	}
	return unsafeFilenameChars.ReplaceAllString(name, "-")
}

// currentName returns the name of the recording being written, empty without one.
func (s *RecordingService) currentName() string {
	if s.recorder == nil {
		return ""
	}
	return s.recorder.FileName()
}

// ExportCSV reads a recording from r and writes it to w as a CSV table with one row per frame, discarded block,
// gap, session, and meet line, in the order of the recording. The frames are decoded and their start times corrected
// as the timing service did it while recording: every session starts with a new decoder and a new clock baseline.
func (s *RecordingService) ExportCSV(r io.Reader, w io.Writer) error {
	reader := recording.NewReader(r)
	first, err := reader.Next()
	if errors.Is(err, io.EOF) {
		return errors.New("the recording is empty")
	}
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	if err := writer.Write(recordingColumns); err != nil {
		return err
	}

	export := &recordingExport{
		writer: writer,
		source: &recordingSource{
			reader: reader,
		},
	}
	export.source.onEntry = export.writeEntry

	session := &first
	for session != nil {
		if err := export.exportSession(*session); err != nil {
			return err
		}
		session = export.source.nextSession
	}

	writer.Flush()
	return writer.Error()
}

// recordingExport writes the rows of one CSV export.
type recordingExport struct {
	writer *csv.Writer
	source *recordingSource
	// session counts the sessions from 1.
	session int
	// clock corrects the start times of the current session.
	clock *startClock
}

// exportSession writes a session line and every row up to the next session or the end of the recording.
func (e *recordingExport) exportSession(session recording.Entry) error {
	e.session++
	e.clock = newStartClock(zerolog.Nop(), fixedZone(session.At))
	e.source.nextSession = nil
	if err := e.writeEntry(session); err != nil {
		return err
	}

	var rowErr error
	decoder := omega.NewDecoder(e.source, omega.WithDiscardHandler(func(data []byte) {
		if rowErr == nil {
			rowErr = e.writeRow(e.source.at, recordingRowDiscarded, map[string]string{
				"DroppedBytes": strconv.Itoa(len(data)),
				"Hex":          hexBytes(data),
			})
		}
	}))

	for {
		frame, err := decoder.Next()
		if rowErr != nil {
			return rowErr
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := e.writeFrame(frame, e.source.at); err != nil {
			return err
		}
	}
}

// writeEntry writes the row of a session, meet, or gap line.
func (e *recordingExport) writeEntry(entry recording.Entry) error {
	switch entry.Type {
	case recording.TypeSession:
		return e.writeRow(entry.At, recordingRowSession, map[string]string{
			"Details": fmt.Sprintf("version=%s source=%s port=%s baudRate=%d",
				entry.Version, entry.Source, entry.Port, entry.BaudRate),
		})
	case recording.TypeMeet:
		names := make([]string, 0, len(entry.Files))
		for _, file := range entry.Files {
			names = append(names, file.Name)
		}
		return e.writeRow(entry.At, recordingRowMeet, map[string]string{
			"Details": strings.Join(names, " "),
		})
	case recording.TypeGap:
		return e.writeRow(entry.At, recordingRowGap, map[string]string{
			"DroppedBytes": strconv.FormatInt(entry.DroppedBytes, 10),
		})
	case recording.TypeData:
		// The data reaches the export through the decoder
	}
	return nil
}

// writeFrame writes the row of a frame that was complete at, with the corrected start time of a start frame.
func (e *recordingExport) writeFrame(frame omega.Frame, at time.Time) error {
	fields := frame.Fields()
	values := map[string]string{
		"Kind":     frame.Kind().String(),
		"MsgType":  fields.MsgType,
		"MsgKind":  fields.MsgKind,
		"TimeType": fields.TimeType,
		"Hex":      strings.TrimSpace(frame.Hex()),
	}

	if event, err := frame.Event(); err == nil {
		values["Event"] = strconv.Itoa(event)
	}
	if heat, err := frame.Heat(); err == nil {
		values["Heat"] = strconv.Itoa(heat)
	}
	if lapCount, err := frame.LapCount(); err == nil {
		values["LapCount"] = strconv.Itoa(lapCount)
	}
	if lane, ok := frame.Lane(); ok {
		values["Lane"] = strconv.Itoa(lane)
	}
	if lap, ok := frame.Lap(); ok {
		values["Lap"] = strconv.Itoa(lap)
	}
	if rank, ok := frame.Rank(); ok {
		values["Rank"] = strconv.Itoa(rank)
	}
	if lanes, ok := frame.UsedLanes(); ok {
		values["UsedLanes"] = joinNumbers(lanes)
	}
	if timeMS, ok := frame.TimeMS(); ok {
		values["TimeMS"] = strconv.Itoa(timeMS)
	}

	if frame.Kind() == omega.KindStart {
		start := e.clock.correct(frame, at)
		values["StartTime"] = start.at.Format(recordingTimeLayout)
		if start.valid {
			values["ClockOffsetMS"] = strconv.FormatInt(start.offset.Milliseconds(), 10)
			values["BaselineMS"] = strconv.FormatInt(start.baseline.Milliseconds(), 10)
			values["DelayMS"] = strconv.FormatInt(start.delay.Milliseconds(), 10)
		}
	}

	return e.writeRow(at, recordingRowFrame, values)
}

// joinNumbers joins numbers with spaces.
func joinNumbers(numbers []int) string {
	texts := make([]string, 0, len(numbers))
	for _, number := range numbers {
		texts = append(texts, strconv.Itoa(number))
	}
	return strings.Join(texts, " ")
}

// writeRow writes a row with its time, session, and type, and the values by column name.
func (e *recordingExport) writeRow(at time.Time, rowType string, values map[string]string) error {
	row := make([]string, len(recordingColumns))
	for i, column := range recordingColumns {
		row[i] = values[column]
	}
	row[0] = at.Format(recordingTimeLayout)
	row[1] = strconv.Itoa(e.session)
	row[2] = rowType
	return e.writer.Write(row)
}

// hexBytes returns bytes as space-separated upper-case hex bytes.
func hexBytes(data []byte) string {
	text, _ := recording.HexBytes(data).MarshalText()
	return string(text)
}

// recordingSource hands the data of one session to a decoder, one data line per read, so a frame is complete
// at the time of the data line that holds its last byte. It passes the other lines to onEntry in their order
// and ends the session with io.EOF at the next session line, which it keeps in nextSession.
type recordingSource struct {
	reader  *recording.Reader
	onEntry func(entry recording.Entry) error
	// chunk holds the unread bytes of the current data line, at is when it arrived.
	chunk []byte
	at    time.Time
	// nextSession is the session line that ended the current session, nil at the end of the recording.
	nextSession *recording.Entry
}

// Read returns the bytes of the current data line, reading the next lines when it is used up.
func (s *recordingSource) Read(p []byte) (int, error) {
	for len(s.chunk) == 0 {
		if s.nextSession != nil {
			return 0, io.EOF
		}

		entry, err := s.reader.Next()
		if err != nil {
			return 0, err
		}

		switch entry.Type {
		case recording.TypeSession:
			s.nextSession = &entry
			return 0, io.EOF
		case recording.TypeData:
			s.chunk = entry.Data
			s.at = entry.At
		case recording.TypeMeet, recording.TypeGap:
			if err := s.onEntry(entry); err != nil {
				return 0, err
			}
		}
	}

	n := copy(p, s.chunk)
	s.chunk = s.chunk[n:]
	return n, nil
}
