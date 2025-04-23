package geosite

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/snowie2000/geoview/strmatcher"

	"github.com/sagernet/sing/common"
	"github.com/snowie2000/geoview/srs"
)

var matcherTypeMap = map[Domain_Type]strmatcher.Type{
	Domain_Plain:  strmatcher.Substr,
	Domain_Regex:  strmatcher.Regex,
	Domain_Domain: strmatcher.Domain,
	Domain_Full:   strmatcher.Full,
}

type GSHandler interface {
	Lookup(domain string) ([]string, error)
	Extract(wantList map[string][]string, regex bool) ([]string, error)
	ToGeosite(wantList map[string][]string) (*GeoSiteList, error)
	ToRuleSet(wantList map[string][]string, regex bool) (*srs.PlainRuleSetCompat, error)
	ToQuantumultX(wantList map[string][]string) ([]string, error)
}

func NewGeositeHandler(filename string, mustexist bool, lowmem bool) GSHandler {
	if lowmem {
		return &GSReaderLowMem{GSReader{filename, mustexist}}
	} else {
		return &GSReader{filename, mustexist}
	}
}

type GSReader struct {
	File      string
	MustExist bool
}

func (r *GSReader) extractV2GeoSite(geositeList []GeoSite, want map[string][]string, regex bool, keyword bool) (list []string, itemlist []Item, err error) {
	match := false
	for _, site := range geositeList {
		if v, ok := want[strings.ToUpper(site.CountryCode)]; !ok {
			log.Println(site.CountryCode, "not found", v)
		}
		if v, ok := want[strings.ToUpper(site.CountryCode)]; ok {
			domains := v2ItemToSing(site.Domain)
			for _, it := range domains {
				switch it.Type {
				case RuleTypeDomainRegex:
					if !regex {
						continue
					}
					fallthrough
				case RuleTypeDomainKeyword:
					if !keyword {
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

func (r *GSReader) extractSingGeoSite(geoReader *GeoSiteReader, codes []string, wantList map[string][]string, regex bool, keyword bool) (list []string, itemlist []Item, err error) {
	for code, _ := range wantList {
		// singbox rulec codes are always lowercased
		if item, err := geoReader.Read(strings.ToLower(code)); err == nil {
			for _, it := range item {
				switch it.Type {
				case RuleTypeDomainRegex:
					if !regex {
						continue
					}
					fallthrough
				case RuleTypeDomainKeyword:
					if !keyword {
						continue
					}
					fallthrough
				case RuleTypeDomain:
					list = append(list, it.Value)
					itemlist = append(itemlist, it)
				case RuleTypeDomainSuffix:
					// remove "." prefix for singbox rules
					if len(it.Value) > 0 && it.Value[0] == '.' {
						it.Value = it.Value[1:]
					}
					list = append(list, it.Value)
					itemlist = append(itemlist, it)
				}
			}
		} else if r.MustExist {
			return nil, nil, err
		}
	}
	return
}

func (r *GSReader) matchSiteAgainstList(domains []*Domain, domain string) (bool, error) {
	g := strmatcher.NewMphMatcherGroup()
	for _, d := range domains {
		matcherType, f := matcherTypeMap[d.Type]
		if !f {
			return false, errors.New("unsupported domain type")
		}
		_, err := g.AddPattern(d.Value, matcherType)
		if err != nil {
			return false, err
		}
	}
	g.Build()
	return len(g.Match(domain)) > 0, nil
}

// search for a domain in all geosite sites and return matched site codes
func (r *GSReader) Lookup(domain string) ([]string, error) {
	fileContent, err := os.ReadFile(r.File)
	if err != nil {
		return nil, err
	}
	matchedList := []string{}

	// try sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		for _, code := range codes {
			wantList := map[string][]string{
				code: nil,
			}
			if _, items, err := r.extractSingGeoSite(geoReader, []string{code}, wantList, true, true); err == nil {
				if ok, _ := r.matchSiteAgainstList(singItemToV2(items), domain); ok {
					matchedList = append(matchedList, code)
				}
			}
		}
		return matchedList, nil
	}

	v2site, err := LoadV2Site(fileContent)
	if err == nil {
		defer v2site.Close()
		sitecodes := v2site.Codes()
		for _, code := range sitecodes {
			if geositeList, err := v2site.ReadSites([]string{code}, r.MustExist); err == nil && len(geositeList) > 0 {
				if ok, _ := r.matchSiteAgainstList(geositeList[0].Domain, domain); ok {
					matchedList = append(matchedList, code)

					// now try extra matches with attributes
					groups := make(map[string][]*Domain) // separate into multiple domain groups with each attributes
					for _, domain := range geositeList[0].Domain {
						if domain.Attribute != nil && len(domain.Attribute) > 0 {
							for _, attr := range domain.Attribute {
								if g, ok := groups[attr.Key]; ok {
									groups[attr.Key] = append(g, domain)
								} else {
									groups[attr.Key] = []*Domain{domain}
								}
							}
						}
					}
					// now match against each sub group
					for attr, g := range groups {
						if ok, _ := r.matchSiteAgainstList(g, domain); ok {
							matchedList = append(matchedList, code+"@"+attr)
						}
					}
				}
			}
		}
		return matchedList, nil
	}

	return nil, fmt.Errorf("Not a valid geosite format")
}

func (r *GSReader) Extract(wantList map[string][]string, regex bool) ([]string, error) {
	fileContent, err := os.ReadFile(r.File)
	if err != nil {
		return nil, err
	}

	var geositeList []GeoSite
	// try sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		domains, _, err := r.extractSingGeoSite(geoReader, codes, wantList, regex, false)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}

	v2site, err := LoadV2Site(fileContent)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
	}
	if err == nil {
		defer v2site.Close()
		domains, _, err := r.extractV2GeoSite(geositeList, wantList, regex, false)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}

	return nil, fmt.Errorf("Extract failed: %s", err.Error())
}

func (r *GSReader) ToGeosite(wantList map[string][]string) (*GeoSiteList, error) {
	fileContent, err := os.ReadFile(r.File)
	if err != nil {
		return nil, err
	}
	codeList := make(map[string][]string)
	for c := range wantList {
		// remove duplicate codes, remove attributes
		sn := strings.SplitN(c, "@", 2)
		if _, ok := codeList[sn[0]]; !ok {
			codeList[sn[0]] = nil
		}
	}
	geolist := new(GeoSiteList)
	// sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		for _, code := range codes {
			if _, ok := codeList[strings.ToUpper(code)]; !ok {
				continue // skip unwanted codes
			}
			tmpList := make(map[string][]string)
			tmpList[code] = nil // the value never gets read
			_, itemlist, err := r.extractSingGeoSite(geoReader, codes, tmpList, true, true)
			if err == nil {
				// convert Item to geosite
				gs := &GeoSite{
					CountryCode: strings.ToUpper(code), // v2ray expects an uppercased country code
					Domain:      singItemToV2(itemlist),
				}
				geolist.Entry = append(geolist.Entry, gs)
			}
		}
		// Sort protoList so the marshaled list is reproducible
		sort.SliceStable(geolist.Entry, func(i, j int) bool {
			return geolist.Entry[i].CountryCode < geolist.Entry[j].CountryCode
		})
		return geolist, nil
	}

	var geositeList []GeoSite
	v2site, err := LoadV2Site(fileContent)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
		if err != nil {
			return nil, err
		}
		defer v2site.Close()
		for i := 0; i < len(geositeList); i++ {
			geolist.Entry = append(geolist.Entry, &geositeList[i])
		}
		// Sort protoList so the marshaled list is reproducible
		sort.SliceStable(geolist.Entry, func(i, j int) bool {
			return geolist.Entry[i].CountryCode < geolist.Entry[j].CountryCode
		})
		return geolist, nil
	}
	return nil, fmt.Errorf("Convert to geosite failed: %s", err.Error())
}

