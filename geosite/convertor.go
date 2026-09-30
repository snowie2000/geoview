package geosite

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

func parseDomain(line string) *Domain {
	d := new(Domain)
	line = strings.TrimSpace(line)
	if line == "" {
		return nil // empty line
	}
	if strings.HasPrefix(line, "full:") {
		d.Type = Domain_Full
		line = strings.TrimLeft(line, "full:")
	} else {
		d.Type = Domain_Domain
	}
	seg := strings.Split(line, "@")
	if len(seg) == 0 {
		return nil // invalid line
	}
	for i, v := range seg {
		if i == 0 {
			d.Value = strings.ToLower(strings.TrimSpace(v))
		} else {
			// append attributes
			d.Attribute = append(d.Attribute, &Domain_Attribute{
				Key: strings.ToLower(strings.TrimSpace(v)),
			})
		}
	}
	return d
}

func TxtToGeosite(filename string) (*GeoSiteList, error) {
	codeExp := regexp.MustCompile(`^\s*#\s*(.+?)\s*$`)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	codeMap := make(map[string]*GeoSite)
	allocCode := func(code string) *GeoSite {
		if _, ok := codeMap[code]; !ok {
			codeMap[code] = new(GeoSite)
			codeMap[code].CountryCode = code
		}
		return codeMap[code]
	}

	currentCode := "ALL"
	currentMap := allocCode(currentCode)

	// 4. Loop through each line
	for scanner.Scan() {
		// scanner.Text() returns the line without the newline character
		line := scanner.Text()
		match := codeExp.FindStringSubmatch(line)
		if len(match) > 1 {
			currentCode = strings.ToUpper(strings.TrimSpace(match[1]))
			currentMap = allocCode(currentCode) // change what current map points to
			continue
		}
		if domain := parseDomain(line); domain != nil {
			currentMap.Domain = append(currentMap.Domain, domain)
		}
	}

	siteList := new(GeoSiteList)
	for _, domains := range codeMap {
		if len(domains.Domain) > 0 {
			siteList.Entry = append(siteList.Entry, domains)
		}
	}
	return siteList, nil
}
