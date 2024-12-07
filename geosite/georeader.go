package geosite

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/sagernet/sing/common"
	"github.com/snowie2000/geoview/srs"
)

func extractV2GeoSite(geositeList []GeoSite, want map[string][]string, regex bool) (list []string, itemlist []Item, err error) {
	match := false
	for _, site := range geositeList {
		if v, ok := want[strings.ToUpper(site.CountryCode)]; !ok {
			log.Println(site.CountryCode, "not found", v)
		}
		if v, ok := want[strings.ToUpper(site.CountryCode)]; ok {
			domains := processGeositeEntry(&site)
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
					// check attr
					match = true
					for _, attr := range v {
						if it.Attr == nil {
							match = false
							break
						}
						if _, ok := it.Attr[attr]; !ok {
							match = false
							break // ignore domains with wrong attributes
						}
					}
					if match {
						list = append(list, it.Value)
						itemlist = append(itemlist, it)
					}
				}
			}
		}
	}
	return
}

func extractSingGeoSite(geoReader *GeoSiteReader, codes []string, wantList map[string][]string, regex bool) (list []string, itemlist []Item, err error) {
	for code, _ := range wantList {
		if item, err := geoReader.Read(code); err == nil {
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
					itemlist = append(itemlist, it)
				}
			}
		}
	}
	return
}

func Extract(file string, wantList map[string][]string, regex bool) ([]string, error) {
	fileContent, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var geositeList []GeoSite
	geositeList, err = LoadV2Site(fileContent, wantList)
	if err == nil {
		domains, _, err := extractV2GeoSite(geositeList, wantList, regex)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}

	// try sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		domains, _, err := extractSingGeoSite(geoReader, codes, wantList, regex)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}
	return nil, fmt.Errorf("Not a valid geosite format")
}

// to the ruleset json format of sing-box 1.20+
func ToRuleSet(file string, wantList map[string][]string, regex bool) (*srs.PlainRuleSetCompat, error) {
	fileContent, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var geositeList []GeoSite
	geositeList, err = LoadV2Site(fileContent, wantList)
	if err == nil {
		_, itemlist, err := extractV2GeoSite(geositeList, wantList, regex)
		if err == nil {
			return itemToRuleset(itemlist)
		}
		return nil, err
	}

	// try sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		_, itemlist, err := extractSingGeoSite(geoReader, codes, wantList, regex)
		if err == nil {
			return itemToRuleset(itemlist)
		}
		return nil, err
	}
	return nil, fmt.Errorf("Not a valid geosite format")
}

func itemToRuleset(itemlist []Item) (*srs.PlainRuleSetCompat, error) {
	ruleset := &srs.PlainRuleSetCompat{
		Version: srs.RuleSetVersion1,
	}
	rule := srs.HeadlessRule{
		Type: srs.RuleTypeDefault,
	}
	for _, it := range itemlist {
		switch it.Type {
		case RuleTypeDomain:
			rule.DefaultOptions.Domain = append(rule.DefaultOptions.Domain, it.Value)
		case RuleTypeDomainKeyword:
			rule.DefaultOptions.DomainKeyword = append(rule.DefaultOptions.DomainKeyword, it.Value)
		case RuleTypeDomainRegex:
			rule.DefaultOptions.DomainRegex = append(rule.DefaultOptions.DomainRegex, it.Value)
		case RuleTypeDomainSuffix:
			rule.DefaultOptions.DomainSuffix = append(rule.DefaultOptions.DomainSuffix, it.Value)
		}
	}
	ruleset.Options.Rules = []srs.HeadlessRule{rule}
	return ruleset, nil
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
		for _, attr := range domain.Attribute {
			if item.Attr == nil {
				item.Attr = make(map[string]struct{})
			}
			item.Attr[attr.Key] = struct{}{}
		}
		domains = append(domains, item)
	}

	return domains
}
