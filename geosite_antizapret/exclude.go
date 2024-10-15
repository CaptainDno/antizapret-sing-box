package geosite_antizapret

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

func GetExcludedDomains(path string) (*[]regexp.Regexp, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	sc := bufio.NewScanner(file)

	var expressions []regexp.Regexp

	for sc.Scan() {
		if sc.Text()[0] == '#' {
			continue
		}
		re, err := regexp.Compile(sc.Text())
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		expressions = append(expressions, *re)
		fmt.Println("Excluding (regex): " + re.String())
	}

	err = sc.Err()
	if err != nil {
		return nil, err
	}

	err = file.Close()

	if err != nil {
		return nil, err
	}

	return &expressions, nil
}
