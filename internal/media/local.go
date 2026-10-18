package media

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// tempFilePattern names the files Put and Copy write before they rename them into place.
	tempFilePattern = ".upload-*.tmp"
	// staleTempFileAge is the age from which a temporary file of Put or Copy counts as left behind by a crash
	// or a power loss. No write of a running process takes this long.
	staleTempFileAge = time.Hour
)

// Query parameters of a signed local URL.
const (
	expiresParam   = "expires"
	signatureParam = "signature"
)

const (
	// signingKeySize is the length of a generated signing key, the block size of HMAC-SHA256.
	signingKeySize = 32
)

// localStorage stores objects as files in the local filesystem.
type localStorage struct {
	root       string
	signedBase string
	signingKey []byte
	now        func() time.Time
}

// newLocalStorage creates a localStorage in the local directory of options.
func newLocalStorage(options Options) (*localStorage, error) {
	key := options.SigningKey
	if len(key) == 0 {
		key = make([]byte, signingKeySize)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
	}

	now := options.Clock
	if now == nil {
		now = time.Now
	}

	storage := &localStorage{
		root:       filepath.Clean(options.LocalPath),
		signedBase: strings.TrimRight(options.SignedURLBase, "/"),
		signingKey: key,
		now:        now,
	}

	if err := storage.TestConnection(context.Background()); err != nil {
		return nil, err
	}
	storage.removeStaleTempFiles()

	return storage, nil
}

// removeStaleTempFiles deletes the temporary files of Put and Copy that a crash or a power loss left behind.
// List skips them, so without the cleanup they would fill the disk unnoticed.
// Failures only leave the files in place, so they are ignored.
func (l *localStorage) removeStaleTempFiles() {
	cutoff := l.now().Add(-staleTempFileAge)

	_ = filepath.WalkDir(l.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // A file that cannot be read stays in place, the walk goes on.
		}
		if matched, _ := filepath.Match(tempFilePattern, d.Name()); !matched {
			return nil
		}

		info, err := d.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(p)
		}
		return nil
	})
}

// TestConnection implements Storage. It creates the root directory if it is missing.
func (l *localStorage) TestConnection(_ context.Context) error {
	if err := os.MkdirAll(l.root, 0o755); err != nil {
		return err
	}
	return checkWritable(l.root)
}

// testLocalStorage checks that a local storage could be created in root without creating it:
// root, or its nearest existing parent if root is missing, must be a writable directory.
func testLocalStorage(ctx context.Context, root string) error {
	dir := filepath.Clean(root)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("media: %s is not a directory", dir)
			}
			return checkWritable(dir)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return err
		}
		dir = parent
	}
}

// checkWritable creates and removes a file in dir to check that the storage can write there.
func checkWritable(dir string) error {
	tempFile, err := os.CreateTemp(dir, ".write-check-*")
	if err != nil {
		return err
	}

	tempName := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempName)
		return err
	}

	return os.Remove(tempName)
}

// Exists implements Storage.
func (l *localStorage) Exists(_ context.Context, path string) (bool, error) {
	target, err := l.fullPath(path)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return !info.IsDir(), nil
}

// Get implements Storage.
func (l *localStorage) Get(_ context.Context, path string) (io.ReadCloser, error) {
	target, err := l.fullPath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// os.Open succeeds on a directory too.
	// Report it the same way Exists and Size do.
	if info, err := file.Stat(); err != nil {
		_ = file.Close()
		return nil, err
	} else if info.IsDir() {
		_ = file.Close()
		return nil, ErrNotFound
	}

	return file, nil
}

// Put implements Storage.
func (l *localStorage) Put(_ context.Context, path string, r io.Reader, _ int64) error {
	target, err := l.fullPath(path)
	if err != nil {
		return err
	}
	return writeFile(target, r)
}

// Delete implements Storage.
func (l *localStorage) Delete(_ context.Context, path string) error {
	target, err := l.fullPath(path)
	if err != nil {
		return err
	}

	if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return nil
}

// Size implements Storage.
func (l *localStorage) Size(_ context.Context, path string) (int64, error) {
	target, err := l.fullPath(path)
	if err != nil {
		return 0, err
	}

	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, ErrNotFound
		}
		return 0, err
	}

	if info.IsDir() {
		return 0, ErrInvalidPath
	}

	return info.Size(), nil
}

