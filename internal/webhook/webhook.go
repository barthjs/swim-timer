// Package webhook delivers signed messages to HTTP endpoints following the Standard Webhooks specification.
// It signs every request, guards against requests into private networks, and classifies the response,
// so the caller decides whether to retry, wait, or give up on a target.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// requestTimeout bounds one delivery, from dialing to reading the response.
	requestTimeout = 10 * time.Second
	// idleConnTimeout keeps a connection open between frequent deliveries to the same target.
	idleConnTimeout = 90 * time.Second
	// maxResponseBody bounds how much of a response is read and kept.
	maxResponseBody = 4 << 10
	// secretPrefix starts every secret, as the specification defines it.
	secretPrefix = "whsec_"
	// minSecretBytes and maxSecretBytes bound the decoded secret, as the specification recommends.
	minSecretBytes = 24
	maxSecretBytes = 64
	// generatedSecretBytes is the length of a secret GenerateSecret creates.
	generatedSecretBytes = 32
	// messageIDPrefix starts the id of every message.
	messageIDPrefix = "msg_"
)

var (
	// ErrInvalidSecret is returned for a secret that is not "whsec_" followed by 24 to 64 bytes in base64.
	ErrInvalidSecret = errors.New("webhook: invalid secret")
	// ErrInvalidURL is returned for a target URL that is malformed or uses a scheme that is not allowed.
	ErrInvalidURL = errors.New("webhook: invalid url")
	// ErrUntrustedCertificate is returned when the certificate of a target cannot be verified,
	// for example because it is self-signed. Target.SkipTLSVerify accepts such a certificate.
	ErrUntrustedCertificate = errors.New("webhook: untrusted certificate")
	// ErrBlockedAddress is returned when a target resolves to an address in a private or local network
	// that the allowed networks do not contain.
	ErrBlockedAddress = errors.New("webhook: address not allowed")
)

// blockedNetworks are the networks without a predicate in netip that a target must not resolve to.
var blockedNetworks = []netip.Prefix{
	// "This network", which some systems route to the local host
	netip.MustParsePrefix("0.0.0.0/8"),
	// Shared address space of carrier-grade NAT
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("255.255.255.255/32"),
	// NAT64 for a local network, which translates into addresses the guard cannot see
	netip.MustParsePrefix("64:ff9b:1::/48"),
}

var (
	// nat64Network is the well-known NAT64 prefix, whose addresses carry an IPv4 address in their last four bytes.
	nat64Network = netip.MustParsePrefix("64:ff9b::/96")
	// sixToFourNetwork is the 6to4 prefix, whose addresses carry an IPv4 address after their first two bytes.
	sixToFourNetwork = netip.MustParsePrefix("2002::/16")
)

// Options holds the settings of a Client.
type Options struct {
	// UserAgent is sent with every request.
	UserAgent string
	// AllowedNetworks are networks a target may resolve to, although they are private or local.
	AllowedNetworks []netip.Prefix
}

// Target is an endpoint and the secrets the messages to it are signed with.
// Several secrets sign every message with each of them, so a receiver can switch to a new secret without a gap.
type Target struct {
	URL     string
	Secrets []string
	// SkipTLSVerify accepts any certificate of the target, such as a self-signed one in a local network.
	// The signature still lets the target check the messages, but a man in the middle could read them.
	SkipTLSVerify bool
}

// Payload is a message as it is sent, built once so a retry sends the same id and body.
type Payload struct {
	ID   string
	Body []byte
}

// message is the envelope of every payload.
type message struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Data      any       `json:"data"`
}

// Outcome is how a delivery ended, which tells the caller what to do next.
type Outcome int

const (
	// OutcomeDelivered means the target accepted the message.
	OutcomeDelivered Outcome = iota
	// OutcomeRetry means a transient failure, so the caller tries again after a backoff.
	OutcomeRetry
	// OutcomeDeferred means the target asked to wait for Result.RetryAfter, which is no failed attempt.
	OutcomeDeferred
	// OutcomeGone means the target no longer exists, so the caller stops delivering to it.
	OutcomeGone
	// OutcomeRejected means a permanent failure, so sending the message again fails as well.
	OutcomeRejected
)

// Result describes one delivery.
type Result struct {
	Outcome Outcome
	// StatusCode is the status of the response, 0 if none was received.
	StatusCode int
	// Duration is how long the delivery took.
	Duration time.Duration
	// RetryAfter is how long the target asked to wait, set for OutcomeDeferred.
	RetryAfter time.Duration
	// ResponseBody is the start of the response body.
	ResponseBody string
	// Err is the cause of a delivery without a response, nil if the target answered.
	Err error
}

// Request is a delivery as it would be sent, so a caller can show a message to the admin of a receiver.
type Request struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
}

