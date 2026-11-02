package notify

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

func PublicHTTPS(ctx context.Context, raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("only a public https url")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if blockedHost(host) || digitsOnly(host) {
		return fmt.Errorf("internal host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if privateIP(ip) {
			return fmt.Errorf("internal address")
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("lookup host: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("host has no addresses")
	}
	for _, ip := range ips {
		if privateIP(ip.IP) {
			return fmt.Errorf("internal address")
		}
	}
	return nil
}

func blockedHost(host string) bool {
	switch host {
	case "localhost", "metadata.google.internal", "metadata.google.com":
		return true
	}
	return strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") ||
		strings.HasSuffix(host, ".cluster.local") ||
		host == "kubernetes.default" ||
		host == "kubernetes.default.svc"
}

func digitsOnly(host string) bool {
	if host == "" {
		return false
	}
	for _, r := range host {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func logHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "invalid"
	}
	return u.Host
}

func privateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if inner := embeddedV4(ip); inner != nil {
		return privateIP(inner)
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	if v4[0] == 0 || (v4[0] == 100 && v4[1]&0xc0 == 64) {
		return true
	}
	if v4[0] == 100 && v4[1] == 100 && v4[2] == 100 && v4[3] == 200 {
		return true
	}
	return false
}

func embeddedV4(ip net.IP) net.IP {
	ip = ip.To16()
	if ip == nil || ip.To4() != nil {
		return nil
	}
	switch {
	case ip[0] == 0x20 && ip[1] == 0x02:
		return net.IPv4(ip[2], ip[3], ip[4], ip[5])
	case ip[0] == 0x00 && ip[1] == 0x64 && ip[2] == 0xff && ip[3] == 0x9b && isZero(ip[4:12]):
		return net.IPv4(ip[12], ip[13], ip[14], ip[15])
	case ip[0] == 0x00 && ip[1] == 0x64 && ip[2] == 0xff && ip[3] == 0x9b && ip[4] == 0x00 && ip[5] == 0x01:
		return net.IPv4(ip[12], ip[13], ip[14], ip[15])
	case ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0 && ip[3] == 0:
		return net.IPv4(ip[12]^0xff, ip[13]^0xff, ip[14]^0xff, ip[15]^0xff)
	default:
		return nil
	}
}

func isZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func dialPublic(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	if ip := net.ParseIP(host); ip != nil {
		if privateIP(ip) {
			return nil, fmt.Errorf("internal address")
		}
		return dialer.DialContext(ctx, network, addr)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("lookup host: %w", err)
	}
	var last error
	for _, ip := range ips {
		if privateIP(ip.IP) {
			last = fmt.Errorf("internal address")
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("host has no addresses")
	}
	return nil, last
}
