// Package update replaces the running binary with the matching archive of the latest GitHub release.
// It only replaces binaries it may own, so a binary from a package manager or inside a container is left alone.
package update

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

const (
	defaultAPIURL          = "https://api.github.com"
	defaultMaxDownloadSize = 256 << 20
	defaultTimeout         = 5 * time.Minute
	checksumsName          = "checksums.txt"
	// maxChecksumsSize bounds the checksums file, which lists one line per release asset.
	maxChecksumsSize = 1 << 20
	// maxReleaseSize bounds the release metadata returned by the API.
	maxReleaseSize = 4 << 20
)

// containerFiles are the files Docker and Podman create inside a container.
var containerFiles = []string{"/.dockerenv", "/run/.containerenv"}

var (
	// ErrDevelopmentBuild is returned when the build has no release version or architecture.
	ErrDevelopmentBuild = errors.New("update: development builds cannot update themselves")
	// ErrContainer is returned when the process runs inside a container, which is updated by pulling a new image.
	ErrContainer = errors.New("update: running in a container, pull a new image instead")
	// ErrPackageManaged is returned when a package manager owns the binary.
	ErrPackageManaged = errors.New("update: the binary is managed by the package manager")
	// ErrNotWritable is returned when the directory of the binary cannot be written.
	ErrNotWritable = errors.New("update: the directory of the binary is not writable")
	// ErrNoAsset is returned when a release has no archive for this platform.
	ErrNoAsset = errors.New("update: the release has no archive for this platform")
)

// Options holds the settings of an Updater.
type Options struct {
	// Repository is the GitHub repository as "owner/name".
	Repository string
	// Binary is the file name of the binary inside the release archives, without extension.
	Binary string
	// Version is the running version, such as "v1.2.3".
	Version string
	// Arch is the architecture part of the archive names, such as "amd64" or "armv7".
	Arch string
	// APIURL is the base URL of the GitHub API, https://api.github.com if empty.
	APIURL string
	// Token authenticates the API requests if set, which raises the rate limit.
	Token string
	// HTTPClient sends the requests, a client with a timeout of five minutes if nil.
	HTTPClient *http.Client
	// MaxDownloadSize bounds the downloaded archive, 256 MiB if zero.
	MaxDownloadSize int64
}

// Release is the latest release of the repository.
type Release struct {
	// Version is the tag of the release.
	Version string
	// URL is the web page of the release.
	URL string
	// Newer reports whether the release is newer than the running version.
	Newer bool

	assetURL     string
	assetName    string
	checksumsURL string
}

// Updater checks for and applies releases.
type Updater struct {
	options Options
	client  *http.Client

	// mu guards replaced, since Apply and Restart may run on different goroutines.
	mu sync.Mutex
	// replaced is the binary the last Apply replaced, resolved before the replacement renamed the running file.
	replaced string

	// The fields below locate the environment. Tests replace them.
	executable    func() (string, error)
	getenv        func(string) string
	containerFile []string
	dpkgInfoDir   string
}

// New creates an Updater.
func New(options Options) *Updater {
	if options.APIURL == "" {
		options.APIURL = defaultAPIURL
	}
	if options.MaxDownloadSize == 0 {
		options.MaxDownloadSize = defaultMaxDownloadSize
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: defaultTimeout,
		}
	}

	return &Updater{
		options:       options,
		client:        client,
		executable:    os.Executable,
		getenv:        os.Getenv,
		containerFile: containerFiles,
		dpkgInfoDir:   "/var/lib/dpkg/info",
	}
}

// Version returns the running version.
func (u *Updater) Version() string {
	return u.options.Version
}

// githubRelease is the part of the GitHub release response the updater reads.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Check fetches the latest release and compares it with the running version.
func (u *Updater) Check(ctx context.Context) (Release, error) {
	url := strings.TrimRight(u.options.APIURL, "/") + "/repos/" + u.options.Repository + "/releases/latest"
	response, err := u.get(ctx, url, "application/vnd.github+json")
	if err != nil {
		return Release{}, err
	}
	defer func() { _ = response.Body.Close() }()

	var latest githubRelease
	if err := json.UnmarshalRead(io.LimitReader(response.Body, maxReleaseSize), &latest); err != nil {
		return Release{}, fmt.Errorf("update: reading the release: %w", err)
	}

	release := Release{
		Version:   latest.TagName,
		URL:       latest.HTMLURL,
		Newer:     semver.IsValid(u.options.Version) && semver.Compare(latest.TagName, u.options.Version) > 0,
		assetName: u.assetName(),
	}
	for _, asset := range latest.Assets {
		switch asset.Name {
		case release.assetName:
			release.assetURL = asset.URL
		case checksumsName:
			release.checksumsURL = asset.URL
		}
	}
	return release, nil
}

// Supported reports why the running binary cannot replace itself, or nil if it can.
func (u *Updater) Supported() error {
	_, err := u.binary()
	return err
}

// binary returns the resolved path of the running binary if it can replace itself, see Supported.
func (u *Updater) binary() (string, error) {
	if !semver.IsValid(u.options.Version) || u.options.Arch == "" {
		return "", ErrDevelopmentBuild
	}
	if inContainer(u.getenv, u.containerFile) {
		return "", ErrContainer
	}

	executable, err := u.executable()
	if err != nil {
		return "", fmt.Errorf("update: locating the binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	if u.packageManaged(executable) {
		return "", ErrPackageManaged
	}
	if !dirWritable(filepath.Dir(executable)) {
		return "", ErrNotWritable
	}
	return executable, nil
}

// inContainer reports whether getenv names a container or one of the marker files exists.
func inContainer(getenv func(string) string, markers []string) bool {
	if getenv("container") != "" {
		return true
	}
	for _, path := range markers {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// packageManaged reports whether dpkg lists executable as a file of the package named after the binary.
func (u *Updater) packageManaged(executable string) bool {
	list, err := os.ReadFile(filepath.Join(u.dpkgInfoDir, u.options.Binary+".list"))
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(list), "\n") {
		if strings.TrimSpace(line) == executable {
			return true
		}
	}
	return false
}

// dirWritable reports whether a new file can be created in dir, which replacing the binary requires.
func dirWritable(dir string) bool {
	probe, err := os.CreateTemp(dir, ".update-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return true
}

// assetName returns the archive name of this platform, following the naming of the release builds.
func (u *Updater) assetName() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return u.options.Binary + "_" + runtime.GOOS + "_" + u.options.Arch + ext
}

// binaryName returns the file name of the binary inside an archive.
func (u *Updater) binaryName() string {
	if runtime.GOOS == "windows" {
		return u.options.Binary + ".exe"
	}
	return u.options.Binary
}

// get sends an authenticated GET request and fails on any status but 200.
func (u *Updater) get(ctx context.Context, url, accept string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", accept)
	if u.options.Token != "" {
		request.Header.Set("Authorization", "Bearer "+u.options.Token)
	}

	response, err := u.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("update: requesting %s: %w", url, err)
	}
	if response.StatusCode == http.StatusOK {
		return response, nil
	}

	_ = response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("update: %s answered %d, the rate limit of the GitHub API may be exhausted", url, response.StatusCode)
	}
	return nil, fmt.Errorf("update: %s answered %d", url, response.StatusCode)
}
