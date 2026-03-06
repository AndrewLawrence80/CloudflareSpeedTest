package cmd

import (
	"net"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	"github.com/spf13/cobra"
)

var pipelineCmd = &cobra.Command{
	Use:   "test-pipeline",
	Short: "Run the full test pipeline",
	Long:  "Run the full test pipeline. This command will execute the entire testing process, including building the database, performing ICMP ping tests, and conducting bandwidth tests. It is a convenient way to run all tests in sequence with a single command.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		doBuildDB(ctx)
		doICMPv4Ping(ctx)
		doBandwidthV4Test(ctx)

		systemIPv6Addrs, err := detectSystemIPv6()
		if err != nil {
			log.GetLogger().Warn("could not detect system IPv6 addresses", "error", err)
		}
		if len(systemIPv6Addrs) == 0 {
			log.GetLogger().Warn("no system IPv6 addresses detected, skipping IPv6 tests")
		}

		doICMPv6Ping(ctx)
		doBandwidthV6Test(ctx)
	},
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
}

func detectSystemIPv6() ([]net.IP, error) {
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
