package icmp

import (
	"context"
	"time"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	probing "github.com/prometheus-community/pro-bing"
)

// PingConfig holds tunable parameters for a single ping run.
type PingConfig struct {
	Count    int
	Timeout  time.Duration
	Interval time.Duration
}

// DefaultPingConfig returns sensible defaults.
func DefaultPingConfig() *PingConfig {
	return &PingConfig{
		Count:    4,
		Timeout:  3 * time.Second,
		Interval: time.Second,
	}
}

// PingIP sends ICMP echo requests to ip using the supplied config.
// If ctx carries a deadline that expires sooner than cfg.Timeout, the
// deadline is used instead so the two mechanisms do not conflict.
func PingIP(ctx context.Context, ip string, cfg *PingConfig) (*probing.Statistics, error) {

	if cfg == nil {
		cfg = DefaultPingConfig()
	}

	timeout := cfg.Timeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}

	pinger := probing.New(ip)
	pinger.SetPrivileged(true)
	pinger.Count = cfg.Count
	pinger.Timeout = timeout
	pinger.Interval = cfg.Interval

	log.GetLogger().Info("pinging ip", "ip", ip, "count", pinger.Count, "timeout", pinger.Timeout, "interval", pinger.Interval)

	if err := pinger.RunWithContext(ctx); err != nil {
		log.GetLogger().Error("ping failed", "ip", ip, "error", err)
		return nil, err
	}
	return pinger.Statistics(), nil
}
