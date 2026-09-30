package geoip

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
)

func parseIPOrCIDR(s string) (net.IP, net.IPMask, error) {
	// CIDR form
	if ip, network, err := net.ParseCIDR(s); err == nil {
		return ip, network.Mask, nil
	}

	// Plain IP
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, nil, fmt.Errorf("invalid IP or CIDR: %q", s)
	}

	if ip4 := ip.To4(); ip4 != nil {
		return ip4, net.CIDRMask(32, 32), nil
	}

	return ip, net.CIDRMask(128, 128), nil
}

func TxtToGeoIP(txt string) (*GeoIPList, error) {
	codeExp := regexp.MustCompile(`^\s*#\s*(.+?)\s*$`)
	file, err := os.Open(txt)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	codeMap := make(map[string]*GeoIP)
	allocCode := func(code string) *GeoIP {
		if _, ok := codeMap[code]; !ok {
			codeMap[code] = new(GeoIP)
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
		if ip, mask, err := parseIPOrCIDR(line); err == nil {
			var cidr CIDR
			ones, _ := mask.Size()
			cidr.Ip = ip
			cidr.Prefix = uint32(ones)
			currentMap.Cidr = append(currentMap.Cidr, &cidr)
		}
	}
	// save back to geoiplist
	ipList := new(GeoIPList)
	for _, ips := range codeMap {
		if len(ips.Cidr) > 0 {
			ipList.Entry = append(ipList.Entry, ips)
		}
	}
	return ipList, nil
}
