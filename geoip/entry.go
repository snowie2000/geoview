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
	IgIPv4 IPIgnoreType = "IPv4"
	IgIPv6 IPIgnoreType = "IPv6"
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
	name          string
	IgIPv4Builder *netipx.IPSetBuilder
	IgIPv6Builder *netipx.IPSetBuilder
	IgIPv4Set     *netipx.IPSet
	IgIPv6Set     *netipx.IPSet
}

func NewEntry(name string) *Entry {
	return &Entry{
		name: strings.ToUpper(strings.TrimSpace(name)),
	}
}

func (e *Entry) GetName() string {
	return e.name
}

func (e *Entry) hasIgIPv4Builder() bool {
	return e.IgIPv4Builder != nil
}

func (e *Entry) hasIgIPv6Builder() bool {
	return e.IgIPv6Builder != nil
}

func (e *Entry) hasIgIPv4Set() bool {
	return e.IgIPv4Set != nil
}

func (e *Entry) hasIgIPv6Set() bool {
	return e.IgIPv6Set != nil
}

func (e *Entry) GetIgIPv4Set() (*netipx.IPSet, error) {
	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	if e.hasIgIPv4Set() {
		return e.IgIPv4Set, nil
	}

	return nil, fmt.Errorf("entry %s has no IgIPv4 set", e.GetName())
}

func (e *Entry) GetIgIPv6Set() (*netipx.IPSet, error) {
	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	if e.hasIgIPv6Set() {
		return e.IgIPv6Set, nil
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
		if !e.hasIgIPv4Builder() {
			e.IgIPv4Builder = new(netipx.IPSetBuilder)
		}
		e.IgIPv4Builder.AddPrefix(*prefix)
	case IgIPv6:
		if !e.hasIgIPv6Builder() {
			e.IgIPv6Builder = new(netipx.IPSetBuilder)
		}
		e.IgIPv6Builder.AddPrefix(*prefix)
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (e *Entry) remove(prefix *netip.Prefix, ipType IPIgnoreType) error {
	switch ipType {
	case IgIPv4:
		if e.hasIgIPv4Builder() {
			e.IgIPv4Builder.RemovePrefix(*prefix)
		}
	case IgIPv6:
		if e.hasIgIPv6Builder() {
			e.IgIPv6Builder.RemovePrefix(*prefix)
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
	if e.hasIgIPv4Builder() && !e.hasIgIPv4Set() {
		IgIPv4set, err := e.IgIPv4Builder.IPSet()
		if err != nil {
			return err
		}
		e.IgIPv4Set = IgIPv4set
	}

	if e.hasIgIPv6Builder() && !e.hasIgIPv6Set() {
		IgIPv6set, err := e.IgIPv6Builder.IPSet()
		if err != nil {
			return err
		}
		e.IgIPv6Set = IgIPv6set
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
	disableIgIPv4, disableIgIPv6 := false, false
	switch ignoreIPType {
	case IgIPv4:
		disableIgIPv4 = true
	case IgIPv6:
		disableIgIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	prefixes := make([]netip.Prefix, 0, 1024)

	if !disableIgIPv4 && e.hasIgIPv4Set() {
		prefixes = append(prefixes, e.IgIPv4Set.Prefixes()...)
	}

	if !disableIgIPv6 && e.hasIgIPv6Set() {
		prefixes = append(prefixes, e.IgIPv6Set.Prefixes()...)
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
	disableIgIPv4, disableIgIPv6 := false, false
	switch ignoreIPType {
	case IgIPv4:
		disableIgIPv4 = true
	case IgIPv6:
		disableIgIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	ipranges := make([]netipx.IPRange, 0, 1024)

	if !disableIgIPv4 && e.hasIgIPv4Set() {
		ipranges = append(ipranges, e.IgIPv4Set.Ranges()...)
	}

	if !disableIgIPv6 && e.hasIgIPv6Set() {
		ipranges = append(ipranges, e.IgIPv6Set.Ranges()...)
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
	disableIgIPv4, disableIgIPv6 := false, false
	switch ignoreIPType {
	case IgIPv4:
		disableIgIPv4 = true
	case IgIPv6:
		disableIgIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	cidrList := make([]string, 0, 1024)

	if !disableIgIPv4 && e.hasIgIPv4Set() {
		for _, prefix := range e.IgIPv4Set.Prefixes() {
			cidrList = append(cidrList, prefix.String())
		}
	}

	if !disableIgIPv6 && e.hasIgIPv6Set() {
		for _, prefix := range e.IgIPv6Set.Prefixes() {
			cidrList = append(cidrList, prefix.String())
		}
	}

	if len(cidrList) > 0 {
		return cidrList, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}
