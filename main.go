package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/cloudflare"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/dns"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/domain"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/download"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/icmp"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/model"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/store"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/concurrency"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	godotenv.Load(".env")

	domains, err := domain.LoadAllDomains()
	if err != nil {
		log.GetLogger().Error("failed to load domains", "error", err)
		return
	}
	log.GetLogger().Info("loaded domains", "count", len(domains))

	v4ips, v6ips, err := ResolveDNSRecords(ctx, domains)
	if err != nil {
		log.GetLogger().Error("failed to resolve DNS records", "error", err)
		return
	}
	log.GetLogger().Info("resolved DNS records", "v4_count", len(v4ips), "v6_count", len(v6ips))

	cloudflarev4ips := getCloudflareIPs(v4ips)
	cloudflarev6ips := getCloudflareIPs(v6ips)
	log.GetLogger().Info("get Cloudflare IPs", "v4_count", len(cloudflarev4ips), "v6_count", len(cloudflarev6ips))

	// Always test IPv4.
	if err := ExecICMPPing(ctx, cloudflarev4ips); err != nil {
		log.GetLogger().Error("ICMPv4 ping phase failed", "error", err)
		return
	}

	topN := envInt("TOP_N_IPS", 10)

	topV4IPs, err := SelectTopIPsByRTT(ctx, cloudflarev4ips, topN)
	if err != nil {
		log.GetLogger().Error("failed to select top IPv4 IPs by RTT", "error", err)
		return
	}
	log.GetLogger().Info("selected top IPv4 IPs for download test", "count", len(topV4IPs))
	ExecDownload(ctx, topV4IPs)

	// Test IPv6 only when the system has a routable IPv6 address.
	systemIPv6Addrs, err := DetectSystemIPv6()
	if err != nil {
		log.GetLogger().Warn("could not detect system IPv6 addresses", "error", err)
	}
	if len(systemIPv6Addrs) > 0 && len(cloudflarev6ips) > 0 {
		log.GetLogger().Info("system has IPv6, running IPv6 speed test", "system_addrs", systemIPv6Addrs)
		if err := ExecICMPPing(ctx, cloudflarev6ips); err != nil {
			log.GetLogger().Error("ICMPv6 ping phase failed", "error", err)
			return
		}
		topV6IPs, err := SelectTopIPsByRTT(ctx, cloudflarev6ips, topN)
		if err != nil {
			log.GetLogger().Error("failed to select top IPv6 IPs by RTT", "error", err)
			return
		}
		log.GetLogger().Info("selected top IPv6 IPs for download test", "count", len(topV6IPs))
		ExecDownload(ctx, topV6IPs)
	} else {
		log.GetLogger().Info("skipping IPv6 speed test: no routable system IPv6 address found")
	}
}

// ResolveDNSRecords concurrently resolves all domains and returns the unique
// IPv4 and IPv6 addresses. Results are also persisted to the database.
func ResolveDNSRecords(ctx context.Context, domains []string) ([]string, []string, error) {
	nRoutines := envInt("NUM_DNS_WORKERS", 16)

	var mu sync.Mutex
	v4Set := make(map[string]struct{})
	v6Set := make(map[string]struct{})

	executor := concurrency.NewSimpleExecutor(uint(nRoutines), 0)
	var existingRecords []model.DNSRecord
	if err := store.GetDB().WithContext(ctx).Find(&existingRecords).Error; err != nil {
		return nil, nil, err
	}
	for _, r := range existingRecords {
		for _, ip := range r.IPv4 {
			v4Set[ip] = struct{}{}
		}
		for _, ip := range r.IPv6 {
			v6Set[ip] = struct{}{}
		}
	}
	existingRecordMap := make(map[string]model.DNSRecord)
	for _, r := range existingRecords {
		existingRecordMap[r.Domain] = r
	}
	for _, d := range domains {
		_, ok := existingRecordMap[d]
		if ok {
			log.GetLogger().InfoContext(ctx, "skipping DNS resolution for domain with existing record", "domain", d)
			continue
		}
		if err := executor.Submit(func() {
			select {
			case <-ctx.Done():
				log.GetLogger().InfoContext(ctx, "DNS resolution cancelled", "domain", d)
				return
			default:
			}
			log.GetLogger().InfoContext(ctx, "resolving domain", "domain", d)
			v4ips, v6ips, err := dns.Resolve(ctx, d)
			if err != nil {
				log.GetLogger().ErrorContext(ctx, "failed to resolve domain", "domain", d, "error", err)
				return
			}

			v4Strs := make([]string, len(v4ips))
			for i, ip := range v4ips {
				v4Strs[i] = ip.String()
			}
			v6Strs := make([]string, len(v6ips))
			for i, ip := range v6ips {
				v6Strs[i] = ip.String()
			}

			mu.Lock()
			for _, s := range v4Strs {
				v4Set[s] = struct{}{}
			}
			for _, s := range v6Strs {
				v6Set[s] = struct{}{}
			}
			mu.Unlock()

			store.GetDB().WithContext(ctx).Create(&model.DNSRecord{
				Domain: d,
				IPv4:   v4Strs,
				IPv6:   v6Strs,
			})
			log.GetLogger().InfoContext(ctx, "resolved domain", "domain", d, "v4", v4Strs, "v6", v6Strs)
		}); err != nil {
			log.GetLogger().ErrorContext(ctx, "failed to submit DNS task", "domain", d, "error", err)
		}

	}
	executor.Close()
	executor.Wait()

	return setToSlice(v4Set), setToSlice(v6Set), nil
}

