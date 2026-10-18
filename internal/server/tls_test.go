package server

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/appdirs"
)

func TestServer_certFiles(t *testing.T) {
	// The certificate search starts in the working directory, which the subtests change, so it cannot run in parallel

	t.Run("returns the configured files", func(t *testing.T) {
		t.Parallel()

		srv := New(zerolog.Nop(), Options{
			SSLCertFile: "/path/cert.pem",
			SSLKeyFile:  "/path/key.pem",
		}, nil)

		certFile, keyFile, err := srv.certFiles()

		require.NoError(t, err)
		assert.Equal(t, "/path/cert.pem", certFile)
		assert.Equal(t, "/path/key.pem", keyFile)
	})

	t.Run("finds a certificate in the working directory", func(t *testing.T) {
		t.Chdir(t.TempDir())
		require.NoError(t, generateSelfSignedCert(
			filepath.Join("certs", "cert.pem"),
			filepath.Join("certs", "key.pem"),
			[]string{"localhost", "127.0.0.1", "::1", "127.0.0.2"},
		))
		srv := New(zerolog.Nop(), Options{
			Host:     "127.0.0.2",
			CertDirs: emptyRoots(t),
		}, nil)

		certFile, keyFile, err := srv.certFiles()

		require.NoError(t, err)
		assert.Equal(t, filepath.Join("certs", "cert.pem"), certFile)
		assert.Equal(t, filepath.Join("certs", "key.pem"), keyFile)
	})

	t.Run("generates a certificate in the system directory", func(t *testing.T) {
		t.Chdir(t.TempDir())
		roots := emptyRoots(t)
		srv := New(zerolog.Nop(), Options{
			Host:     "127.0.0.1",
			AppName:  "example-app",
			CertDirs: roots,
		}, nil)

		certFile, keyFile, err := srv.certFiles()

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(roots.System, "example-app", "certs", "cert.pem"), certFile)
		cert, err := loadCert(certFile, keyFile)
		require.NoError(t, err)
		assert.True(t, certCovers(cert, []string{"localhost", "127.0.0.1"}))
	})

	t.Run("reuses the generated certificate", func(t *testing.T) {
		t.Chdir(t.TempDir())
		srv := New(zerolog.Nop(), Options{
			Host:     "127.0.0.1",
			AppName:  "example-app",
			CertDirs: emptyRoots(t),
		}, nil)
		certFile, _, err := srv.certFiles()
		require.NoError(t, err)

		again, _, err := srv.certFiles()

		require.NoError(t, err)
		assert.Equal(t, certFile, again)
	})

	t.Run("falls back to a writable directory", func(t *testing.T) {
		t.Chdir(t.TempDir())
		// The system root is a file, so no directory can be created below it
		blocked := filepath.Join(t.TempDir(), "blocked")
		require.NoError(t, os.WriteFile(blocked, []byte("x"), 0o644))
		roots := appdirs.Roots{
			User:   t.TempDir(),
			System: blocked,
		}
		srv := New(zerolog.Nop(), Options{
			AppName:  "example-app",
			CertDirs: roots,
		}, nil)

		certFile, _, err := srv.certFiles()

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(roots.User, "example-app", "certs", "cert.pem"), certFile)
	})

	t.Run("rejects a setup without a writable directory", func(t *testing.T) {
		blocked := filepath.Join(t.TempDir(), "blocked")
		require.NoError(t, os.WriteFile(blocked, []byte("x"), 0o644))
		t.Chdir(filepath.Dir(blocked))
		require.NoError(t, os.WriteFile("certs", []byte("x"), 0o644))
		srv := New(zerolog.Nop(), Options{
			CertDirs: appdirs.Roots{
				User:   blocked,
				System: blocked,
			},
		}, nil)

		_, _, err := srv.certFiles()

		assert.Error(t, err)
	})
}

