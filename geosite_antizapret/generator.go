package geosite_antizapret

import (
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/sagernet/sing-box/common/geosite"
	"golang.org/x/text/encoding/charmap"
	"io"
	"net"
	"net/http"
	"path"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultDownloadURL = "https://raw.githubusercontent.com/zapret-info/z-i/master/dump.csv"

type Generator struct {
	downloadURL string
	httpClient  *http.Client
}

type GeneratorOption func(*Generator)

func WithDownloadURL(downloadURL string) GeneratorOption {
	return func(g *Generator) {
		g.downloadURL = downloadURL
	}
}

func WithHTTPClient(httpClient *http.Client) GeneratorOption {
	return func(g *Generator) {
		g.httpClient = httpClient
	}
}

func NewGenerator(opts ...GeneratorOption) *Generator {
	g := &Generator{
		downloadURL: DefaultDownloadURL,
		httpClient:  http.DefaultClient,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

func (g *Generator) generate(in io.Reader, geositePath, geoipPath, rulesetJSONPath, rulesetBinPath string) error {
	fmt.Println("Loading configuration files...")
	antizapretConfigs, err := g.fetchAntizapretConfigs()
	start := time.Now()
	if err != nil {
		return fmt.Errorf("cannot fetch antizapret configs: %w", err)
	}
	fmt.Printf("✓ Configs loaded\n\tExclusion regular expressions: %d\n\tAdditionally included domains: %d\n\tExcluded IP/subnets: %d\n",
		len(*antizapretConfigs.ExcludeRegexp), len(antizapretConfigs.IncludeHosts), len(antizapretConfigs.ExcludeIPs))

	// create csv reader with CP1251 decoder
	r := csv.NewReader(charmap.Windows1251.NewDecoder().Reader(in))
	r.Comma = ';'
	r.FieldsPerRecord = -1

	ipNetsSet := make(map[string]struct{})
	domainSuffixesSet := make(map[string]struct{})
	domainsSet := make(map[string]struct{})

	recordsChannel := make(chan []string, 5000)
	ipOutChannel := make(chan *net.IPNet, 5000)
	ruleOutChannel := make(chan geosite.Item, 5000)

	var included, excluded atomic.Uint64

	// Start parallel processing
	wgProc := sync.WaitGroup{}
	fmt.Printf("\nStarting %d initial processing routines...\n", runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		wgProc.Add(1)
		go func() {
			inc, exc := ProcessRecords(recordsChannel, antizapretConfigs, ipOutChannel, ruleOutChannel)
			included.Add(inc)
			excluded.Add(exc)
			wgProc.Done()
		}()
	}
	fmt.Println("✓ Done")
	wgGen := sync.WaitGroup{}

	// Generate geoip
	fmt.Println("\nStarting geoip.db generator...")
	wgGen.Add(1)
	go func() {
		if err := GenerateGeoip(geoipPath, ipOutChannel, ipNetsSet); err != nil {
			fmt.Errorf("cannot generate geoip: %w", err)
		}
		wgGen.Done()
		fmt.Printf("\n✓ geoip.db generated\n\tTime since start: %s\n", time.Since(start))
	}()
	fmt.Println("✓ Done")

	// Generate geosite
	wgGen.Add(1)
	fmt.Println("\nStarting geosite.db generator...")
	go func() {
		siteGenStart := time.Now()
		if err := GenerateGeosite(geositePath, ruleOutChannel, domainsSet, domainSuffixesSet); err != nil {
			fmt.Errorf("cannot generate geosite: %w", err)
		}
		wgGen.Done()
		fmt.Printf("\n✓ geosite.db generated\n\tTime since start: %s\n", time.Since(siteGenStart))
	}()
	fmt.Println("✓ Done")

	go func() {
		// Additional domains
		for _, host := range antizapretConfigs.IncludeHosts {
			ruleOutChannel <- geosite.Item{
				Type:  geosite.RuleTypeDomainSuffix,
				Value: "." + host,
			}

			ruleOutChannel <- geosite.Item{
				Type:  geosite.RuleTypeDomain,
				Value: host,
			}
		}
		wgProc.Wait()

		//Close all channels
		close(ruleOutChannel)
		close(ipOutChannel)

		fmt.Printf("\n✓ Initial processing finished.\n\tRecords included: %d\n\tRecords excluded: %d\n\tAddtitional hosts: %d\n\tTime since start: %s\n",
			included.Load(), excluded.Load(), len(antizapretConfigs.IncludeHosts), time.Since(start))
	}()

	first := true
	fmt.Println("\nReading CSV...")

	recordsRead := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("cannot parse csv file: %w", err)
		}

		if len(rec) < 2 {
			if first {
				first = false
				continue
			}
			return errors.New("something wrong with csv")
		}

		recordsChannel <- rec
		recordsRead++
	}
	// Close records channel
	close(recordsChannel)

	fmt.Printf("\n✓ CSV processed.\n\tTotal record count: %d\n\tTime since start: %s\n", recordsRead, time.Since(start))

	// Wait for sets to populate
	wgGen.Wait()

	fmt.Printf("\nReady to generate ruleset.\n\tIPs/subnets total: %d\n\tDomains (exact) total: %d\n\tDomains (suffixes) total: %d\n", len(ipNetsSet), len(domainsSet), len(domainSuffixesSet))

	// Rule Set
	fmt.Println("\nGenerating ruleset")
	err = GenerateRuleset(rulesetJSONPath, rulesetBinPath, ipNetsSet, domainsSet, domainSuffixesSet)
	if err != nil {
		return err
	}
	fmt.Printf("✓ Generated ruleset\n\tTime since start: %s\n", time.Since(start))
	return nil
}

func (g *Generator) GenerateAndWrite(outputBasePath string) error {
	resp, err := g.httpClient.Get(g.downloadURL)
	if err != nil {
		return fmt.Errorf("cannot get dump from github: %w", err)
	}
	defer resp.Body.Close()

	return g.generate(
		resp.Body,
		path.Join(outputBasePath, "geosite.db"),
		path.Join(outputBasePath, "geoip.db"),
		path.Join(outputBasePath, "ruleset.json"),
		path.Join(outputBasePath, "ruleset.srs"),
	)
}
