package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/barthjs/swim-timer/internal/appdirs"
)

const (
	certsDirName = "certs"
	// certValidity matches the CA/Browser Forum's cap on publicly trusted certificate lifetimes,
	// so the same value stays reasonable if this certificate is ever inspected by tooling that enforces it.
	certValidity = 397 * 24 * time.Hour
)

// certFiles resolves the certificate/key pair to serve over TLS. Explicit paths take precedence.
// Otherwise, a self-signed certificate is generated and cached in the first directory that already holds one,
// or else the first writable one (see resolveCertDir and defaultCertDir).
func (s *Server) certFiles() (certFile, keyFile string, err error) {
	if s.options.SSLCertFile != "" {
		return s.options.SSLCertFile, s.options.SSLKeyFile, nil
	}

	roots := s.options.CertDirs
	if roots == (appdirs.Roots{}) {
		roots = appdirs.ConfigRoots()
	}

	dir := resolveCertDir(roots, s.options.AppName)
	if dir == "" {
		dir, err = defaultCertDir(roots, s.options.AppName)
		if err != nil {
			return "", "", err
		}
	}

	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	if err := s.ensureSelfSignedCert(certFile, keyFile, s.certHosts()); err != nil {
		return "", "", err
	}

	return certFile, keyFile, nil
}

// certHosts lists the hostnames and IPs a locally generated certificate must cover.
// Binding to one specific host only requires that host.
// Binding to all interfaces also covers every local network address.
func (s *Server) certHosts() []string {
	hosts := []string{"localhost", "127.0.0.1", "::1"}

	if allInterfaces(s.options.Host) {
		return append(hosts, localIPs()...)
	}
	return append(hosts, s.options.Host)
}

// ensureSelfSignedCert makes certFile/keyFile hold a valid, unexpired self-signed certificate that covers
// every entry in hosts (a mix of hostnames and IP addresses). An existing pair that already covers all hosts
// and has not expired is left untouched. Otherwise, a new ECDSA P-256 key pair and certificate are generated
// and written in its place.
func (s *Server) ensureSelfSignedCert(certFile, keyFile string, hosts []string) error {
	if cert, err := loadCert(certFile, keyFile); err == nil && certCovers(cert, hosts) {
		s.logger.Debug().Str("cert", certFile).Msg("reusing existing self-signed TLS certificate")
		return nil
	}

	s.logger.Info().Str("cert", certFile).Strs("hosts", hosts).Msg("generating self-signed TLS certificate")
	return generateSelfSignedCert(certFile, keyFile, hosts)
}

// resolveCertDir reports the first of the candidate certificate directories that already contains a cert.pem
// and key.pem pair, or empty if none does, in which case a caller generating a new certificate must pick a location
// itself (see defaultCertDir). Uses the same search order as an existing config file (see appdirs.Roots.SearchDirs):
// working directory, then the user config directory, then the system config directory.
func resolveCertDir(roots appdirs.Roots, appDirName string) string {
	for _, dir := range roots.SearchDirs(appDirName, certsDirName) {
		if _, err := os.Stat(filepath.Join(dir, "cert.pem")); err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, "key.pem")); err != nil {
			continue
		}
		return dir
	}

	return ""
}

// defaultCertDir picks a location for a newly generated certificate:
// the same candidate directories resolveCertDir searches, tried in order until one is writable.
func defaultCertDir(roots appdirs.Roots, appDirName string) (string, error) {
	candidates := roots.WriteDirs(appDirName, certsDirName)
	for _, dir := range candidates {
		if appdirs.DirWritable(dir) {
			return dir, nil
		}
	}

	return "", fmt.Errorf("no writable location found for generated TLS certificates (tried %v)", candidates)
}

// localIPs returns every non-loopback IP address bound to a local network interface.
func localIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	var ips []string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ips = append(ips, ipNet.IP.String())
	}

	return ips
}

// loadCert reads and parses an existing certificate/key pair,
// failing if either file is missing, unreadable, malformed, or expired.
func loadCert(certFile, keyFile string) (*x509.Certificate, error) {
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, err
	}

	if time.Now().After(cert.NotAfter) {
		return nil, fmt.Errorf("certificate %q expired on %s", certFile, cert.NotAfter)
	}

	return cert, nil
}

// certCovers reports whether a cert is valid for every host in hosts.
func certCovers(cert *x509.Certificate, hosts []string) bool {
	for _, host := range hosts {
		if cert.VerifyHostname(host) != nil {
			return false
		}
	}

	return true
}

// generateSelfSignedCert creates a fresh ECDSA P-256 key and a self-signed certificate covering hosts,
// then writes them to keyFile and certFile.
func generateSelfSignedCert(certFile, keyFile string, hosts []string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("generating certificate serial number: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: hosts[0],
		},
		NotBefore:   now.Add(-5 * time.Minute),
		NotAfter:    now.Add(certValidity),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
			continue
		}
		template.DNSNames = append(template.DNSNames, host)
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshaling private key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(keyFile), 0o755); err != nil {
		return fmt.Errorf("creating private key directory: %w", err)
	}
	if err := writePEM(keyFile, "EC PRIVATE KEY", keyDER, 0o600); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(certFile), 0o755); err != nil {
		return fmt.Errorf("creating certificate directory: %w", err)
	}
	if err := writePEM(certFile, "CERTIFICATE", der, 0o644); err != nil {
		return fmt.Errorf("writing certificate: %w", err)
	}

	return nil
}

// writePEM PEM-encodes der under blockType and writes it to the path with the given permissions.
func writePEM(path, blockType string, der []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	if err := pem.Encode(f, &pem.Block{
		Type:  blockType,
		Bytes: der,
	}); err != nil {
		_ = f.Close()
		return err
	}
	// Closing flushes the written file, so its error means the file may be incomplete
	return f.Close()
}