func TestServer_certHosts(t *testing.T) {
	t.Parallel()

	t.Run("adds only a specific host", func(t *testing.T) {
		t.Parallel()

		srv := New(zerolog.Nop(), Options{
			Host: "example.test",
		}, nil)

		hosts := srv.certHosts()

		assert.Equal(t, []string{"localhost", "127.0.0.1", "::1", "example.test"}, hosts)
	})

	t.Run("adds the local addresses for all interfaces", func(t *testing.T) {
		t.Parallel()

		srv := New(zerolog.Nop(), Options{
			Host: "0.0.0.0",
		}, nil)

		hosts := srv.certHosts()

		assert.Equal(t, []string{"localhost", "127.0.0.1", "::1"}, hosts[:3])
		assert.Equal(t, append(hosts[:3:3], localIPs()...), hosts)
	})
}

func TestServer_ensureSelfSignedCert(t *testing.T) {
	t.Parallel()

	srv := New(zerolog.Nop(), Options{}, nil)

	t.Run("creates a missing certificate", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		certFile := filepath.Join(dir, "cert.pem")
		keyFile := filepath.Join(dir, "key.pem")

		err := srv.ensureSelfSignedCert(certFile, keyFile, []string{"localhost"})

		require.NoError(t, err)
		_, err = loadCert(certFile, keyFile)
		assert.NoError(t, err)
	})

	t.Run("keeps a certificate that covers every host", func(t *testing.T) {
		t.Parallel()

		certFile, keyFile := newCert(t, "localhost", "127.0.0.1")
		before, err := os.ReadFile(certFile)
		require.NoError(t, err)

		err = srv.ensureSelfSignedCert(certFile, keyFile, []string{"localhost"})

		require.NoError(t, err)
		after, err := os.ReadFile(certFile)
		require.NoError(t, err)
		assert.Equal(t, before, after)
	})

	t.Run("replaces a certificate that misses a host", func(t *testing.T) {
		t.Parallel()

		certFile, keyFile := newCert(t, "localhost")

		err := srv.ensureSelfSignedCert(certFile, keyFile, []string{"localhost", "10.0.0.1"})

		require.NoError(t, err)
		cert, err := loadCert(certFile, keyFile)
		require.NoError(t, err)
		assert.True(t, certCovers(cert, []string{"localhost", "10.0.0.1"}))
	})
}

func TestResolveCertDir(t *testing.T) {
	// The test changes the working directory, which the certificate search starts in, so it cannot run in parallel

	t.Run("returns nothing without a certificate", func(t *testing.T) {
		t.Chdir(t.TempDir())
		roots := emptyRoots(t)

		dir := resolveCertDir(roots, "example-app")

		assert.Empty(t, dir)
	})
}

// emptyRoots returns config roots in fresh temporary directories.
func emptyRoots(t *testing.T) appdirs.Roots {
	t.Helper()
	return appdirs.Roots{
		User:   t.TempDir(),
		System: t.TempDir(),
	}
}

func TestDefaultCertDir(t *testing.T) {
	// The test changes the working directory, which the certificate search starts in, so it cannot run in parallel

	t.Run("rejects a setup without a writable directory", func(t *testing.T) {
		blocked := filepath.Join(t.TempDir(), "blocked")
		require.NoError(t, os.WriteFile(blocked, []byte("x"), 0o644))
		// The working directory is the blocked file's parent with a file named "certs" in it
		t.Chdir(filepath.Dir(blocked))
		require.NoError(t, os.WriteFile("certs", []byte("x"), 0o644))

		_, err := defaultCertDir(appdirs.Roots{
			User:   blocked,
			System: blocked,
		}, "example-app")

		assert.Error(t, err)
	})
}

func TestLocalIPs(t *testing.T) {
	t.Parallel()

	t.Run("excludes the loopback addresses", func(t *testing.T) {
		t.Parallel()

		ips := localIPs()

		for _, ip := range ips {
			parsed := net.ParseIP(ip)
			require.NotNil(t, parsed, ip)
			assert.False(t, parsed.IsLoopback(), ip)
		}
	})
}

