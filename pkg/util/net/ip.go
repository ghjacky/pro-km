package net

import (
	"net"
	"strings"
)

// IsIPv4 return true if a ip is valid IPV4
func IsIPv4(ip string) bool {
	return net.ParseIP(ip) != nil && !strings.Contains(ip, ":")
}
