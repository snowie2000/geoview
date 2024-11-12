package geoip

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

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
		return nil, fmt.Errorf("no entry is generated")
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

	return
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

	var geoipList GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return err
	}

	for _, geoip := range geoipList.Entry {
		name := strings.ToUpper(strings.TrimSpace(geoip.CountryCode))

		if len(g.Want) > 0 && !g.Want[name] {
			continue
		}

		entry, found := entries[name]
		if !found {
			entry = NewEntry(name)
		}

		for _, v2rayCIDR := range geoip.Cidr {
			ipStr := net.IP(v2rayCIDR.GetIp()).String() + "/" + fmt.Sprint(v2rayCIDR.GetPrefix())
			if err := entry.AddPrefix(ipStr); err != nil {
				return err
			}
		}

		entries[name] = entry
	}

	return nil
}