// to the ruleset json format of sing-box 1.20+
func (r *GSReader) ToRuleSet(wantList map[string][]string, regex bool) (*srs.PlainRuleSetCompat, error) {
	fileContent, err := os.ReadFile(r.File)
	if err != nil {
		return nil, err
	}

	var geositeList []GeoSite
	// try sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		_, itemlist, err := r.extractSingGeoSite(geoReader, codes, wantList, regex, false)
		if len(itemlist) == 0 {
			return nil, errors.New("empty domain set")
		}
		if err == nil {
			return itemToRuleset(itemlist)
		}
		return nil, err
	}

	v2site, err := LoadV2Site(fileContent)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
		if err != nil {
			return nil, err
		}
		defer v2site.Close()
		_, itemlist, err := r.extractV2GeoSite(geositeList, wantList, regex, false)
		if len(itemlist) == 0 {
			return nil, errors.New("empty domain set")
		}
		if err == nil {
			return itemToRuleset(itemlist)
		}
		return nil, err
	}
	return nil, fmt.Errorf("Convert to ruleset failed: %s", err.Error())
}

func (r *GSReader) ToQuantumultX(wantList map[string][]string) ([]string, error) {
	fileContent, err := os.ReadFile(r.File)
	if err != nil {
		return nil, err
	}
	var geositeList []GeoSite
	// try sing-box geosite
	geoReader, codes, err := LoadSingSite(fileContent)
	if err == nil && len(codes) > 0 {
		_, itemlist, err := r.extractSingGeoSite(geoReader, codes, wantList, false, false)
		if err == nil {
			return itemToQxRule(itemlist)
		}
		return nil, err
	}

	v2site, err := LoadV2Site(fileContent)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
		if err != nil {
			return nil, err
		}
		defer v2site.Close()
		_, itemlist, err := r.extractV2GeoSite(geositeList, wantList, false, true)
		if err == nil {
			return itemToQxRule(itemlist)
		}
		return nil, err
	}
	return nil, fmt.Errorf("Convert to QuantumultX failed: %s", err.Error())
}

