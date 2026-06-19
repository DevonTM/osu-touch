package main

import (
	"log"
	"net"
	"strings"
)

func lanIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Could not list network interfaces: %v", err)
		return nil
	}

	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil || !isPrivateIPv4(ip4) {
				continue
			}
			ips = append(ips, ip4.String())
		}
	}
	return ips
}

func isPrivateIPv4(ip net.IP) bool {
	text := ip.String()
	return strings.HasPrefix(text, "10.") ||
		strings.HasPrefix(text, "192.168.") ||
		strings.HasPrefix(text, "172.16.") || strings.HasPrefix(text, "172.17.") ||
		strings.HasPrefix(text, "172.18.") || strings.HasPrefix(text, "172.19.") ||
		strings.HasPrefix(text, "172.20.") || strings.HasPrefix(text, "172.21.") ||
		strings.HasPrefix(text, "172.22.") || strings.HasPrefix(text, "172.23.") ||
		strings.HasPrefix(text, "172.24.") || strings.HasPrefix(text, "172.25.") ||
		strings.HasPrefix(text, "172.26.") || strings.HasPrefix(text, "172.27.") ||
		strings.HasPrefix(text, "172.28.") || strings.HasPrefix(text, "172.29.") ||
		strings.HasPrefix(text, "172.30.") || strings.HasPrefix(text, "172.31.")
}
