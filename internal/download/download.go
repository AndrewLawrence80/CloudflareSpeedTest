package download

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
)

//go:generate gotests -w -exported download.go
//go:generate mockgen -source download.go -destination mocks/download_mock.go -package mocks

// BufferSize is the per-read buffer size used while draining the response body (2 MiB).
const BufferSize = 1 << 21

type DownloadConfig struct {
	TimeOut time.Duration
}

func DefaultDownloadConfig() *DownloadConfig {
	return &DownloadConfig{
		TimeOut: 30 * time.Second,
	}
}

// DownloadSummary holds the result of a successful download measurement.
type DownloadSummary struct {
	IP        string
	Duration  time.Duration
	Size      float64 // MiB downloaded
	Bandwidth float64 // MiB/s
}

// Download measures the download throughput of rawURL via the CDN node at ip.
// The dialer is overridden to connect directly to ip while preserving the
// original Host header and TLS SNI, so CDN routing and certificate validation
// work correctly.
func Download(ctx context.Context, rawURL string, ip string, config *DownloadConfig) (*DownloadSummary, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to parse url", "url", rawURL, "error", err)
		return nil, err
	}

	targetPort, err := resolvePort(parsedURL)
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to resolve target port", "url", rawURL, "error", err)
		return nil, err
	}

	transport := &http.Transport{
		DialContext: fixedHostDialer(ip, targetPort),
	}
	if parsedURL.Scheme == "https" {
		transport.TLSClientConfig = &tls.Config{
			ServerName: parsedURL.Hostname(),
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.TimeOut,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to create http request", "url", rawURL, "error", err)
		return nil, err
	}
	// Always set Host so the CDN node receives the correct virtual-host name
	// regardless of the IP we dialled directly.
	req.Host = parsedURL.Host

	rsp, err := client.Do(req)
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to send http request", "url", rawURL, "ip", ip, "error", err)
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-OK HTTP status: %s", rsp.Status)
	}

	return measureDownload(ctx, rsp.Body, ip, rawURL, config.TimeOut)
}

// measureDownload drains body for up to timeout and returns throughput statistics.
// It always returns a non-nil *DownloadSummary with however many bytes were read.
// The returned error is non-nil only for unexpected mid-transfer failures;
// normal terminations (EOF, context cancellation, deadline) return err == nil.
func measureDownload(ctx context.Context, body io.Reader, ip, rawURL string, timeout time.Duration) (*DownloadSummary, error) {
	buffer := make([]byte, BufferSize)
	var bytesRead int64
	var readErr error
	start := time.Now()
	deadline := start.Add(timeout)

	for time.Now().Before(deadline) {
		n, err := body.Read(buffer)
		if n > 0 {
			bytesRead += int64(n)
		}
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				// Normal end of body.
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				// External cancellation or client/read timeout — expected during speed testing.
				log.GetLogger().DebugContext(ctx, "download stopped early", "ip", ip, "reason", err, "bytes_read", bytesRead)
			default:
				// Unexpected transfer error; report it but still surface the partial measurement.
				log.GetLogger().ErrorContext(ctx, "download read error", "url", rawURL, "ip", ip, "error", err)
				readErr = err
			}
			break
		}
	}

	duration := time.Since(start)
	sizeMiB := float64(bytesRead) / (1 << 20)
	bw := safeBandwidth(sizeMiB, duration)
	log.GetLogger().InfoContext(ctx, "download summary", "ip", ip, "size_mib", sizeMiB, "duration", duration, "bandwidth_mib_s", bw, "error", readErr)
	return &DownloadSummary{
		IP:        ip,
		Duration:  duration,
		Size:      sizeMiB,
		Bandwidth: bw,
	}, readErr
}

// safeBandwidth returns MiB/s, returning 0 for zero or negative durations.
func safeBandwidth(sizeMiB float64, duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	return sizeMiB / duration.Seconds()
}

// fixedHostDialer returns a DialContext that always connects to targetIP:targetPort,
// bypassing DNS so a specific CDN node is tested directly.
func fixedHostDialer(targetIP, targetPort string) func(ctx context.Context, network, address string) (net.Conn, error) {
	addr := net.JoinHostPort(targetIP, targetPort)
	return func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
}

// resolvePort returns the port to connect to for u.
// An explicit port in the URL takes precedence over the scheme default.
func resolvePort(u *url.URL) (string, error) {
	if port := u.Port(); port != "" {
		return port, nil
	}
	switch u.Scheme {
	case "https":
		return "443", nil
	case "http":
		return "80", nil
	default:
		return "", fmt.Errorf("unsupported URL scheme: %q", u.Scheme)
	}
}