func TestLoadCert(t *testing.T) {
	t.Parallel()

	certFile, _ := newCert(t, "localhost")
	_, otherKey := newCert(t, "localhost")
	garbage := filepath.Join(t.TempDir(), "garbage.pem")
	require.NoError(t, os.WriteFile(garbage, []byte("not a certificate"), 0o600))

	tests := []struct {
		name     string
		certFile string
		keyFile  string
	}{
		{
			name:     "rejects missing files",
			certFile: filepath.Join(t.TempDir(), "cert.pem"),
			keyFile:  filepath.Join(t.TempDir(), "key.pem"),
		},
		{
			name:     "rejects the key of another certificate",
			certFile: certFile,
			keyFile:  otherKey,
		},
		{
			name:     "rejects malformed files",
			certFile: garbage,
			keyFile:  garbage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := loadCert(tt.certFile, tt.keyFile)

			assert.Error(t, err)
		})
	}
}

func TestCertCovers(t *testing.T) {
	t.Parallel()

	certFile, keyFile := newCert(t, "localhost", "127.0.0.1")
	cert, err := loadCert(certFile, keyFile)
	require.NoError(t, err)

	tests := []struct {
		name  string
		hosts []string
		want  bool
	}{
		{
			name:  "covers the hosts of the certificate",
			hosts: []string{"localhost", "127.0.0.1"},
			want:  true,
		},
		{
			name:  "covers an empty list",
			hosts: nil,
			want:  true,
		},
		{
			name:  "misses another address",
			hosts: []string{"localhost", "10.0.0.1"},
			want:  false,
		},
		{
			name:  "misses another name",
			hosts: []string{"example.test"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			covers := certCovers(cert, tt.hosts)

			assert.Equal(t, tt.want, covers)
		})
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	t.Parallel()

	t.Run("creates the certificate for the hosts", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		certFile := filepath.Join(dir, "cert.pem")
		keyFile := filepath.Join(dir, "key.pem")

		err := generateSelfSignedCert(certFile, keyFile, []string{"localhost", "127.0.0.1", "example.test"})

		require.NoError(t, err)
		cert, err := loadCert(certFile, keyFile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", cert.Subject.CommonName)
		assert.ElementsMatch(t, []string{"localhost", "example.test"}, cert.DNSNames)
		require.Len(t, cert.IPAddresses, 1)
		assert.Equal(t, "127.0.0.1", cert.IPAddresses[0].String())
		assert.WithinDuration(t, time.Now().Add(certValidity), cert.NotAfter, time.Minute)
		keyInfo, err := os.Stat(keyFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), keyInfo.Mode().Perm(), "the private key is not readable by others")
	})

	t.Run("creates missing directories", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "nested", "certs")

		err := generateSelfSignedCert(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"), []string{"localhost"})

		require.NoError(t, err)
		_, err = loadCert(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
		assert.NoError(t, err)
	})

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	dir := t.TempDir()

	tests := []struct {
		name     string
		certFile string
		keyFile  string
	}{
		{
			name:     "rejects a key directory below a file",
			certFile: filepath.Join(dir, "cert.pem"),
			keyFile:  filepath.Join(file, "key.pem"),
		},
		{
			name:     "rejects a certificate directory below a file",
			certFile: filepath.Join(file, "cert.pem"),
			keyFile:  filepath.Join(dir, "key.pem"),
		},
		{
			name:     "rejects a key path that is a directory",
			certFile: filepath.Join(dir, "cert.pem"),
			keyFile:  dir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := generateSelfSignedCert(tt.certFile, tt.keyFile, []string{"localhost"})

			assert.Error(t, err)
		})
	}
}

// newCert generates a self-signed certificate for hosts in a temporary directory.
func newCert(t *testing.T, hosts ...string) (certFile, keyFile string) {
	t.Helper()

	dir := t.TempDir()
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	require.NoError(t, generateSelfSignedCert(certFile, keyFile, hosts))
	return certFile, keyFile
}
