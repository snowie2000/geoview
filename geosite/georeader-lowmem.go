package geosite

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/sagernet/sing/common"
	"github.com/snowie2000/geoview/srs"
)

type GSReaderLowMem struct {
	GSReader
}

// search for a domain in all geosite sites and return matched site codes
func (r *GSReaderLowMem) Lookup(domain string) ([]string, error) {
	matchedList := []string{}

	// try sing-box geosite
	geoReader, codes, err := LoadSingSiteFromFile(r.File)
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

	gcCounter := 0
	v2site, err := LoadV2SiteFromFile(r.File)
	if err == nil {
		defer v2site.Close()
		sitecodes := v2site.Codes()
		for _, code := range sitecodes {
			if geositeList, err := v2site.ReadSites([]string{code}, r.MustExist); err == nil && len(geositeList) > 0 {
				gcCounter++
				if len(geositeList[0].Domain) > 3000 {
					gcCounter += 100 // force GC for large domain sets
				}
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
			if gcCounter > 10 {
				runtime.GC()
				gcCounter = 0
			}
		}
		return matchedList, nil
	}

	return nil, fmt.Errorf("Not a valid geosite format")
}

func (r *GSReaderLowMem) Extract(wantList map[string][]string, regex bool) ([]string, error) {
	var geositeList []GeoSite
	// try sing-box geosite
	geoReader, codes, err := LoadSingSiteFromFile(r.File)
	if err == nil && len(codes) > 0 {
		domains, _, err := r.extractSingGeoSite(geoReader, codes, wantList, regex, false)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}

	v2site, err := LoadV2SiteFromFile(r.File)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		defer v2site.Close()
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
		if err != nil {
			return nil, err
		}
		domains, _, err := r.extractV2GeoSite(geositeList, wantList, regex, false)
		if err == nil {
			domains = common.Uniq(domains)
			sort.Strings(domains)
		}
		return domains, err
	}

	return nil, fmt.Errorf("Extract failed: %s", err.Error())
}

func (r *GSReaderLowMem) ToGeosite(wantList map[string][]string) (*GeoSiteList, error) {
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
	geoReader, codes, err := LoadSingSiteFromFile(r.File)
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
	v2site, err := LoadV2SiteFromFile(r.File)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		defer v2site.Close()
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
		if err != nil {
			return nil, err
		}
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
func (r *GSReaderLowMem) ToRuleSet(wantList map[string][]string, regex bool) (*srs.PlainRuleSetCompat, error) {
	var geositeList []GeoSite
	// try sing-box geosite
	geoReader, codes, err := LoadSingSiteFromFile(r.File)
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

	v2site, err := LoadV2SiteFromFile(r.File)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		defer v2site.Close()
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
		if err != nil {
			return nil, err
		}
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

func (r *GSReaderLowMem) ToQuantumultX(wantList map[string][]string) ([]string, error) {
	var geositeList []GeoSite
	// try sing-box geosite
	geoReader, codes, err := LoadSingSiteFromFile(r.File)
	if err == nil && len(codes) > 0 {
		_, itemlist, err := r.extractSingGeoSite(geoReader, codes, wantList, false, false)
		if err == nil {
			return itemToQxRule(itemlist)
		}
		return nil, err
	}

	v2site, err := LoadV2SiteFromFile(r.File)
	codes = []string{}
	for key := range wantList {
		codes = append(codes, key)
	}
	if err == nil {
		defer v2site.Close()
		geositeList, err = v2site.ReadSites(codes, r.MustExist)
		if err != nil {
			return nil, err
		}
		_, itemlist, err := r.extractV2GeoSite(geositeList, wantList, false, true)
		if err == nil {
			return itemToQxRule(itemlist)
		}
		return nil, err
	}
	return nil, fmt.Errorf("Convert to QuantumultX failed: %s", err.Error())
}
