package utils

import (
	"fmt"
	"net"
	"net/url"
)

// IsSSRFSafe checks if a remote destination URL is pointing to a forbidden private or local IP range.
func IsSSRFSafe(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	// Resolve the host to IP addresses
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return fmt.Errorf("could not resolve host %s: %w", u.Hostname(), err)
	}

	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
			return fmt.Errorf("Security Block (SSRF): Destination IP %s is not allowed (private/local network range)", ip)
		}
	}

	return nil
}
