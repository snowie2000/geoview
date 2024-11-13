package geoip

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"go4.org/netipx"
)

type IPIgnoreType string

const (
	IgIPv4 IPIgnoreType = "IgIPv4"
	IgIPv6 IPIgnoreType = "IgIPv6"
)

type IgnoreIPOption func() IPIgnoreType

func IgnoreIPv4() IPIgnoreType {
	return IgIPv4
}

func IgnoreIPv6() IPIgnoreType {
	return IgIPv6
}

var (
	ErrDuplicatedConverter = errors.New("duplicated converter")
	ErrUnknownAction       = errors.New("unknown action")
	ErrNotSupportedFormat  = errors.New("not supported format")
	ErrInvalidIPType       = errors.New("invalid IP type")
	ErrInvalidIP           = errors.New("invalid IP address")
	ErrInvalidIPLength     = errors.New("invalid IP address length")
	ErrInvalidIPNet        = errors.New("invalid IPNet address")
	ErrInvalidCIDR         = errors.New("invalid CIDR")
	ErrInvalidPrefix       = errors.New("invalid prefix")
	ErrInvalidPrefixType   = errors.New("invalid prefix type")
	ErrCommentLine         = errors.New("comment line")
)

type Entry struct {
	name        string
	IPv4Builder *netipx.IPSetBuilder
	IPv6Builder *netipx.IPSetBuilder
	IPv4Set     *netipx.IPSet
	IPv6Set     *netipx.IPSet
}

func NewEntry(name string) *Entry {
	return &Entry{
		name: strings.ToUpper(strings.TrimSpace(name)),
	}
}

func (e *Entry) GetName() string {
	return e.name
}

func (e *Entry) hasIPv4Builder() bool {
	return e.IPv4Builder != nil
}

func (e *Entry) hasIPv6Builder() bool {
	return e.IPv6Builder != nil
}

func (e *Entry) hasIPv4Set() bool {
	return e.IPv4Set != nil
}

func (e *Entry) hasIPv6Set() bool {
	return e.IPv6Set != nil
}

func (e *Entry) GetIPv4Set() (*netipx.IPSet, error) {
	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	if e.hasIPv4Set() {
		return e.IPv4Set, nil
	}

	return nil, fmt.Errorf("entry %s has no IgIPv4 set", e.GetName())
}

func (e *Entry) GetIPv6Set() (*netipx.IPSet, error) {
	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	if e.hasIPv6Set() {
		return e.IPv6Set, nil
	}

	return nil, fmt.Errorf("entry %s has no IgIPv6 set", e.GetName())
}

