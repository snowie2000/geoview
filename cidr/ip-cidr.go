// ip-cidr
package cidr

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

type OutputType byte

const (
	OutputTypeCidr OutputType = iota + 1
	OutputTypeRange
	OutputTypeSum = OutputTypeCidr + OutputTypeRange
)

func ParseRange(text string) (IRange, error) {
	if index := strings.IndexByte(text, '/'); index != -1 {
		if _, network, err := net.ParseCIDR(text); err == nil {
			return IpNetWrapper{network}, nil
		} else {
			return nil, err
		}
	}
	if ip := parseIp(text); ip != nil {
		return IpWrapper{ip}, nil
	}
	if index := strings.IndexByte(text, '-'); index != -1 {
		if start, end := parseIp(text[:index]), parseIp(text[index+1:]); start != nil && end != nil {
			if len(start) == len(end) && !lessThan(end, start) {
				return &Range{start: start, end: end}, nil
			}
		}
		return nil, &net.ParseError{Type: "range", Text: text}
	}
	return nil, &net.ParseError{Type: "ip/CIDR address/range", Text: text}
}

func SortAndMerge(wrappers []IRange) []IRange {
	if len(wrappers) < 2 {
		return wrappers
	}
	ranges := make([]*Range, 0, len(wrappers))
	for _, e := range wrappers {
		ranges = append(ranges, e.ToRange())
	}
	sort.Sort(Ranges(ranges))

	res := make([]IRange, 0, len(ranges))
	now := ranges[0]
	familyLength := now.familyLength()
	start, end := now.start, now.end
	for i, count := 1, len(ranges); i < count; i++ {
		now := ranges[i]
		if fl := now.familyLength(); fl != familyLength {
			res = append(res, &Range{start, end})
			familyLength = fl
			start, end = now.start, now.end
			continue
		}
		if allFF(end) || !lessThan(addOne(end), now.start) {
			if lessThan(end, now.end) {
				end = now.end
			}
		} else {
			res = append(res, &Range{start, end})
			start, end = now.start, now.end
		}
	}
	return append(res, &Range{start, end})
}

func MarshalText(wrappers []IRange, outputType OutputType) []IRange {
	result := make([]IRange, 0, len(wrappers))
	if outputType == OutputTypeRange {
		for _, r := range wrappers {
			result = append(result, r.ToRange())
		}
	} else {
		for _, r := range wrappers {
			for _, ipNet := range r.ToIpNets() {
				// can't use range iterator, for operator address of is taken
				// it seems a trick of golang here
				result = append(result, IpNetWrapper{ipNet})
			}
		}
	}
	return result
}

func parseIp(str string) net.IP {
	for _, b := range str {
		switch b {
		case '.':
			return net.ParseIP(str).To4()
		case ':':
			return net.ParseIP(str).To16()
		}
	}
	return nil
}

func assert(condition bool, format string, args ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assert failed: "+format, args...))
	}
}
