package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/cloudflare"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/icmp"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/model"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/store"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/concurrency"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	progressbar "github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"gorm.io/gorm/clause"
)

var icmpv4PingCmd = &cobra.Command{
	Use:   "icmpv4-ping",
	Short: "Perform ICMPv4 ping tests to Cloudflare's network",
	Long:  "Perform ICMPv4 ping tests to Cloudflare's network. This command will read the list of IPv4 addresses from the database and perform ICMPv4 ping tests to measure latency. The results will be stored in the database for analysis.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doICMPv4Ping(ctx)
	},
}

var icmpv6PingCmd = &cobra.Command{
	Use:   "icmpv6-ping",
	Short: "Perform ICMPv6 ping tests to Cloudflare's network",
	Long:  "Perform ICMPv6 ping tests to Cloudflare's network. This command will read the list of IPv6 addresses from the database and perform ICMPv6 ping tests to measure latency. The results will be stored in the database for analysis.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doICMPv6Ping(ctx)
	},
}

func init() {
	rootCmd.AddCommand(icmpv4PingCmd)
	rootCmd.AddCommand(icmpv6PingCmd)
}

func doICMPv4Ping(ctx context.Context) {
	log.GetLogger().InfoContext(ctx, "loading IPv4 addresses for ICMPv4 ping tests")
	var records []model.DNSRecord
	if err := store.GetDB().WithContext(ctx).
		Where("success = ?", true).
		Where("is_cloudflare = ?", true).
		Find(&records).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load DNS records", "error", err)
		return
	}
	ipCollection := make(map[string]struct{})
	for _, r := range records {
		for _, ip := range r.IPv4 {
			if cloudflare.IsCloudflareIP(ip) {
				ipCollection[ip] = struct{}{}
			}
		}
	}
	ips := make([]string, 0, len(ipCollection))
	for ip := range ipCollection {
		ips = append(ips, ip)
	}
	log.GetLogger().InfoContext(ctx, "loaded IPv4 addresses for ICMPv4 ping tests", "count", len(ips))
	if err := pingIP(ctx, ips); err != nil {
		log.GetLogger().ErrorContext(ctx, "error occurred during ICMPv4 ping tests", "error", err)
	}
}

func doICMPv6Ping(ctx context.Context) {
	log.GetLogger().InfoContext(ctx, "loading IPv6 addresses for ICMPv6 ping tests")
	var records []model.DNSRecord
	if err := store.GetDB().WithContext(ctx).Where("success = ?", true).Find(&records).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load DNS records", "error", err)
		return
	}
	ipCollection := make(map[string]struct{})
	for _, r := range records {
		for _, ip := range r.IPv6 {
			if cloudflare.IsCloudflareIP(ip) {
				ipCollection[ip] = struct{}{}
			}
		}
	}
	ips := make([]string, 0, len(ipCollection))
	for ip := range ipCollection {
		ips = append(ips, ip)
	}
	log.GetLogger().InfoContext(ctx, "loaded IPv6 addresses for ICMPv6 ping tests", "count", len(ips))
	if err := pingIP(ctx, ips); err != nil {
		log.GetLogger().ErrorContext(ctx, "error occurred during ICMPv6 ping tests", "error", err)
	}
}

func pingIP(ctx context.Context, ips []string) error {
	if len(ips) == 0 {
		msg := "no IPs to ping, may be missing DNS records or no Cloudflare IPs found, you can run the 'build-db' command to populate the database with DNS records"
		fmt.Println(msg)
		log.GetLogger().InfoContext(ctx, msg)
		return nil
	}
	pbar := progressbar.Default(int64(len(ips)), "pinging IPs")
	nRoutines := common.EnvInt("NUM_ICMP_WORKERS", 16)
	cfg := &icmp.PingConfig{
		Count:    common.EnvInt("ICMP_COUNT", 4),
		Timeout:  time.Duration(common.EnvInt("ICMP_TIMEOUT", 3)) * time.Second,
		Interval: time.Duration(common.EnvInt("ICMP_INTERVAL", 1)) * time.Second,
	}

	executor := concurrency.NewSimpleExecutor(uint(nRoutines), 0)
	for _, ip := range ips {
		if err := executor.Submit(ctx, func() {
			ip := ip // Capture local variable
			select {
			case <-ctx.Done():
				return
			default:
			}

			stats, err := icmp.PingIP(ctx, ip, cfg)
			if err != nil {
				log.GetLogger().ErrorContext(ctx, "ping failed", "ip", ip, "error", err)
				return
			}
			log.GetLogger().InfoContext(ctx, "ping result", "ip", ip,
				"min_rtt", stats.MinRtt, "avg_rtt", stats.AvgRtt,
				"max_rtt", stats.MaxRtt, "packet_loss", stats.PacketLoss)
			store.GetDB().WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "ip"}},
				DoUpdates: clause.AssignmentColumns([]string{"min_rtt", "avg_rtt", "max_rtt", "packet_loss"}),
			}).Create(&model.ICMPingSummary{
				IP:         ip,
				MinRTT:     stats.MinRtt.Seconds(),
				AvgRTT:     stats.AvgRtt.Seconds(),
				MaxRTT:     stats.MaxRtt.Seconds(),
				PacketLoss: stats.PacketLoss,
			})
			pbar.Add(1)
		}); err != nil {
			log.GetLogger().ErrorContext(ctx, "failed to submit ping task", "ip", ip, "error", err)
		}
	}
	executor.Close()
	executor.Wait()
	return nil
}
