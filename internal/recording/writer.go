package recording

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

const (
	// maxQueuedBytes bounds the data waiting to be written. It holds about a minute of a device sending
	// without pause at 9600 baud. Data beyond it is dropped, so the reader of the device never waits for the disk.
	maxQueuedBytes = 64 << 10
	// queueSize bounds the number of waiting lines, for a device that delivers its data in many small reads.
	queueSize = 4096
)

// WriterOptions holds the settings of a Writer.
type WriterOptions struct {
	// Dir is the directory of the recording files.
	Dir string
	// Session describes the application, written at the start of every session. Its format is set by the Writer.
	Session Session
}

// queuedEntry is a line waiting to be written.
type queuedEntry struct {
	entry Entry
	// size is the number of data bytes the entry holds in the queue.
	size int64
	// droppedBefore is the number of data bytes dropped between the previous data entry and this one.
	droppedBefore int64
}

// Writer appends a session to the recording file of the current day in a background goroutine.
// Data never waits for the disk: what does not fit into the queue is dropped and reported as a gap.
// Meet lines are never dropped, since the data after them could not be assigned to its meet without them.
type Writer struct {
	logger  zerolog.Logger
	options WriterOptions

	entries chan queuedEntry
	// queuedBytes and droppedBytes count data bytes, see Data.
	queuedBytes  atomic.Int64
	droppedBytes atomic.Int64

	// stop asks run to write the queued entries and return, done reports that it returned.
	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{}
	started  atomic.Bool

	// fileName is the name of the file being written, empty until it is open.
	fileName atomic.Pointer[string]
}

// NewWriter creates a Writer. Start begins the recording.
func NewWriter(logger zerolog.Logger, options WriterOptions) *Writer {
	return &Writer{
		logger:  logger.With().Str("component", "recording").Logger(),
		options: options,
		entries: make(chan queuedEntry, queueSize),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// FileName returns the name of the recording file for the day of t in its location.
func FileName(t time.Time) string {
	return "recording_" + t.Format(time.DateOnly) + ".jsonl"
}

// Start opens the recording file of the current day in a background goroutine, starts a session in it,
// and writes the queued entries until Close is called. If the file cannot be opened,
// it logs the error and discards the entries, so the timing keeps running without the recording.
// It may only be called once.
func (w *Writer) Start() {
	if !w.started.CompareAndSwap(false, true) {
		w.logger.Error().Msg("recording writer is already running")
		return
	}
	go w.run()
}

// run writes the recording, see Start.
func (w *Writer) run() {
	defer close(w.done)

	file, err := w.openFile(time.Now())
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to open the recording file, the timing runs without recording")
		w.discardUntilStopped()
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			w.logger.Error().Err(err).Msg("failed to close the recording file")
		}
	}()

	encoder := json.NewEncoder(file)
	session := w.options.Session
	session.Format = FormatVersion
	w.write(encoder, Entry{
		Type:    TypeSession,
		At:      time.Now(),
		Session: &session,
	})

	for {
		select {
		case queued := <-w.entries:
			w.writeQueued(encoder, queued)
		case <-w.stop:
			w.drain(encoder)
			return
		}
	}
}

// Close asks the writer to write the queued entries and waits until they are written.
// Called before Start, it only keeps the writer from waiting for further entries.
func (w *Writer) Close() {
	w.stopOnce.Do(func() { close(w.stop) })
	if w.started.Load() {
		<-w.done
	}
}

// FileName returns the name of the recording file being written, or empty while no file is open.
func (w *Writer) FileName() string {
	name := w.fileName.Load()
	if name == nil {
		return ""
	}
	return *name
}

// Reader returns a reader that passes every read of r on unchanged and records its bytes at the time of the read.
func (w *Writer) Reader(r io.Reader) io.Reader {
	return &recordingReader{
		reader: r,
		writer: w,
	}
}

