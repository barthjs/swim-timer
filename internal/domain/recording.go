package domain

// Recording is a stored recording of the timing data, named with its extension, such as recording_2026-10-18.jsonl.
type Recording struct {
	// Name is the file name with its extension.
	Name string
	// Path is the storage path the replay is configured with.
	Path string
	// Current reports whether the timing service is writing the recording right now.
	Current bool
}
