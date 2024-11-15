package geoip

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"

	"github.com/snowie2000/geoview/protohelper"
	"github.com/snowie2000/geoview/srs"
	"google.golang.org/protobuf/proto"
)

type GeoIPDatIn struct {
	URI  string
	Want map[string]bool
}

type IPType int

const (
	IPv4 IPType = 1
	IPv6 IPType = 2
)

func (g *GeoIPDatIn) Extract(ipType IPType) (list []string, err error) {
	entries := make(map[string]*Entry)

	err = g.parseFile(g.URI, entries)

	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no match countrycode found")
	}

	var ignoreIPType IgnoreIPOption
	if ipType&IPv4 == 0 {
		ignoreIPType = IgnoreIPv4
	}
	if ipType&IPv6 == 0 {
		ignoreIPType = IgnoreIPv6
	}

	for _, entry := range entries {
		if t, err := entry.MarshalText(ignoreIPType); err == nil && t != nil {
			list = append(list, t...)
		}
	}

	/*
		var ranges []cidr.IRange
		for _, v := range list {
			if r, err := cidr.ParseRange(v); err == nil {
				ranges = append(ranges, r)
			}
		}
		ranges = cidr.SortAndMerge(ranges)
		list = nil
		for _, r := range ranges {
			for _, n := range r.ToIpNets() {
				list = append(list, n.String())
			}
		}
	*/
	return
}

func (g *GeoIPDatIn) ToRuleSet(ipType IPType) (*srs.PlainRuleSetCompat, error) {
	cidrlist, err := g.Extract(ipType)
	if err != nil {
		return nil, err
	}

	ruleset := &srs.PlainRuleSetCompat{
		Version: srs.RuleSetVersion1,
	}
	rule := srs.HeadlessRule{
		Type: srs.RuleTypeDefault,
	}
	rule.DefaultOptions.IPCIDR = cidrlist
	ruleset.Options.Rules = []srs.HeadlessRule{rule}
	return ruleset, nil
}

func (g *GeoIPDatIn) parseFile(path string, entries map[string]*Entry) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := g.generateEntries(file, entries); err != nil {
		return err
	}

	return nil
}

func (g *GeoIPDatIn) generateEntries(reader io.Reader, entries map[string]*Entry) error {
	geoipBytes, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	ipStrList := make([]string, 0)
	for code := range g.Want {
		var geoip GeoIP
		stripped := protohelper.FindCode(geoipBytes, []byte(code))
		if stripped != nil {
			proto.Unmarshal(stripped, &geoip)

			for _, v2rayCIDR := range geoip.Cidr {
				ipStr := net.IP(v2rayCIDR.GetIp()).String() + "/" + fmt.Sprint(v2rayCIDR.GetPrefix())
				ipStrList = append(ipStrList, ipStr)
			}
		} else {
			// log.Println("code not found", code)
		}
	}

	geoipBytes = nil
	runtime.GC()

	entry := NewEntry("global")
	counter := 0
	for _, ip := range ipStrList {
		if err := entry.AddPrefix(ip); err != nil {
			return err
		}
		if counter++; counter > 10000 {
			runtime.GC()
			counter = 0
		}
	}
	entries["global"] = entry
	return nil
}
