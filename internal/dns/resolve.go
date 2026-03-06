package dns

import (
	"context"
	"errors"
	"net"
)

// Resolve looks up the IPv4 and IPv6 addresses for dnsName.
// The provided ctx controls the lifetime of the lookup; callers should set
// a deadline to avoid blocking indefinitely.
func Resolve(ctx context.Context, dnsName string) ([]net.IP, []net.IP, error) {
	if dnsName == "" {
		return nil, nil, errors.New("dnsName is empty")
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, dnsName)
	if err != nil {
		return nil, nil, err
	}
	var ipv4List, ipv6List []net.IP
	for _, addr := range addrs {
		if v4 := addr.IP.To4(); v4 != nil {
			ipv4List = append(ipv4List, v4)
		} else {
			ipv6List = append(ipv6List, addr.IP.To16())
		}
	}
	return ipv4List, ipv6List, nil
}