// List implements Storage. It only walks the directory the prefix names, not the whole root.
func (l *localStorage) List(_ context.Context, prefix string) ([]string, error) {
	var paths []string

	start, err := l.listStart(prefix)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		rel, err := filepath.Rel(l.root, p)
		if err != nil {
			return err
		}

		slashPath := filepath.ToSlash(rel)
		if strings.HasPrefix(slashPath, prefix) {
			paths = append(paths, slashPath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

// listStart returns the directory below which every path starting with prefix lies:
// the prefix itself if it ends with a slash, otherwise its parent directory.
func (l *localStorage) listStart(prefix string) (string, error) {
	dir := prefix
	if !strings.HasSuffix(dir, "/") {
		dir = filepath.ToSlash(filepath.Dir(dir))
	}
	dir = strings.TrimSuffix(dir, "/")

	if dir == "" || dir == "." {
		return l.root, nil
	}
	return l.fullPath(dir)
}

// Copy implements Storage.
func (l *localStorage) Copy(_ context.Context, src, dest string) error {
	srcPath, err := l.fullPath(src)
	if err != nil {
		return err
	}
	destPath, err := l.fullPath(dest)
	if err != nil {
		return err
	}

	sourceFile, err := os.Open(srcPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	return writeFile(destPath, sourceFile)
}

// writeFile writes r to a temporary file next to target and renames it into place,
// so readers never see a partial file and a source that is the target itself stays intact until the rename.
func writeFile(target string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), tempFilePattern)
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}

	if err := os.Rename(tmpName, target); err != nil {
		return err
	}

	return syncDir(filepath.Dir(target))
}

// syncDir flushes a directory to disk, so a rename into it survives a power loss.
func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = handle.Close() }()

	if err := handle.Sync(); err != nil && !errors.Is(err, os.ErrInvalid) {
		return err
	}
	return nil
}

// SignedURL implements Storage. The URL carries the expiry and an HMAC of path and expiry.
func (l *localStorage) SignedURL(_ context.Context, path string, ttl time.Duration) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", ErrInvalidTTL
	}

	objectPath := filepath.ToSlash(filepath.Clean(path))
	expires := l.now().Add(ttl).Unix()

	query := url.Values{}
	query.Set(expiresParam, strconv.FormatInt(expires, 10))
	query.Set(signatureParam, l.signature(objectPath, expires))

	return l.signedBase + "/" + escapePath(objectPath) + "?" + query.Encode(), nil
}

// SignedHandler serves objects of the local driver behind the URLs SignedURL creates.
//
// Mount it below Options.SignedURLBase with the prefix stripped, for example with http.StripPrefix.
// A missing, invalid, or expired signature answers 403. Other drivers serve their signed URLs themselves,
// so for them the handler answers 404.
func SignedHandler(s Storage) http.Handler {
	local, ok := s.(*localStorage)
	if !ok {
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(local.serveSigned)
}

// serveSigned checks the signature and expiry of a request and serves the file it names.
func (l *localStorage) serveSigned(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	objectPath := strings.TrimPrefix(r.URL.Path, "/")
	expires, ok := l.verify(objectPath, r.URL.Query())
	if !ok {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	target, err := l.fullPath(objectPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	file, err := os.Open(target)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// The URL stops working at its expiry, so caches must not keep the response any longer
	maxAge := max(int64(0), expires-l.now().Unix())
	w.Header().Set("Cache-Control", "private, max-age="+strconv.FormatInt(maxAge, 10))
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

// fullPath validates a relative storage path and joins it with the storage root.
func (l *localStorage) fullPath(path string) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}
	return filepath.Join(l.root, filepath.Clean(path)), nil
}

// verify checks the signature and the expiry of a signed URL and returns the expiry.
func (l *localStorage) verify(objectPath string, query url.Values) (int64, bool) {
	expires, err := strconv.ParseInt(query.Get(expiresParam), 10, 64)
	if err != nil || l.now().Unix() > expires {
		return 0, false
	}

	expected := l.signature(objectPath, expires)
	return expires, hmac.Equal([]byte(expected), []byte(query.Get(signatureParam)))
}

// signature returns the HMAC of an object path and its expiry, URL-safe encoded.
func (l *localStorage) signature(objectPath string, expires int64) string {
	mac := hmac.New(sha256.New, l.signingKey)
	_, _ = mac.Write([]byte(objectPath + "\n" + strconv.FormatInt(expires, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// escapePath escapes every segment of a slash-separated path for use in a URL.
func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
