package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/minio/selfupdate"
)

// Apply downloads the archive of release, verifies it against the checksums of the release, and replaces the binary
// with the one from the archive. The running process keeps the old binary until it restarts, see Restart.
// A canceled ctx stops Apply until the replacement starts, which then runs to its end,
// so a binary is never left half replaced.
func (u *Updater) Apply(ctx context.Context, release Release) error {
	target, err := u.binary()
	if err != nil {
		return err
	}
	if release.assetURL == "" || release.checksumsURL == "" {
		return ErrNoAsset
	}

	expected, err := u.checksum(ctx, release)
	if err != nil {
		return err
	}

	archive, err := u.download(ctx, release.assetURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = archive.Close()
		_ = os.Remove(archive.Name())
	}()

	if err := verify(archive, expected); err != nil {
		return err
	}

	binary, closeBinary, err := u.extract(archive)
	if err != nil {
		return err
	}
	defer closeBinary()

	// Verifying and extracting do not watch ctx, a stop during them leaves the binary untouched here
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("update: %w", err)
	}

	// The path is resolved before the replacement, which renames the running file, so Restart finds the new binary
	if err := selfupdate.Apply(binary, selfupdate.Options{
		TargetPath: target,
	}); err != nil {
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			return fmt.Errorf("update: replacing the binary failed and the old one could not be restored: %w", rollbackErr)
		}
		return fmt.Errorf("update: replacing the binary: %w", err)
	}

	u.mu.Lock()
	u.replaced = target
	u.mu.Unlock()
	return nil
}

// restartPath returns the binary Restart starts: the one the last Apply replaced, or the running binary without one.
// After a replacement, os.Executable on Linux reports the renamed old file, so the path Apply resolved is used.
func (u *Updater) restartPath() (string, error) {
	u.mu.Lock()
	replaced := u.replaced
	u.mu.Unlock()

	if replaced != "" {
		return replaced, nil
	}
	return u.executable()
}

// checksum reads the expected SHA-256 of the release archive from the checksums file of the release.
func (u *Updater) checksum(ctx context.Context, release Release) (string, error) {
	response, err := u.get(ctx, release.checksumsURL, "application/octet-stream")
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()

	scanner := bufio.NewScanner(io.LimitReader(response.Body, maxChecksumsSize))
	for scanner.Scan() {
		sum, name, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "  ")
		if ok && strings.TrimSpace(name) == release.assetName {
			return strings.ToLower(sum), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("update: reading the checksums: %w", err)
	}
	return "", fmt.Errorf("update: the checksums do not list %s", release.assetName)
}

// download streams url into a temporary file, failing once it exceeds the download limit.
func (u *Updater) download(ctx context.Context, url string) (*os.File, error) {
	response, err := u.get(ctx, url, "application/octet-stream")
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	file, err := os.CreateTemp("", "update-*")
	if err != nil {
		return nil, err
	}

	// One byte more than the limit tells a download at the limit from a larger one
	written, err := io.Copy(file, io.LimitReader(response.Body, u.options.MaxDownloadSize+1))
	if err == nil && written > u.options.MaxDownloadSize {
		err = fmt.Errorf("update: the archive is larger than %d bytes", u.options.MaxDownloadSize)
	}
	if err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

// verify compares the SHA-256 of file with the expected hex digest.
func verify(file *os.File, expected string) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if actual := hex.EncodeToString(hash.Sum(nil)); actual != expected {
		return fmt.Errorf("update: checksum mismatch, expected %s, got %s", expected, actual)
	}
	return nil
}

// extract returns a reader of the binary inside the archive and a function that releases it.
func (u *Updater) extract(archive *os.File) (io.Reader, func(), error) {
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return nil, nil, err
	}
	if runtime.GOOS == "windows" {
		return extractZip(archive, u.binaryName())
	}
	return extractTarGz(archive, u.binaryName())
}

// extractTarGz returns a reader of the file name in a gzip-compressed tar archive.
func extractTarGz(archive *os.File, name string) (io.Reader, func(), error) {
	gz, err := gzip.NewReader(archive)
	if err != nil {
		return nil, nil, fmt.Errorf("update: reading the archive: %w", err)
	}

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			_ = gz.Close()
			return nil, nil, fmt.Errorf("update: the archive does not contain %s", name)
		}
		if err != nil {
			_ = gz.Close()
			return nil, nil, fmt.Errorf("update: reading the archive: %w", err)
		}
		if header.Typeflag == tar.TypeReg && path.Base(header.Name) == name {
			return reader, func() { _ = gz.Close() }, nil
		}
	}
}

// extractZip returns a reader of the file name in a zip archive.
func extractZip(archive *os.File, name string) (io.Reader, func(), error) {
	info, err := archive.Stat()
	if err != nil {
		return nil, nil, err
	}
	reader, err := zip.NewReader(archive, info.Size())
	if err != nil {
		return nil, nil, fmt.Errorf("update: reading the archive: %w", err)
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() || path.Base(file.Name) != name {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("update: reading the archive: %w", err)
		}
		return opened, func() { _ = opened.Close() }, nil
	}
	return nil, nil, fmt.Errorf("update: the archive does not contain %s", name)
}