// ExecICMPPing concurrently pings all IPs and persists the results to the database.
func ExecICMPPing(ctx context.Context, ips []string) error {
	nRoutines := envInt("NUM_ICMP_WORKERS", 16)
	cfg := &icmp.PingConfig{
		Count:    envInt("ICMP_COUNT", 4),
		Timeout:  time.Duration(envInt("ICMP_TIMEOUT", 3)) * time.Second,
		Interval: time.Duration(envInt("ICMP_INTERVAL", 1)) * time.Second,
	}

	executor := concurrency.NewSimpleExecutor(uint(nRoutines), 0)
	for _, ip := range ips {
		if err := executor.Submit(func() {
			select {
			case <-ctx.Done():
				log.GetLogger().InfoContext(ctx, "ping task cancelled", "ip", ip)
				return
			default:
			}

			log.GetLogger().InfoContext(ctx, "pinging IP", "ip", ip)
			stats, err := icmp.PingIP(ctx, ip, cfg)
			if err != nil {
				log.GetLogger().ErrorContext(ctx, "ping failed", "ip", ip, "error", err)
				return
			}
			log.GetLogger().InfoContext(ctx, "ping result", "ip", ip,
				"min_rtt", stats.MinRtt, "avg_rtt", stats.AvgRtt,
				"max_rtt", stats.MaxRtt, "packet_loss", stats.PacketLoss)
			store.GetDB().WithContext(ctx).Create(&model.ICMPingSummary{
				IP:         ip,
				MinRTT:     stats.MinRtt.Seconds(),
				AvgRTT:     stats.AvgRtt.Seconds(),
				MaxRTT:     stats.MaxRtt.Seconds(),
				PacketLoss: stats.PacketLoss,
			})
		}); err != nil {
			log.GetLogger().ErrorContext(ctx, "failed to submit ping task", "ip", ip, "error", err)
		}
	}
	executor.Close()
	executor.Wait()
	return nil
}

// SelectTopIPsByRTT queries the database for the n IPs from the provided
// candidate set with the lowest average RTT, excluding those with 100% packet
// loss. Scoping by candidates keeps IPv4 and IPv6 selections independent.
func SelectTopIPsByRTT(ctx context.Context, candidates []string, n int) ([]string, error) {
	var records []model.ICMPingSummary
	if err := store.GetDB().WithContext(ctx).
		Where("ip IN ? AND packet_loss < ?", candidates, 0.25).
		Order("avg_rtt asc").
		Limit(n).
		Find(&records).Error; err != nil {
		return nil, err
	}
	ips := make([]string, len(records))
	for i, r := range records {
		ips[i] = r.IP
	}
	return ips, nil
}

// DetectSystemIPv6 returns all routable (global unicast) IPv6 addresses found
// on local network interfaces, excluding loopback (::1) and link-local
// (fe80::/10) addresses. A non-empty result means the system can use IPv6.
func DetectSystemIPv6() ([]net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var routable []net.IP
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.To4() != nil {
			continue // skip unresolved or IPv4 addresses
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue // skip ::1 and fe80::/10
		}
		if ip.IsGlobalUnicast() {
			routable = append(routable, ip)
		}
	}
	return routable, nil
}

// ExecDownload concurrently measures download throughput for each IP.
func ExecDownload(ctx context.Context, ips []string) {
	testURL := common.EnvOr("TEST_URL", "https://speed.cloudflare.com/__down?bytes=100000000")
	nRoutines := envInt("NUM_HTTP_WORKERS", 4)
	cfg := &download.DownloadConfig{
		TimeOut: time.Duration(envInt("HTTP_TIMEOUT", 30)) * time.Second,
	}

	executor := concurrency.NewSimpleExecutor(uint(nRoutines), 0)
	for _, ip := range ips {
		if err := executor.Submit(func() {
			select {
			case <-ctx.Done():
				log.GetLogger().InfoContext(ctx, "download task cancelled", "ip", ip)
				return
			default:
			}

			log.GetLogger().InfoContext(ctx, " starting download test", "ip", ip, "url", testURL)
			summary, err := download.Download(ctx, testURL, ip, cfg)
			if err != nil {
				log.GetLogger().ErrorContext(ctx, "download failed", "ip", ip, "error", err)
				return
			}
			log.GetLogger().InfoContext(ctx, "download result", "ip", ip,
				"size_mib", summary.Size, "bandwidth_mib_s", summary.Bandwidth, "duration", summary.Duration)
			store.GetDB().WithContext(ctx).Create(&model.DownloadSummary{
				IP:        ip,
				Bandwidth: summary.Bandwidth,
			})
		}); err != nil {
			log.GetLogger().ErrorContext(ctx, "failed to submit download task", "ip", ip, "error", err)
		}
	}
	executor.Close()
	executor.Wait()
}

func getCloudflareIPs(ips []string) []string {
	var filtered []string
	for _, ip := range ips {
		if cloudflare.IsCloudflareIP(ip) {
			filtered = append(filtered, ip)
		}
	}
	return filtered
}

// envInt reads an integer from an environment variable, returning fallback when unset.
// Panics with a clear message if the value is set but cannot be parsed.
func envInt(key string, fallback int) int {
	s := common.EnvOr(key, "")
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		panic("invalid value for " + key + ": " + err.Error())
	}
	return v
}

// setToSlice converts a string set to a slice.
func setToSlice(s map[string]struct{}) []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	return out
}