func (e *Entry) processPrefix(src any) (*netip.Prefix, IPIgnoreType, error) {
	switch src := src.(type) {
	case net.IP:
		ip, ok := netipx.FromStdIP(src)
		if !ok {
			return nil, "", ErrInvalidIP
		}
		ip = ip.Unmap()
		switch {
		case ip.Is4():
			prefix := netip.PrefixFrom(ip, 32)
			return &prefix, IgIPv4, nil
		case ip.Is6():
			prefix := netip.PrefixFrom(ip, 128)
			return &prefix, IgIPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *net.IPNet:
		prefix, ok := netipx.FromStdIPNet(src)
		if !ok {
			return nil, "", ErrInvalidIPNet
		}
		ip := prefix.Addr().Unmap()
		switch {
		case ip.Is4():
			return &prefix, IgIPv4, nil
		case ip.Is6():
			return &prefix, IgIPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case netip.Addr:
		src = src.Unmap()
		switch {
		case src.Is4():
			prefix := netip.PrefixFrom(src, 32)
			return &prefix, IgIPv4, nil
		case src.Is6():
			prefix := netip.PrefixFrom(src, 128)
			return &prefix, IgIPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *netip.Addr:
		*src = (*src).Unmap()
		switch {
		case src.Is4():
			prefix := netip.PrefixFrom(*src, 32)
			return &prefix, IgIPv4, nil
		case src.Is6():
			prefix := netip.PrefixFrom(*src, 128)
			return &prefix, IgIPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case netip.Prefix:
		ip := src.Addr()
		switch {
		case ip.Is4():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IgIPv4, nil
		case ip.Is4In6():
			ip = ip.Unmap()
			bits := src.Bits()
			if bits < 96 {
				return nil, "", ErrInvalidPrefix
			}
			prefix, err := ip.Prefix(bits - 96)
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IgIPv4, nil
		case ip.Is6():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IgIPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *netip.Prefix:
		ip := src.Addr()
		switch {
		case ip.Is4():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IgIPv4, nil
		case ip.Is4In6():
			ip = ip.Unmap()
			bits := src.Bits()
			if bits < 96 {
				return nil, "", ErrInvalidPrefix
			}
			prefix, err := ip.Prefix(bits - 96)
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IgIPv4, nil
		case ip.Is6():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IgIPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case string:
		src, _, _ = strings.Cut(src, "#")
		src, _, _ = strings.Cut(src, "//")
		src, _, _ = strings.Cut(src, "/*")
		src = strings.TrimSpace(src)
		if src == "" {
			return nil, "", ErrCommentLine
		}

		switch strings.Contains(src, "/") {
		case true: // src is CIDR notation
			ip, network, err := net.ParseCIDR(src)
			if err != nil {
				return nil, "", ErrInvalidCIDR
			}
			addr, ok := netipx.FromStdIP(ip)
			if !ok {
				return nil, "", ErrInvalidIP
			}
			if addr.Unmap().Is4() && strings.Contains(network.String(), "::") { // src is invalid IgIPv4-mapped IgIPv6 address
				return nil, "", ErrInvalidCIDR
			}
			prefix, ok := netipx.FromStdIPNet(network)
			if !ok {
				return nil, "", ErrInvalidIPNet
			}

			addr = prefix.Addr().Unmap()
			switch {
			case addr.Is4():
				return &prefix, IgIPv4, nil
			case addr.Is6():
				return &prefix, IgIPv6, nil
			default:
				return nil, "", ErrInvalidIPLength
			}

		case false: // src is IP address
			ip, err := netip.ParseAddr(src)
			if err != nil {
				return nil, "", ErrInvalidIP
			}
			ip = ip.Unmap()
			switch {
			case ip.Is4():
				prefix := netip.PrefixFrom(ip, 32)
				return &prefix, IgIPv4, nil
			case ip.Is6():
				prefix := netip.PrefixFrom(ip, 128)
				return &prefix, IgIPv6, nil
			default:
				return nil, "", ErrInvalidIPLength
			}
		}
	}

	return nil, "", ErrInvalidPrefixType
}

func (e *Entry) add(prefix *netip.Prefix, ipType IPIgnoreType) error {
	switch ipType {
	case IgIPv4:
		if !e.hasIPv4Builder() {
			e.IPv4Builder = new(netipx.IPSetBuilder)
		}
		e.IPv4Builder.AddPrefix(*prefix)
	case IgIPv6:
		if !e.hasIPv6Builder() {
			e.IPv6Builder = new(netipx.IPSetBuilder)
		}
		e.IPv6Builder.AddPrefix(*prefix)
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (e *Entry) remove(prefix *netip.Prefix, ipType IPIgnoreType) error {
	switch ipType {
	case IgIPv4:
		if e.hasIPv4Builder() {
			e.IPv4Builder.RemovePrefix(*prefix)
		}
	case IgIPv6:
		if e.hasIPv6Builder() {
			e.IPv6Builder.RemovePrefix(*prefix)
		}
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (e *Entry) AddPrefix(cidr any) error {
	prefix, ipType, err := e.processPrefix(cidr)
	if err != nil && err != ErrCommentLine {
		return err
	}
	if err := e.add(prefix, ipType); err != nil {
		return err
	}
	return nil
}

func (e *Entry) RemovePrefix(cidr string) error {
	prefix, ipType, err := e.processPrefix(cidr)
	if err != nil && err != ErrCommentLine {
		return err
	}
	if err := e.remove(prefix, ipType); err != nil {
		return err
	}
	return nil
}

func (e *Entry) buildIPSet() error {
	if e.hasIPv4Builder() && !e.hasIPv4Set() {
		IPv4set, err := e.IPv4Builder.IPSet()
		if err != nil {
			return err
		}
		e.IPv4Set = IPv4set
	}

	if e.hasIPv6Builder() && !e.hasIPv6Set() {
		IPv6set, err := e.IPv6Builder.IPSet()
		if err != nil {
			return err
		}
		e.IPv6Set = IPv6set
	}

	return nil
}

func (e *Entry) MarshalPrefix(opts ...IgnoreIPOption) ([]netip.Prefix, error) {
	var ignoreIPType IPIgnoreType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}
	disableIPv4, disableIPv6 := false, false
	switch ignoreIPType {
	case IgIPv4:
		disableIPv4 = true
	case IgIPv6:
		disableIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	prefixes := make([]netip.Prefix, 0, 1024)

	if !disableIPv4 && e.hasIPv4Set() {
		prefixes = append(prefixes, e.IPv4Set.Prefixes()...)
	}

	if !disableIPv6 && e.hasIPv6Set() {
		prefixes = append(prefixes, e.IPv6Set.Prefixes()...)
	}

	if len(prefixes) > 0 {
		return prefixes, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}

func (e *Entry) MarshalIPRange(opts ...IgnoreIPOption) ([]netipx.IPRange, error) {
	var ignoreIPType IPIgnoreType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}
	disableIPv4, disableIPv6 := false, false
	switch ignoreIPType {
	case IgIPv4:
		disableIPv4 = true
	case IgIPv6:
		disableIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	ipranges := make([]netipx.IPRange, 0, 1024)

	if !disableIPv4 && e.hasIPv4Set() {
		ipranges = append(ipranges, e.IPv4Set.Ranges()...)
	}

	if !disableIPv6 && e.hasIPv6Set() {
		ipranges = append(ipranges, e.IPv6Set.Ranges()...)
	}

	if len(ipranges) > 0 {
		return ipranges, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}

func (e *Entry) MarshalText(opts ...IgnoreIPOption) ([]string, error) {
	var ignoreIPType IPIgnoreType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}
	disableIPv4, disableIPv6 := false, false
	switch ignoreIPType {
	case IgIPv4:
		disableIPv4 = true
	case IgIPv6:
		disableIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	cidrList := make([]string, 0, 1024)

	if !disableIPv4 && e.hasIPv4Set() {
		for _, prefix := range e.IPv4Set.Prefixes() {
			cidrList = append(cidrList, prefix.String())
		}
	}

	if !disableIPv6 && e.hasIPv6Set() {
		for _, prefix := range e.IPv6Set.Prefixes() {
			cidrList = append(cidrList, prefix.String())
		}
	}

	if len(cidrList) > 0 {
		return cidrList, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}
