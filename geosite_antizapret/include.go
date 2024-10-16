package geosite_antizapret

import (
	"bufio"
	"os"
)

func GetIncludedDomains(path string) ([]string, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	sc := bufio.NewScanner(file)

	var domains []string

	for sc.Scan() {
		if sc.Text()[0] == '#' {
			continue
		}
		domains = append(domains, sc.Text())
	}

	err = sc.Err()
	if err != nil {
		return nil, err
	}

	err = file.Close()

	if err != nil {
		return nil, err
	}

	return domains, nil
}
