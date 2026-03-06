package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/cmd"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	godotenv.Load(".env")
	cmd.ExecuteContext(ctx)
}
