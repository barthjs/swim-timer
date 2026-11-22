package server

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// newCert generates a self-signed certificate for hosts in a temporary directory.
func newCert(t *testing.T, hosts ...string) (certFile, keyFile string) {
	t.Helper()

	dir := t.TempDir()
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	require.NoError(t, generateSelfSignedCert(certFile, keyFile, hosts))
	return certFile, keyFile
}
