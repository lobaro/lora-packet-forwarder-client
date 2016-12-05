package gateway

import (
	"net"
)

func addIPToGatewayStatsPacket(stat *GatewayStatsPacket, ip net.IP) {
	var ips []string
	if existingIPs, ok := stat.CustomData["ip"]; ok {
		if existingIPs, ok := existingIPs.([]string); ok {
			for _, existingIP := range existingIPs {
				if ip.String() == existingIP {
					return
				}
			}
			ips = append(existingIPs, ip.String())
		}
	}
	stat.CustomData["ip"] = ips
}
