package geoip

import (
	"fmt"
	"io"
	"net"
	"os"

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

func printMemoryUsage(prefix string) {
	// var memStats runtime.MemStats
	// runtime.ReadMemStats(&memStats)
	// totalMB := float64(memStats.Alloc) / 1024 / 1024
	// fmt.Fprintf(os.Stderr, "[%s] Total memory used: %.2f MB\n", prefix, totalMB)
}

func (g *GeoIPDatIn) Extract(ipType IPType) (list []string, err error) {
	printMemoryUsage("startup baseline")
	entries := make(map[string]*Entry)

	err = g.parseFile(g.URI, entries)

	printMemoryUsage("file parsed")
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
	printMemoryUsage("ip marshalled")

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
	printMemoryUsage("before read file")
	geoipBytes, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	printMemoryUsage("file read")

	entry := NewEntry("global")
	for code := range g.Want {
		var geoip GeoIP
		stripped := findCountryCode(geoipBytes, []byte(code))
		if stripped != nil {
			proto.Unmarshal(stripped, &geoip)

			printMemoryUsage("protobuf parsed")
			for _, v2rayCIDR := range geoip.Cidr {
				ipStr := net.IP(v2rayCIDR.GetIp()).String() + "/" + fmt.Sprint(v2rayCIDR.GetPrefix())
				if err := entry.AddPrefix(ipStr); err != nil {
					return err
				}
			}
			printMemoryUsage("entry updated")
		} else {
			// log.Println("code not found", code)
		}
	}
	printMemoryUsage("protobuf parsed")
	entries["global"] = entry

	return nil
}

// helpers

func findCountryCode(data, code []byte) []byte {
	codeL := len(code)
	if codeL == 0 {
		return nil
	}
	for {
		dataL := len(data)
		if dataL < 2 {
			return nil
		}
		x, y := decodeVarint(data[1:])
		if x == 0 && y == 0 {
			return nil
		}
		headL, bodyL := 1+y, int(x)
		dataL -= headL
		if dataL < bodyL {
			return nil
		}
		data = data[headL:]
		if int(data[1]) == codeL {
			for i := 0; i < codeL && data[2+i] == code[i]; i++ {
				if i+1 == codeL {
					return data[:bodyL]
				}
			}
		}
		if dataL == bodyL {
			return nil
		}
		data = data[bodyL:]
	}
}

func decodeVarint(buf []byte) (x uint64, n int) {
	for shift := uint(0); shift < 64; shift += 7 {
		if n >= len(buf) {
			return 0, 0
		}
		b := uint64(buf[n])
		n++
		x |= (b & 0x7F) << shift
		if (b & 0x80) == 0 {
			return x, n
		}
	}

	// The number is too large to represent in a 64-bit value.
	return 0, 0
}
