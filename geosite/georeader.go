package geosite

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sagernet/sing/common"
)

func extractV2GeoSite(geositeList []*GeoSite, wantList []string, regex bool) (list []string, err error) {
	want := make(map[string]bool)
	for _, v := range wantList {
		want[strings.ToUpper(v)] = true
	}

	for _, site := range geositeList {
		if v, ok := want[strings.ToUpper(site.CountryCode)]; ok && v {
			domains := processGeositeEntry(site)
			for _, it := range domains {
				switch it.Type {
				case RuleTypeDomainRegex:
					if !regex {
						continue
					}
					fallthrough
				case RuleTypeDomain:
					fallthrough
				case RuleTypeDomainSuffix:
					list = append(list, it.Value)
				}
			}
		}
	}
	return
}

func extractSingGeoSite(geoReader *GeoSiteReader, codes []string, wantList []string, regex bool) (list []string, err error) {
	for _, v := range wantList {
		if item, err := geoReader.Read(v); err == nil {
			for _, it := range item {
				switch it.Type {
				case RuleTypeDomainRegex:
					if !regex {
						continue
					}
					fallthrough
				case RuleTypeDomain:
					fallthrough
				case RuleTypeDomainSuffix:
					list = append(list, it.Value)
				}
			}
		}
	}
	return
}

func Extract(file string, wantList []string, regex bool) ([]string, error) {
	fileContent, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var geositeList []*GeoSite
	geositeList, err = LoadV2Site(fileContent)
	if err == nil {
		domains, err := extractV2GeoSite(geositeList, wantList, regex)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}

	// try sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		domains, err := extractSingGeoSite(geoReader, codes, wantList, regex)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}
	return nil, fmt.Errorf("Not a valid geosite format")
}

func processGeositeEntry(vGeositeEntry *GeoSite) []Item {
	var domains []Item
	var item Item

	for _, domain := range vGeositeEntry.Domain {
		item.Value = domain.Value
		switch domain.Type {
		case Domain_Full:
			item.Type = RuleTypeDomain
		case Domain_Regex:
			item.Type = RuleTypeDomainRegex
		case Domain_Domain:
			item.Type = RuleTypeDomainSuffix
		case Domain_Plain:
			item.Type = RuleTypeDomainKeyword
		default:
			item.Type = RuleTypeDomain
		}
		domains = append(domains, item)

		// for _, attribute := range domain.Attribute {
		// 	entry.WriteString(" @" + attribute.Key)
		// }
	}

	return domains
}