// Client delivers payloads to targets. It is safe for concurrent use and keeps connections open between deliveries.
type Client struct {
	options Options
	client  *http.Client
	// insecureClient skips the verification of certificates, for the targets with Target.SkipTLSVerify.
	insecureClient *http.Client
}

// New creates a Client.
func New(options Options) *Client {
	c := &Client{
		options: options,
	}
	c.client = c.newHTTPClient(false)
	c.insecureClient = c.newHTTPClient(true)
	return c
}

// GenerateSecret returns a new random secret in the format of the specification.
func GenerateSecret() string {
	key := make([]byte, generatedSecretBytes)
	// crypto/rand.Read never returns an error and crashes the program if the system has no randomness
	_, _ = rand.Read(key)
	return secretPrefix + base64.StdEncoding.EncodeToString(key)
}

// ParseSecret returns the key of a secret, or ErrInvalidSecret if the secret is not in the format of the specification.
func ParseSecret(secret string) ([]byte, error) {
	encoded, ok := strings.CutPrefix(secret, secretPrefix)
	if !ok {
		return nil, ErrInvalidSecret
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) < minSecretBytes || len(key) > maxSecretBytes {
		return nil, ErrInvalidSecret
	}
	return key, nil
}

// NewPayload encodes a message of the given type with data under a new id, stamped with the current time.
func NewPayload(typ string, data any) (Payload, error) {
	id := messageIDPrefix + rand.Text()
	body, err := json.Marshal(message{
		ID:        id,
		Type:      typ,
		CreatedAt: time.Now(),
		Data:      data,
	})
	if err != nil {
		return Payload{}, fmt.Errorf("webhook: encoding %s message: %w", typ, err)
	}
	return Payload{
		ID:   id,
		Body: body,
	}, nil
}

// CheckURL reports whether a target URL can be delivered to: ErrInvalidURL for a malformed URL or a scheme other than
// http and https, and ErrBlockedAddress for a literal address in a blocked network. A host name is resolved
// and checked on every delivery, since its address can change.
func (c *Client) CheckURL(rawURL string) error {
	target, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q is not http or https", ErrInvalidURL, target.Scheme)
	}
	if target.Hostname() == "" {
		return fmt.Errorf("%w: the host is missing", ErrInvalidURL)
	}

	if addr, err := netip.ParseAddr(target.Hostname()); err == nil && !c.addressAllowed(addr) {
		return ErrBlockedAddress
	}
	return nil
}

// Send delivers a payload to a target once and classifies the outcome. An invalid URL or secret
// and a blocked address are rejected, while a failure of the network or a timeout is retried.
func (c *Client) Send(ctx context.Context, target Target, payload Payload) Result {
	if err := c.CheckURL(target.URL); err != nil {
		return Result{
			Outcome: OutcomeRejected,
			Err:     err,
		}
	}
	keys, err := parseSecrets(target.Secrets)
	if err != nil {
		return Result{
			Outcome: OutcomeRejected,
			Err:     err,
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(payload.Body))
	if err != nil {
		return Result{
			Outcome: OutcomeRejected,
			Err:     fmt.Errorf("%w: %w", ErrInvalidURL, err),
		}
	}
	request.Header = c.header(keys, payload, time.Now())

	client := c.client
	if target.SkipTLSVerify {
		client = c.insecureClient
	}

	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		outcome := OutcomeRetry
		if errors.Is(err, ErrBlockedAddress) {
			outcome = OutcomeRejected
		}
		// A certificate can be replaced or accepted while the delivery waits, so it is retried
		if _, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
			err = fmt.Errorf("%w: %w", ErrUntrustedCertificate, err)
		}
		return Result{
			Outcome:  outcome,
			Duration: time.Since(started),
			Err:      err,
		}
	}
	defer func() { _ = response.Body.Close() }()

	// The body only serves the log of the caller, so a failure to read it does not change the outcome
	body, _ := io.ReadAll(io.LimitReader(response.Body, maxResponseBody))
	outcome, retryAfter := classify(response.StatusCode, response.Header.Get("Retry-After"), time.Now())
	return Result{
		Outcome:      outcome,
		StatusCode:   response.StatusCode,
		Duration:     time.Since(started),
		RetryAfter:   retryAfter,
		ResponseBody: string(body),
	}
}

// Preview returns the request Send would make for a payload now, signed with the secrets of the target,
// without sending it. Only a missing or invalid secret fails, with ErrInvalidSecret.
func (c *Client) Preview(target Target, payload Payload) (Request, error) {
	keys, err := parseSecrets(target.Secrets)
	if err != nil {
		return Request{}, err
	}
	return Request{
		Method: http.MethodPost,
		URL:    target.URL,
		Header: c.header(keys, payload, time.Now()),
		Body:   payload.Body,
	}, nil
}

