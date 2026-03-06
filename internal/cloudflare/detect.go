package cloudflare

import (
	"net/netip"
	"os"
	"strings"
	"sync"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
)

var getCloudflareIPRanges = sync.OnceValue(func() []netip.Prefix {
	path := common.EnvOr("CLOUDFLARE_IP_RANGE_FILE_PATH", "./cloudflare_ip_range.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(data), "\n")
	prefixes := make([]netip.Prefix, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		prefix, parseErr := netip.ParsePrefix(line)
		if parseErr != nil {
			panic(parseErr)
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes
})

func IsCloudflareIP(ip string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}

	for _, cidr := range getCloudflareIPRanges() {
		if cidr.Contains(addr) {
			return true
		}
	}
	return false
}