func itemToQxRule(itemlist []Item) ([]string, error) {
	list := []string{}
	for _, it := range itemlist {
		switch it.Type {
		case RuleTypeDomain:
			{
				list = append(list, fmt.Sprintf("host,%s,Proxy", it.Value))
			}
		case RuleTypeDomainSuffix:
			{
				list = append(list, fmt.Sprintf("host-suffix,%s,Proxy", it.Value))
			}
		case RuleTypeDomainKeyword:
			{
				list = append(list, fmt.Sprintf("host-keyword,%s,Proxy", it.Value))
			}
		}
	}
	return list, nil
}

func itemToRuleset(itemlist []Item) (*srs.PlainRuleSetCompat, error) {
	ruleset := &srs.PlainRuleSetCompat{
		Version: srs.RuleSetVersionCurrent,
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
			// *ray and sing-box support both suffix w/o dot, we preserve what we get
			domain := it.Value
			rule.DefaultOptions.DomainSuffix = append(rule.DefaultOptions.DomainSuffix, domain)
		}
	}
	ruleset.Options.Rules = []srs.HeadlessRule{rule}
	return ruleset, nil
}

func singItemToV2(singItem []Item) []*Domain {
	list := []*Domain{}
	for _, item := range singItem {
		d := &Domain{}
		d.Value = item.Value
		switch item.Type {
		case RuleTypeDomain:
			d.Type = Domain_Full
		case RuleTypeDomainSuffix:
			d.Type = Domain_Domain
		case RuleTypeDomainKeyword:
			d.Type = Domain_Plain
		case RuleTypeDomainRegex:
			d.Type = Domain_Regex
		}
		list = append(list, d)
	}
	return list
}

func v2ItemToSing(v2Item []*Domain) []Item {
	var item Item
	items := []Item{}

	for _, domain := range v2Item {
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
		items = append(items, item)
	}

	return items
}