// header returns the headers of a delivery, with the signature of the payload at the given time.
func (c *Client) header(keys [][]byte, payload Payload, now time.Time) http.Header {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("User-Agent", c.options.UserAgent)
	header.Set("webhook-id", payload.ID)
	header.Set("webhook-timestamp", timestamp)
	header.Set("webhook-signature", sign(keys, payload.ID, timestamp, payload.Body))
	return header
}

// newHTTPClient builds the HTTP client of the deliveries, one that verifies the certificates of the targets
// and one that skips the verification.
func (c *Client) newHTTPClient(skipTLSVerify bool) *http.Client {
	dialer := &net.Dialer{
		Timeout: requestTimeout,
		Control: c.checkDial,
	}
	transport := &http.Transport{
		// No proxy, since the proxy would be dialed instead of the target and bypass the address check
		Proxy:               nil,
		DialContext:         dialer.DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: requestTimeout,
		IdleConnTimeout:     idleConnTimeout,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			// The admin turns the verification off for a single target, such as one with a self-signed certificate
			InsecureSkipVerify: skipTLSVerify, //nolint:gosec // Only for targets that ask for it with SkipTLSVerify.
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		// A redirect is answered as it is, since following it could lead into a private network
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// checkDial rejects a connection to an address that is not allowed. It runs on the resolved address
// right before the connection, so a host name that resolves to another address later is checked as well.
func (c *Client) checkDial(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBlockedAddress, err)
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBlockedAddress, err)
	}
	if !c.addressAllowed(addr) {
		return fmt.Errorf("%w: %s", ErrBlockedAddress, addr)
	}
	return nil
}

// addressAllowed reports whether a target may be dialed at addr: any public address,
// and a private or local one only inside the allowed networks.
func (c *Client) addressAllowed(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, network := range c.options.AllowedNetworks {
		if network.Contains(addr) {
			return true
		}
	}

	// A gateway forwards these addresses to the IPv4 address they carry, so that address decides
	if embedded, ok := embeddedIPv4(addr); ok {
		return c.addressAllowed(embedded)
	}

	if addr.IsLoopback() || addr.IsPrivate() || addr.IsUnspecified() || addr.IsMulticast() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsInterfaceLocalMulticast() {
		return false
	}
	for _, network := range blockedNetworks {
		if network.Contains(addr) {
			return false
		}
	}
	return true
}

// embeddedIPv4 returns the IPv4 address a NAT64 or 6to4 address carries.
func embeddedIPv4(addr netip.Addr) (netip.Addr, bool) {
	raw := addr.As16()
	switch {
	case nat64Network.Contains(addr):
		return netip.AddrFrom4([4]byte{raw[12], raw[13], raw[14], raw[15]}), true
	case sixToFourNetwork.Contains(addr):
		return netip.AddrFrom4([4]byte{raw[2], raw[3], raw[4], raw[5]}), true
	default:
		return netip.Addr{}, false
	}
}

// parseSecrets returns the keys of the secrets, or ErrInvalidSecret if there is none or one is invalid.
func parseSecrets(secrets []string) ([][]byte, error) {
	if len(secrets) == 0 {
		return nil, ErrInvalidSecret
	}
	keys := make([][]byte, 0, len(secrets))
	for _, secret := range secrets {
		key, err := ParseSecret(secret)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// sign returns the signature header: one HMAC-SHA256 of id, timestamp, and body per key, separated by spaces.
func sign(keys [][]byte, id, timestamp string, body []byte) string {
	content := make([]byte, 0, len(id)+len(timestamp)+len(body)+2)
	content = append(content, id...)
	content = append(content, '.')
	content = append(content, timestamp...)
	content = append(content, '.')
	content = append(content, body...)

	signatures := make([]string, 0, len(keys))
	for _, key := range keys {
		mac := hmac.New(sha256.New, key)
		_, _ = mac.Write(content)
		signatures = append(signatures, "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	}
	return strings.Join(signatures, " ")
}

// classify maps the status of a response to an outcome. A target that is overloaded or rate limited
// and names a wait in Retry-After defers the delivery, any other transient status retries it.
func classify(status int, retryAfterHeader string, now time.Time) (Outcome, time.Duration) {
	switch {
	case status >= 200 && status < 300:
		return OutcomeDelivered, 0
	case status == http.StatusGone:
		return OutcomeGone, 0
	case status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable:
		if wait := parseRetryAfter(retryAfterHeader, now); wait > 0 {
			return OutcomeDeferred, wait
		}
		return OutcomeRetry, 0
	case status == http.StatusRequestTimeout || status >= 500:
		return OutcomeRetry, 0
	default:
		return OutcomeRejected, 0
	}
}

// parseRetryAfter returns the wait a Retry-After header names in seconds or as a date, 0 if it names none.
func parseRetryAfter(header string, now time.Time) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(max(seconds, 0)) * time.Second
	}
	if date, err := http.ParseTime(header); err == nil {
		return max(date.Sub(now), 0)
	}
	return 0
}
