package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/download"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/model"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/store"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/concurrency"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"gorm.io/gorm/clause"
)

var bandWidthV4Cmd = &cobra.Command{
	Use:   "bandwidthv4",
	Short: "Perform bandwidth tests on IPv4 to Cloudflare's network",
	Long:  "Perform bandwidth tests on IPv4 to Cloudflare's network. This command will read the list of IPv4 addresses from the database and perform HTTP download tests to measure bandwidth. The results will be stored in the database for analysis.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doBandwidthV4Test(ctx)
	},
}

var bandWidthV6Cmd = &cobra.Command{
	Use:   "bandwidthv6",
	Short: "Perform bandwidth tests on IPv6 to Cloudflare's network",
	Long:  "Perform bandwidth tests on IPv6 to Cloudflare's network. This command will read the list of IPv6 addresses from the database and perform HTTP download tests to measure bandwidth. The results will be stored in the database for analysis.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doBandwidthV6Test(ctx)
	},
}

func init() {
	rootCmd.AddCommand(bandWidthV4Cmd)
	rootCmd.AddCommand(bandWidthV6Cmd)
}

func doBandwidthV4Test(ctx context.Context) {
	ips, err := selectTopV4IPsByRTT(ctx)
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to select top IPv4 IPs for bandwidth test", "error", err)
		return
	}
	downloadTest(ctx, ips)
}

func doBandwidthV6Test(ctx context.Context) {
	ips, err := selectTopV6IPsByRTT(ctx)
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to select top IPv6 IPs for bandwidth test", "error", err)
		return
	}
	downloadTest(ctx, ips)
}

func selectTopV4IPsByRTT(ctx context.Context) ([]string, error) {
	var records []model.ICMPingSummary
	if err := store.GetDB().WithContext(ctx).
		Where("ip like '%.%.%.%'").
		Where("packet_loss <= ?", common.EnvFloat("ICMP_PACKETLOSS_THRESHOLD", 0.25)).
		Order("avg_rtt asc").
		Limit(common.EnvInt("TOP_N_IPS", 10)).
		Find(&records).Error; err != nil {
		return nil, err
	}
	ips := make([]string, len(records))
	for i, r := range records {
		ips[i] = r.IP
	}
	return ips, nil
}
func selectTopV6IPsByRTT(ctx context.Context) ([]string, error) {
	var records []model.ICMPingSummary
	if err := store.GetDB().WithContext(ctx).
		Where("ip like '%:%'").
		Where("packet_loss <= ?", common.EnvFloat("ICMP_PACKETLOSS_THRESHOLD", 0.25)).
		Order("avg_rtt asc").
		Limit(common.EnvInt("TOP_N_IPS", 10)).
		Find(&records).Error; err != nil {
		return nil, err
	}
	ips := make([]string, len(records))
	for i, r := range records {
		ips[i] = r.IP
	}
	return ips, nil
}

func downloadTest(ctx context.Context, ips []string) {
	if len(ips) == 0 {
		msg := "no IPs to test bandwidth, may be missing ICMP ping summaries or no Cloudflare IPs found, you can run the 'icmp-ping' command to perform ICMP ping tests and populate the database with summaries"
		fmt.Println(msg)
		log.GetLogger().InfoContext(ctx, msg)
		return
	}
	pbar := progressbar.Default(int64(len(ips)), "performing bandwidth tests")
	testURL := common.EnvOr("TEST_URL", "https://speed.cloudflare.com/__down?bytes=100000000")
	nRoutines := common.EnvInt("NUM_HTTP_WORKERS", 1)
	cfg := &download.DownloadConfig{
		TimeOut: time.Duration(common.EnvInt("HTTP_TIMEOUT", 30)) * time.Second,
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

			log.GetLogger().InfoContext(ctx, " starting download test", "ip", ip, "url", testURL)
			summary, err := download.Download(ctx, testURL, ip, cfg)
			if err != nil {
				log.GetLogger().ErrorContext(ctx, "download failed", "ip", ip, "error", err)
				return
			}
			log.GetLogger().InfoContext(ctx, "download result", "ip", ip,
				"size_mib", summary.Size, "bandwidth_mib_s", summary.Bandwidth, "duration", summary.Duration)
			store.GetDB().WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "ip"}},
				DoUpdates: clause.AssignmentColumns([]string{"bandwidth", "updated_at"}),
			}).Create(&model.DownloadSummary{
				IP:        ip,
				Bandwidth: summary.Bandwidth,
			})
		}); err != nil {
			log.GetLogger().ErrorContext(ctx, "failed to submit download task", "ip", ip, "error", err)
		}
		pbar.Add(1)
	}
	executor.Close()
	executor.Wait()
}
