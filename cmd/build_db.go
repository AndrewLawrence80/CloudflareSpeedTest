package cmd

import (
	"context"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/cloudflare"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/dns"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/domain"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/model"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/store"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/concurrency"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	progressbar "github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"gorm.io/gorm/clause"
)

var buildDBCmd = &cobra.Command{
	Use:   "build-db",
	Short: "Build the SQLite database with DNS record",
	Long:  "Build the SQLite database with DNS record. This command will load all possible domains from github.com/v2fly/domain-list-community and perform DNS lookups to populate the database with IP addresses and their associated metadata.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doBuildDB(ctx)
	},
}

func init() {
	rootCmd.AddCommand(buildDBCmd)
}

func doBuildDB(ctx context.Context) {
	log.GetLogger().InfoContext(ctx, "loading domains")
	domains, err := domain.LoadAllDomains()
	if err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load domains", "error", err)
		return
	}
	log.GetLogger().InfoContext(ctx, "loaded domains", "count", len(domains))
	if err := resolveDNSRecords(ctx, domains); err != nil {
		log.GetLogger().ErrorContext(ctx, "error occurred during dns resolution", "error", err)
	}
}

func resolveDNSRecords(ctx context.Context, domains []string) error {
	pbar := progressbar.Default(int64(len(domains)), "resolving domains")
	nRoutines := common.EnvInt("NUM_DNS_WORKERS", 16)
	qpm := common.EnvUint("QPM_DNS", 0)

	var existingRecords []model.DNSRecord
	if err := store.GetDB().WithContext(ctx).Find(&existingRecords).Error; err != nil {
		log.GetLogger().ErrorContext(ctx, "failed to load existing DNS records", "error", err)
		return err
	}
	existingRecordsMap := make(map[string]model.DNSRecord, len(existingRecords))
	for _, r := range existingRecords {
		existingRecordsMap[r.Domain] = r
	}
	executor := concurrency.NewSimpleExecutor(uint(nRoutines), qpm)

	for _, d := range domains {
		if existing, exists := existingRecordsMap[d]; exists && existing.Success {
			log.GetLogger().InfoContext(ctx, "skipping already resolved domain", "domain", d)
			pbar.Add(1)
			continue
		}
		if err := executor.Submit(ctx, func() {
			d := d // Capture local variable
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.GetLogger().InfoContext(ctx, "resolving domain", "domain", d)
			v4ips, v6ips, err := dns.Resolve(ctx, d)
			if err != nil {
				log.GetLogger().ErrorContext(ctx, "failed to resolve domain", "domain", d, "error", err)
				if storeErr := store.GetDB().WithContext(ctx).Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "domain"}},
					DoUpdates: clause.AssignmentColumns([]string{"success", "updated_at"}),
				}).Create(&model.DNSRecord{
					Domain:  d,
					Success: false,
				}).Error; storeErr != nil {
					log.GetLogger().ErrorContext(ctx, "failed to store failed DNS record", "domain", d, "error", storeErr)
				}
				pbar.Add(1)
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

			isCloudflare := isCloudflareIP(append(v4Strs, v6Strs...))

			if storeErr := store.GetDB().WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "domain"}},
				DoUpdates: clause.AssignmentColumns([]string{"ipv4", "ipv6", "success", "is_cloudflare", "updated_at"}),
			}).Create(&model.DNSRecord{
				Domain:       d,
				IPv4:         v4Strs,
				IPv6:         v6Strs,
				Success:      true,
				IsCloudflare: isCloudflare,
			}).Error; storeErr != nil {
				log.GetLogger().ErrorContext(ctx, "failed to store DNS record", "domain", d, "error", storeErr)
			}
			log.GetLogger().InfoContext(ctx, "resolved domain", "domain", d, "v4", v4Strs, "v6", v6Strs)
			pbar.Add(1)
		}); err != nil {
			log.GetLogger().ErrorContext(ctx, "failed to submit DNS task", "domain", d, "error", err)
		}
	}
	executor.Close()
	executor.Wait()
	return nil
}

func isCloudflareIP(ips []string) bool {
	for _, ip := range ips {
		if cloudflare.IsCloudflareIP(ip) {
			return true
		}
	}
	return false
}
