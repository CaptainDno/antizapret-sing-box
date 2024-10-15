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
	antizapretConfigs, err := g.fetchAntizapretConfigs()
	if err != nil {
		return fmt.Errorf("cannot fetch antizapret configs: %w", err)
	}
	fmt.Println("Configs loaded.")

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

	// Start parallel processing
	wgProc := sync.WaitGroup{}
	for i := 0; i < runtime.NumCPU(); i++ {
		wgProc.Add(1)
		go func() {
			ProcessRecords(recordsChannel, antizapretConfigs, ipOutChannel, ruleOutChannel)
			wgProc.Done()
		}()
		fmt.Println("Processor started")
	}

	wgGen := sync.WaitGroup{}

	// Generate geoip
	fmt.Println("Generating geoip.db")
	wgGen.Add(1)
	go func() {
		if err := GenerateGeoip(geoipPath, ipOutChannel, ipNetsSet); err != nil {
			fmt.Errorf("cannot generate geoip: %w", err)
		}
		wgGen.Done()
	}()

	// Generate geosite
	wgGen.Add(1)
	fmt.Println("Generating geosite.db")
	go func() {
		if err := GenerateGeosite(geositePath, ruleOutChannel, domainsSet, domainSuffixesSet); err != nil {
			fmt.Errorf("cannot generate geosite: %w", err)
		}
		wgGen.Done()
	}()

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

		fmt.Println("Initial processing finished.")
	}()

	first := true
	fmt.Println("Reading CSV")
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
	}
	// Close records channel
	close(recordsChannel)

	fmt.Println("CSV read finished")

	// Wait for sets to populate
	wgGen.Wait()

	// Rule Set
	fmt.Println("Generating ruleset")
	return GenerateRuleset(rulesetJSONPath, rulesetBinPath, ipNetsSet, domainsSet, domainSuffixesSet)
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
