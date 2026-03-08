package cmd

import (
	"context"
	"os"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/model"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/store"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
)

var exportDNSCmd = &cobra.Command{
	Use:   "export-dns",
	Short: "Export DNS records from the database to a CSV file",
	Long:  "Export DNS records from the database to a CSV file. This command will read all DNS records from the database and export them to a CSV file named 'dns_records.csv' in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doExportDNS(ctx)
	},
}

func doExportDNS(ctx context.Context) {
	var records []model.DNSRecord
	if err := store.GetDB().WithContext(ctx).Where("success = ?", true).Find(&records).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load DNS records for export", "error", err)
		return
	}
	if err := exportToCSV(ctx, &records, "dns_records.csv"); err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to export DNS records to CSV", "error", err)
	}
}

var exportICMPv4Cmd = &cobra.Command{
	Use:   "export-icmp",
	Short: "Export ICMP ping summaries from the database to a CSV file",
	Long:  "Export ICMP ping summaries from the database to a CSV file. This command will read all ICMP ping summaries from the database and export them to a CSV file named 'icmp_summaries.csv' in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doExportICMPv4(ctx)
	},
}

func doExportICMPv4(ctx context.Context) {
	var records []model.ICMPingSummary
	if err := store.GetDB().WithContext(ctx).Where("ip like '%.%.%.%'").
		Order("packet_loss asc").Order("avg_rtt asc").Find(&records).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load ICMP ping summaries for export", "error", err)
		return
	}
	if err := exportToCSV(ctx, &records, "icmp_summaries.csv"); err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to export ICMP ping summaries to CSV", "error", err)
	}
}

var exportICMPv6Cmd = &cobra.Command{
	Use:   "export-icmpv6",
	Short: "Export ICMPv6 ping summaries from the database to a CSV file",
	Long:  "Export ICMPv6 ping summaries from the database to a CSV file. This command will read all ICMPv6 ping summaries from the database and export them to a CSV file named 'icmpv6_summaries.csv' in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doExportICMPv6(ctx)
	},
}

func doExportICMPv6(ctx context.Context) {
	var records []model.ICMPingSummary
	if err := store.GetDB().WithContext(ctx).Where("ip like '%:%'").
		Order("packet_loss asc").Order("avg_rtt asc").Find(&records).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load ICMPv6 ping summaries for export", "error", err)
		return
	}
	if err := exportToCSV(ctx, &records, "icmpv6_summaries.csv"); err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to export ICMPv6 ping summaries to CSV", "error", err)
	}
}

var exportBandwidthV4Cmd = &cobra.Command{
	Use:   "export-bandwidthv4",
	Short: "Export bandwidth test summaries from the database to a CSV file",
	Long:  "Export bandwidth test summaries from the database to a CSV file. This command will read all bandwidth test summaries from the database and export them to a CSV file named 'bandwidth_summaries.csv' in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doExportBandwidthV4(ctx)
	},
}

func doExportBandwidthV4(ctx context.Context) {
	var records []model.DownloadSummary
	if err := store.GetDB().WithContext(ctx).
		Where("ip like '%.%.%.%'").Order("bandwidth desc").Find(&records).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load bandwidth test summaries for export", "error", err)
		return
	}
	if err := exportToCSV(ctx, &records, "bandwidth_summaries.csv"); err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to export bandwidth test summaries to CSV", "error", err)
	}
}

var exportBandwidthV6Cmd = &cobra.Command{
	Use:   "export-bandwidthv6",
	Short: "Export IPv6 bandwidth test summaries from the database to a CSV file",
	Long:  "Export IPv6 bandwidth test summaries from the database to a CSV file. This command will read all IPv6 bandwidth test summaries from the database and export them to a CSV file named 'bandwidthv6_summaries.csv' in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doExportBandwidthV6(ctx)
	},
}

func doExportBandwidthV6(ctx context.Context) {
	var records []model.DownloadSummary
	if err := store.GetDB().WithContext(ctx).
		Where("ip like '%:%'").Order("bandwidth desc").Find(&records).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load IPv6 bandwidth test summaries for export", "error", err)
		return
	}
	if err := exportToCSV(ctx, &records, "bandwidthv6_summaries.csv"); err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to export IPv6 bandwidth test summaries to CSV", "error", err)
	}
}

func init() {
	rootCmd.AddCommand(exportDNSCmd)
	rootCmd.AddCommand(exportICMPv4Cmd)
	rootCmd.AddCommand(exportICMPv6Cmd)
	rootCmd.AddCommand(exportBandwidthV4Cmd)
	rootCmd.AddCommand(exportBandwidthV6Cmd)
}

func exportToCSV(ctx context.Context, data interface{}, filename string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to open CSV file for export", "error", err)
		return err
	}
	defer file.Close()

	if err := gocsv.MarshalFile(data, file); err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to export records to CSV", "error", err)
		return err
	}
	return nil
}
