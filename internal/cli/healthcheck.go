package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/config"
)

// healthcheckTimeout bounds the healthcheck request, below the usual timeout of a container health check.
const healthcheckTimeout = 2 * time.Second

// newHealthcheckCommand checks that the application on this machine answers, for container health checks.
// It runs without the setup, so a check neither builds a logger nor touches the log file.
func newHealthcheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Check that the application is running",
		Long: "Requests the liveness endpoint of the application configured on this machine.\n" +
			"Exits with 0 when it answers, 1 otherwise.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cmd.Flags())
			if err != nil {
				return cmdline.Exit(1, err)
			}
			if err := checkHealth(cmd.Context(), cfg.Server); err != nil {
				return cmdline.Exit(1, err)
			}
			return nil
		},
	}
	return cmdline.SkipSetup(cmd)
}

// checkHealth requests the liveness endpoint of the application. An application listening on all interfaces
// is reached over the loopback address. The certificate is not verified, since the check targets the own process,
// which may use a self-signed certificate.
func checkHealth(ctx context.Context, cfg config.ServerConfig) error {
	scheme := "http"
	if cfg.SSLEnabled {
		scheme = "https"
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	url := scheme + "://" + net.JoinHostPort(host, strconv.Itoa(cfg.Port)) + "/healthz"

	client := &http.Client{
		Timeout: healthcheckTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			}, //nolint:gosec // the own process is checked
		},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	_ = response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %d", url, response.StatusCode)
	}
	return nil
}
