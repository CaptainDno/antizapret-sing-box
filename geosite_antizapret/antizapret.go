package geosite_antizapret

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/netip"
	"regexp"
	"strings"
)

const (
	AntizapretPACGeneratorLightUpstreamBaseURL = "https://bitbucket.org/anticensority/antizapret-pac-generator-light/raw/master/config/"

	ExcludeHostsByIPsDist = "exclude-hosts-by-ips-dist.txt"
	ExcludeHostsDist      = "exclude-hosts-dist.txt"
	ExcludeIPsDist        = "exclude-ips-dist.txt"
	ExcludeRegexpDist     = "exclude-regexp-dist.awk"

	IncludeHostsDist = "include-hosts-dist.txt"
	IncludeIPsDist   = "include-ips-dist.txt"
)

type AntizapretConfigType string

const (
	IPs        AntizapretConfigType = "ips"
	Hosts      AntizapretConfigType = "hosts"
	HostsByIPs AntizapretConfigType = "hosts_by_ips"
	Regexp     AntizapretConfigType = "regexp"
)

type AntizapretConfig struct {
	Type    AntizapretConfigType
	Exclude bool
	URL     string
}

type Configs struct {
	ExcludeHosts  []string
	ExcludeIPs    []netip.Addr
	ExcludeRegexp *[]regexp.Regexp

	IncludeHosts []string
	IncludeIPs   []*net.IPNet
}

func (g *Generator) fetchAntizapretConfigs() (*Configs, error) {
	configs := &Configs{}
	var err error

	// Get Local files
	configs.ExcludeRegexp, err = GetExcludedDomains("excluded.txt")

	if err != nil {
		return nil, fmt.Errorf("cannot get exclude domains: %w", err)
	}

	configs.IncludeHosts, err = GetIncludedDomains("included.txt")
	if err != nil {
		return nil, fmt.Errorf("cannot get included domains: %w", err)
	}

	// Get excluded IPs
	resp, err := g.httpClient.Get(AntizapretPACGeneratorLightUpstreamBaseURL + ExcludeHostsByIPsDist)

	if err != nil {
		return nil, fmt.Errorf("cannot get antizapret exclude config: %w", err)
	}

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		ipStr := scanner.Text()
		ipStr = strings.ReplaceAll(ipStr, "\\", "")
		ipStr = strings.Replace(ipStr, "^", "", 1)
		ipStr = strings.Replace(ipStr, ";", "", 1)
		addr, err := netip.ParseAddr(ipStr)
		if err != nil {
			log.Println(err)
			continue
		}

		configs.ExcludeIPs = append(configs.ExcludeIPs, addr)
	}
	return configs, nil
}
