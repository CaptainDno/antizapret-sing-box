package geosite_antizapret

import (
	"bufio"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"net"
	"os"
)

func GenerateGeoip(path string, ips <-chan *net.IPNet, ipSet map[string]struct{}) error {

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()
	defer file.Close()

	mmdb, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType: "sing-geoip",
		Languages:    []string{"antizapret"},
	})

	if err != nil {
		return err
	}

	for ip := range ips {
		if err := mmdb.Insert(ip, mmdbtype.String("antizapret")); err != nil {
			return err
		}
		ipSet[ip.String()] = struct{}{}
	}

	if _, err := mmdb.WriteTo(writer); err != nil {
		return err
	}

	return nil
}
