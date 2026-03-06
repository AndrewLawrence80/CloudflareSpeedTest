package cmd

import (
	"context"
	"fmt"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "CloudflareSpeedTest",
	Short: "A tool to test the speed of Cloudflare's network from various locations around the world",
	Long:  "A tool to test the speed of Cloudflare's network from various locations around the world. It performs ICMP ping and HTTP download tests to measure latency and bandwidth, respectively. The results are stored in a SQLite database for analysis.",
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of CloudflareSpeedTest",
	Long:  "Print the version number of CloudflareSpeedTest",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CloudflareSpeedTest v0.0.1")
	},
}

func ExecuteContext(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.GetLogger().Error("command execution failed", "error", err)
	}
}