// Data queues the bytes of one read from the device. It never blocks: if the queue is full,
// the bytes are dropped and the next data line that fits is preceded by a gap line.
// The bytes are copied, so the caller may reuse its buffer.
func (w *Writer) Data(at time.Time, data []byte) {
	size := int64(len(data))
	if size == 0 {
		return
	}

	if w.queuedBytes.Add(size) > maxQueuedBytes {
		w.queuedBytes.Add(-size)
		w.droppedBytes.Add(size)
		return
	}

	queued := queuedEntry{
		entry: Entry{
			Type: TypeData,
			At:   at,
			Data: HexBytes(append([]byte(nil), data...)),
		},
		size:          size,
		droppedBefore: w.droppedBytes.Swap(0),
	}
	select {
	case w.entries <- queued:
	default:
		w.queuedBytes.Add(-size)
		w.droppedBytes.Add(size + queued.droppedBefore)
	}
}

// Meet queues the meet files in use from now on. It waits for room in the queue,
// unless the writer is stopping, so a meet line is never dropped while the recording runs.
func (w *Writer) Meet(at time.Time, files []MeetFile) {
	queued := queuedEntry{
		entry: Entry{
			Type:  TypeMeet,
			At:    at,
			Files: files,
		},
	}
	select {
	case w.entries <- queued:
	case <-w.stop:
	}
}

// openFile opens the recording file of the day of now for appending, creating it and its directory if missing.
// A file that ends inside a line, since the application stopped while writing it, gets a line break first,
// so the new session starts on a line of its own.
func (w *Writer) openFile(now time.Time) (*os.File, error) {
	if err := os.MkdirAll(w.options.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating the recording directory %q: %w", w.options.Dir, err)
	}

	path := filepath.Join(w.options.Dir, FileName(now))
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening the recording file %q: %w", path, err)
	}

	if err := terminateLastLine(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("preparing the recording file %q: %w", path, err)
	}

	w.logger.Info().Str("file", path).Msg("recording the timing data")
	name := filepath.Base(path)
	w.fileName.Store(&name)
	return file, nil
}

// terminateLastLine appends a line break to a file whose last byte is not one.
func terminateLastLine(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return nil
	}

	last := make([]byte, 1)
	if _, err := file.ReadAt(last, info.Size()-1); err != nil {
		return err
	}
	if last[0] == '\n' {
		return nil
	}

	_, err = file.Write([]byte{'\n'})
	return err
}

// discardUntilStopped empties the queue until Close is called, so Meet never waits for a writer without a file.
func (w *Writer) discardUntilStopped() {
	for {
		select {
		case queued := <-w.entries:
			w.queuedBytes.Add(-queued.size)
		case <-w.stop:
			return
		}
	}
}

// drain writes the entries still queued and a final gap line for data dropped after the last of them.
func (w *Writer) drain(encoder *json.Encoder) {
	for {
		select {
		case queued := <-w.entries:
			w.writeQueued(encoder, queued)
		default:
			if dropped := w.droppedBytes.Swap(0); dropped > 0 {
				w.write(encoder, Entry{
					Type:         TypeGap,
					At:           time.Now(),
					DroppedBytes: dropped,
				})
			}
			return
		}
	}
}

// writeQueued writes a queued entry, preceded by a gap line for the data dropped before it.
func (w *Writer) writeQueued(encoder *json.Encoder, queued queuedEntry) {
	w.queuedBytes.Add(-queued.size)

	if queued.droppedBefore > 0 {
		w.write(encoder, Entry{
			Type:         TypeGap,
			At:           queued.entry.At,
			DroppedBytes: queued.droppedBefore,
		})
	}
	w.write(encoder, queued.entry)
}

// write encodes an entry as one line. A failed write is logged, and the recording continues with the next entry.
func (w *Writer) write(encoder *json.Encoder, entry Entry) {
	if err := encoder.Encode(entry); err != nil {
		w.logger.Warn().Err(err).Str("type", string(entry.Type)).Msg("failed to write a recording line")
	}
}

// recordingReader records every read of the reader it wraps, see Writer.Reader.
type recordingReader struct {
	reader io.Reader
	writer *Writer
}

// Read reads from the wrapped reader and queues the bytes read for the recording.
func (r *recordingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.writer.Data(time.Now(), p[:n])
	}
	return n, err
}
