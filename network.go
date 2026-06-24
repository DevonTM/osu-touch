package main

import (
	"bytes"
	"log"
	"net"
	"slices"
)

func logServerURLs(addr net.Addr) {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		log.Printf("URL: http://%s", addr.String())
		return
	}

	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		log.Printf("Local URL: http://localhost:%s", port)
		for _, ip := range lanIPs() {
			log.Printf("LAN URL: http://%s:%s", ip, port)
		}
		return
	}

	log.Printf("URL: http://%s", net.JoinHostPort(host, port))
}

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
			if ip4 == nil || !ip4.IsPrivate() {
				continue
			}
			ips = append(ips, ip4.String())
		}
	}
	sortLANIPs(ips)
	return ips
}

func sortLANIPs(ips []string) {
	slices.SortFunc(ips, func(leftIP, rightIP string) int {
		left := net.ParseIP(leftIP).To4()
		right := net.ParseIP(rightIP).To4()

		leftRank := privateIPv4Rank(left)
		rightRank := privateIPv4Rank(right)
		if leftRank != rightRank {
			return leftRank - rightRank
		}
		return compareIPv4(left, right)
	})
}

func privateIPv4Rank(ip net.IP) int {
	if len(ip) != net.IPv4len {
		return 3
	}
	if ip[0] == 192 && ip[1] == 168 {
		return 0
	}
	if ip[0] == 10 {
		return 1
	}
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return 2
	}
	return 3
}

func compareIPv4(left, right net.IP) int {
	if len(left) != net.IPv4len || len(right) != net.IPv4len {
		return 0
	}
	return bytes.Compare(left, right)
}
