package geosite_antizapret

import (
	"bufio"
	"encoding/json"
	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"os"
)

func GenerateRuleset(jsonPath string, binPath string, ips map[string]struct{}, domains map[string]struct{}, domainSuffixes map[string]struct{}) error {

	jsonFile, err := os.OpenFile(jsonPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	jsonWriter := bufio.NewWriter(jsonFile)
	defer jsonWriter.Flush()
	defer jsonFile.Close()

	binFile, err := os.OpenFile(binPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	binWriter := bufio.NewWriter(binFile)
	defer binWriter.Flush()
	defer binFile.Close()

	ruleSet := new(option.PlainRuleSetCompat)
	ruleSet.Version = 1
	ruleSet.Options.Rules = make([]option.HeadlessRule, 1)
	ruleSet.Options.Rules[0].Type = constant.RuleTypeDefault
	ruleSet.Options.Rules[0].DefaultOptions.IPCIDR = make([]string, 0)
	ruleSet.Options.Rules[0].DefaultOptions.Domain = make([]string, 0)
	ruleSet.Options.Rules[0].DefaultOptions.DomainSuffix = make([]string, 0)

	for ipNet := range ips {
		ruleSet.Options.Rules[0].DefaultOptions.IPCIDR = append(ruleSet.Options.Rules[0].DefaultOptions.IPCIDR, ipNet)
	}
	for domain := range domains {
		ruleSet.Options.Rules[0].DefaultOptions.Domain = append(ruleSet.Options.Rules[0].DefaultOptions.Domain, domain)
	}
	for suffix := range domainSuffixes {
		ruleSet.Options.Rules[0].DefaultOptions.DomainSuffix = append(ruleSet.Options.Rules[0].DefaultOptions.DomainSuffix, suffix)
	}

	enc := json.NewEncoder(jsonWriter)
	enc.SetIndent("", "  ")
	if err := enc.Encode(ruleSet); err != nil {
		return err
	}

	plainRuleSet := ruleSet.Upgrade()
	if err := srs.Write(binWriter, plainRuleSet); err != nil {
		return err
	}

	return nil
}
