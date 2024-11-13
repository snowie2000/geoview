package geosite

import (
	"fmt"
	"os"
	"sort"
	"strings"

	singCst "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
)

func extractV2GeoSite(geositeList []*GeoSite, wantList []string, regex bool) (list []string, itemlist []Item, err error) {
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
					itemlist = append(itemlist, it)
				}
			}
		}
	}
	return
}

func extractSingGeoSite(geoReader *GeoSiteReader, codes []string, wantList []string, regex bool) (list []string, itemlist []Item, err error) {
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
					itemlist = append(itemlist, it)
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
func ToRuleSet(file string, wantList []string, regex bool) (*option.PlainRuleSetCompat, error) {
	fileContent, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var geositeList []*GeoSite
	geositeList, err = LoadV2Site(fileContent)
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

func itemToRuleset(itemlist []Item) (*option.PlainRuleSetCompat, error) {
	ruleset := &option.PlainRuleSetCompat{
		Version: singCst.RuleSetVersion1,
	}
	rule := option.HeadlessRule{
		Type: singCst.RuleTypeDefault,
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
	ruleset.Options.Rules = []option.HeadlessRule{rule}
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
		domains = append(domains, item)

		// for _, attribute := range domain.Attribute {
		// 	entry.WriteString(" @" + attribute.Key)
		// }
	}

	return domains
}
