package geosite_antizapret

import (
	"github.com/sagernet/sing-box/common/geosite"
	"log"
	"net"
	"net/netip"
	"strings"
)

func ProcessRecords(records <-chan []string, cfg *Configs, IPsOut chan<- *net.IPNet, rulesOut chan<- geosite.Item) {
	var err error
	for rec := range records {

		// Process domain
		if rec[1] != "" {
			exclude := false

			for _, re := range *cfg.ExcludeRegexp {
				if re.MatchString(rec[1]) {
					exclude = true
					break
				}
			}

			if exclude {
				continue
			}

			if strings.HasPrefix(rec[1], "*") {

				rulesOut <- geosite.Item{
					Type:  geosite.RuleTypeDomain,
					Value: strings.Replace(rec[1], "*.", "", 1),
				}

				rulesOut <- geosite.Item{
					Type:  geosite.RuleTypeDomainSuffix,
					Value: strings.Replace(rec[1], "*", "", 1),
				}
			} else {
				rulesOut <- geosite.Item{
					Type:  geosite.RuleTypeDomain,
					Value: rec[1],
				}
			}
		}

		// Process IP addresses
		ips := strings.Split(rec[0], "|")

		for _, ipStr := range ips {
			if ipStr == "" {
				continue
			}

			var ipNet *net.IPNet
			if strings.Contains(ipStr, "/") {
				_, ipNet, err = net.ParseCIDR(ipStr)
				if err != nil {
					log.Println(err)
					continue
				}
			} else {
				addr, err := netip.ParseAddr(ipStr)
				if err != nil {
					log.Println(err)
					continue
				}

				exclude := false
				for _, ip := range cfg.ExcludeIPs {
					if addr.Compare(ip) == 0 {
						exclude = true
						break
					}
				}
				if exclude {
					break
				}

				ipNet = &net.IPNet{
					IP: addr.AsSlice(),
				}
				if addr.Is4() {
					ipNet.Mask = net.CIDRMask(32, 32)
				} else if addr.Is6() {
					ipNet.Mask = net.CIDRMask(128, 128)
				}
			}

			IPsOut <- ipNet
		}
	}
}
