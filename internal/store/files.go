package store

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/media"
)

// maxFileNameBytes is the longest file name common file systems accept, counted in bytes.
const maxFileNameBytes = 255

// reservedFileNames are the device names Windows reserves as file names.
var reservedFileNames = map[string]struct{}{
	"CON":  {},
	"PRN":  {},
	"AUX":  {},
	"NUL":  {},
	"COM1": {},
	"COM2": {},
	"COM3": {},
	"COM4": {},
	"COM5": {},
	"COM6": {},
	"COM7": {},
	"COM8": {},
	"COM9": {},
	"LPT1": {},
	"LPT2": {},
	"LPT3": {},
	"LPT4": {},
	"LPT5": {},
	"LPT6": {},
	"LPT7": {},
	"LPT8": {},
	"LPT9": {},
}

// jsonExtension is the extension of the files jsonFiles keeps its items in.
const jsonExtension = ".json"

// jsonFiles keeps the items of one kind as indented JSON files in a folder of the storage, one file per name.
// Every method holds the lock, so a list never reads a file that a concurrent save is still writing.
type jsonFiles[T any] struct {
	storage media.Storage
	folder  string
	noun    string

	mu sync.Mutex
}

// newJSONFiles creates a jsonFiles for the items in folder, named by noun in error messages.
func newJSONFiles[T any](storage media.Storage, folder, noun string) *jsonFiles[T] {
	return &jsonFiles[T]{
		storage: storage,
		folder:  folder,
		noun:    noun,
	}
}

// names returns the names of all items, sorted.
func (f *jsonFiles[T]) names(ctx context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	names, err := f.find(ctx)
	if err != nil {
		return nil, err
	}
	slices.Sort(names)

	return names, nil
}

// list returns all items in the order of the storage. A file that cannot be read is left out
// and reported with ErrUnreadableFile, together with the items that could be read.
func (f *jsonFiles[T]) list(ctx context.Context) ([]T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	names, err := f.find(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]T, 0, len(names))
	var unreadable []error
	for _, name := range names {
		item, err := f.read(ctx, name)
		if err != nil {
			unreadable = append(unreadable, fmt.Errorf("%w: %w", ErrUnreadableFile, err))
			continue
		}
		items = append(items, *item)
	}

	return items, errors.Join(unreadable...)
}

// get returns the item with the given name, apperr.ErrNotFound if it does not exist,
// or ErrInvalidName for a name that cannot address a file.
func (f *jsonFiles[T]) get(ctx context.Context, name string) (*T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.read(ctx, name)
}

// save writes the item under the given name, replacing an existing one, or fails with ErrInvalidName.
func (f *jsonFiles[T]) save(ctx context.Context, name string, item T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, err := filePath(f.folder, name, jsonExtension)
	if err != nil {
		return err
	}

	// Sorted map keys keep a file stable between saves
	data, err := json.Marshal(item, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return fmt.Errorf("store: encoding %s %q: %w", f.noun, name, err)
	}

	if err := f.storage.Put(ctx, p, bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("store: saving %s %q: %w", f.noun, name, err)
	}

	return nil
}

// delete removes the item with the given name, returns apperr.ErrNotFound if it does not exist,
// or fails with ErrInvalidName.
func (f *jsonFiles[T]) delete(ctx context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, err := filePath(f.folder, name, jsonExtension)
	if err != nil {
		return err
	}

	exists, err := f.storage.Exists(ctx, p)
	if err != nil {
		return fmt.Errorf("store: checking %s %q: %w", f.noun, name, err)
	}
	if !exists {
		return apperr.ErrNotFound
	}

	if err := f.storage.Delete(ctx, p); err != nil {
		return fmt.Errorf("store: deleting %s %q: %w", f.noun, name, err)
	}

	return nil
}

// find returns the names of the files in the folder while the caller holds the lock.
func (f *jsonFiles[T]) find(ctx context.Context) ([]string, error) {
	paths, err := f.storage.List(ctx, f.folder+"/")
	if err != nil {
		return nil, fmt.Errorf("store: listing %ss: %w", f.noun, err)
	}

	names := make([]string, 0, len(paths))
	for _, p := range paths {
		if name, ok := fileName(f.folder, p, jsonExtension); ok {
			names = append(names, name)
		}
	}

	return names, nil
}

// fileName returns the name of the item kept in the file at a storage path, and false for a file the store did not write.
func fileName(folder, p, extension string) (string, bool) {
	base, found := strings.CutPrefix(p, folder+"/")
	if !found {
		return "", false
	}
	name, found := strings.CutSuffix(base, extension)
	if !found {
		return "", false
	}
	if _, err := filePath(folder, name, extension); err != nil {
		return "", false
	}
	return name, true
}

// read returns the item with the given name while the caller holds the lock.
func (f *jsonFiles[T]) read(ctx context.Context, name string) (*T, error) {
	p, err := filePath(f.folder, name, jsonExtension)
	if err != nil {
		return nil, err
	}

	data, err := media.ReadAll(ctx, f.storage, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading %s %q: %w", f.noun, name, err)
	}

	var item T
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("store: decoding %s %q: %w", f.noun, name, err)
	}

	return &item, nil
}

// filePath returns the storage path of the file that keeps the item with the given name in folder,
// or ErrInvalidName for a name that does not address a file directly inside the folder, since fileName only finds those.
// A name that Windows or Linux refuses as a file name is rejected as well, and so is one that would exceed
// maxFileNameBytes, since the file could not be written.
func filePath(folder, name, extension string) (string, error) {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." {
		return "", ErrInvalidName
	}
	// Windows forbids more than Linux, and rejecting it everywhere keeps a stored name valid on every system
	if strings.ContainsFunc(name, forbiddenInFileName) || isReservedFileName(name) {
		return "", ErrInvalidName
	}
	if len(name)+len(extension) > maxFileNameBytes {
		return "", ErrInvalidName
	}
	return folder + "/" + name + extension, nil
}

// forbiddenInFileName reports whether a character cannot appear in a file name on Windows or Linux.
func forbiddenInFileName(r rune) bool {
	return r < 0x20 || r == 0x7F || strings.ContainsRune(`/\<>:"|?*`, r)
}

// isReservedFileName reports whether the part of a name before its first dot names a device on Windows,
// which reserves these names with any extension.
func isReservedFileName(name string) bool {
	base, _, _ := strings.Cut(name, ".")
	_, reserved := reservedFileNames[strings.ToUpper(strings.TrimSpace(base))]
	return reserved
}
