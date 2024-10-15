package geosite_antizapret

import (
	"bufio"
	"fmt"
	"github.com/sagernet/sing-box/common/geosite"
	"os"
)

func GenerateGeosite(path string, rules <-chan geosite.Item, domains map[string]struct{}, domainSuffixes map[string]struct{}) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()
	defer file.Close()

	arr := make([]geosite.Item, 500)

	for rule := range rules {
		switch rule.Type {
		case geosite.RuleTypeDomain:
			domains[rule.Value] = struct{}{}
			break
		case geosite.RuleTypeDomainSuffix:
			domainSuffixes[rule.Value] = struct{}{}
			break
		default:
			fmt.Printf("Unsupported rule type: %d\n", rule.Type)
		}

		arr = append(arr, rule)
	}

	if err := geosite.Write(writer, map[string][]geosite.Item{
		"antizapret": arr,
	}); err != nil {
		return err
	}
	return nil
}
