package geoip

import (
	"errors"
	"fmt"
	"github.com/snowie2000/geoview/global"
	"io"
	"net"
	"net/netip"
	"os"
	"strconv"

	"github.com/snowie2000/geoview/protohelper"
	"github.com/snowie2000/geoview/srs"
	"go4.org/netipx"
	"google.golang.org/protobuf/proto"
)

type GeoIPDatIn struct {
	URI       string
	Want      map[string]bool
	MustExist bool
}

type IPType int

const (
	IPv4 IPType = 1
	IPv6 IPType = 2
)

func (g *GeoIPDatIn) ToGeoIP() (*GeoIPList, error) {
	reader, err := os.Open(g.URI)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	ipList := new(GeoIPList)
	reader.Seek(0, io.SeekStart)
	codeList := protohelper.CodeListByReader(reader)
	for _, code := range codeList {
		if _, ok := g.Want[code.Name]; ok {
			reader.Seek(code.Offset, io.SeekStart)
			var geoip GeoIP
			stripped := make([]byte, code.Size)
			io.ReadFull(reader, stripped)
			if stripped != nil {
				proto.Unmarshal(stripped, &geoip)

				if len(geoip.Cidr) > 0 {
					ipList.Entry = append(ipList.Entry, &geoip)
				}
			} else {
				// log.Println("code not found", code)
			}
			//runtime.GC()
		}
	}
	return ipList, nil
}

func (g *GeoIPDatIn) FindIP(ip string) (list []string) {
	nip, ok := netipx.FromStdIP(net.ParseIP(ip))
	if !ok {
		return
	}

	// read from url or file
	file, err := os.Open(g.URI)
	if err != nil {
		return
	}
	defer file.Close()

	file.Seek(0, io.SeekStart)
	codeList := protohelper.CodeListByReader(file) // get all available geoip codes
	// codeList := protohelper.CodeList(geoipBytes)
	for _, code := range codeList {
		file.Seek(code.Offset, io.SeekStart)
		var geoip GeoIP
		var prefix netip.Prefix
		stripped := make([]byte, code.Size)
		io.ReadFull(file, stripped)
		if stripped != nil {
			proto.Unmarshal(stripped, &geoip)
			for _, v2rayCIDR := range geoip.Cidr {
				vip := net.IP(v2rayCIDR.GetIp())
				if is4 := vip.To4() != nil; is4 == nip.Is4() {
					ipStr := vip.String() + "/" + fmt.Sprint(v2rayCIDR.GetPrefix())
					prefix, _ = netip.ParsePrefix(ipStr)
					if prefix.Contains(nip) {
						list = append(list, code.Name)
						break
					}
				}
			}
		} else {
			// log.Println("code not found", code)
		}
	}
	return
}

func (g *GeoIPDatIn) Extract(ipType IPType) (list []string, err error) {
	//log.Println("extracting", ipType)
	err, list = g.parseFile(g.URI, ipType)
	//log.Println("file read")

	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("no match countrycode found")
	}
	//log.Println("complete")
	return
}

func (g *GeoIPDatIn) ToRuleSet(ipType IPType) (*srs.PlainRuleSetCompat, error) {
	cidrlist, err := g.Extract(ipType)
	if err != nil {
		return nil, err
	}
	if len(cidrlist) == 0 {
		return nil, errors.New("empty ip set")
	}

	ruleset := &srs.PlainRuleSetCompat{
		Version: srs.RuleSetVersionCurrent,
	}
	rule := srs.HeadlessRule{
		Type: srs.RuleTypeDefault,
	}
	rule.DefaultOptions.IPCIDR = cidrlist
	ruleset.Options.Rules = []srs.HeadlessRule{rule}
	return ruleset, nil
}

func (g *GeoIPDatIn) ToQuantumultX(ipType IPType) ([]string, error) {
	// extract ip rules from the database
	var list4 []string
	var list6 []string
	err, list := g.parseFile(g.URI, IPv4)
	if err == nil {
		list4 = make([]string, len(list))
		// now convert ip-cidr into qx filter format
		for i, cidr := range list {
			list4[i] = fmt.Sprintf("ip-cidr, %s, Proxy", cidr)
		}
	}

	err, list = g.parseFile(g.URI, IPv6)
	if err == nil {
		list6 = make([]string, len(list))
		// now convert ip-cidr into qx filter format
		for i, cidr := range list {
			list6[i] = fmt.Sprintf("ip6-cidr, %s, Proxy", cidr)
		}
	}

	var ignoreIPType IPIgnoreType = ""
	if ipType&IPv4 == 0 {
		ignoreIPType = IgIPv4
	}
	if ipType&IPv6 == 0 {
		ignoreIPType = IgIPv6
	}
	// return request lists
	switch ignoreIPType {
	case IgIPv4:
		// only output ipv6 results
		return list6, nil
	case IgIPv6:
		// only output ipv4 results;
		return list4, nil
	default:
		// output both
		return append(list4, list6...), nil
	}
}

func (g *GeoIPDatIn) parseFile(path string, iptype IPType) (error, []string) {
	file, err := os.Open(path)
	if err != nil {
		return err, nil
	}
	defer file.Close()

	return g.generateEntries(file, iptype)
}

func (g *GeoIPDatIn) generateEntries(reader io.ReadSeeker, iptype IPType) (error, []string) {
	if global.Lowmem {
		return g.generateEntriesFromFile(reader, iptype)
	}

	geoipBytes, err := io.ReadAll(reader)
	if err != nil {
		return err, nil
	}
	allowIPv4 := iptype&IPv4 != 0
	allowIPv6 := iptype&IPv6 != 0
	var (
		ip   net.IP
		list []string = nil
	)
	for code := range g.Want {
		var geoip GeoIP
		stripped := protohelper.FindCode(geoipBytes, []byte(code))
		if stripped != nil {
			proto.Unmarshal(stripped, &geoip)

			for _, v2rayCIDR := range geoip.Cidr {
				ip = net.IP(v2rayCIDR.GetIp())
				if ip.To4() != nil && allowIPv4 {
					list = append(list, ip.String()+"/"+strconv.Itoa(int(v2rayCIDR.GetPrefix())))
				} else if allowIPv6 {
					list = append(list, ip.String()+"/"+strconv.Itoa(int(v2rayCIDR.GetPrefix())))
				}
			}
		} else if g.MustExist {
			return fmt.Errorf("%s doesn't exist", code), nil
		}
	}

	return nil, list
}

func (g *GeoIPDatIn) generateEntriesFromFile(reader io.ReadSeeker, iptype IPType) (error, []string) {
	reader.Seek(0, io.SeekStart)
	codeList := protohelper.CodeListByReader(reader)
	allowIPv4 := iptype&IPv4 != 0
	allowIPv6 := iptype&IPv6 != 0
	var (
		ip   net.IP
		list []string = nil
	)
	for _, code := range codeList {
		if _, ok := g.Want[code.Name]; ok {
			reader.Seek(code.Offset, io.SeekStart)
			var geoip GeoIP
			stripped := make([]byte, code.Size)
			io.ReadFull(reader, stripped)
			//log.Println("code read")
			if stripped != nil {
				if err := proto.Unmarshal(stripped, &geoip); err != nil {
					return err, nil
				}
				//log.Println("protobuf ready")

				for _, v2rayCIDR := range geoip.Cidr {
					ip = net.IP(v2rayCIDR.GetIp())
					if ip.To4() != nil && allowIPv4 {
						list = append(list, ip.String()+"/"+strconv.Itoa(int(v2rayCIDR.GetPrefix())))
					} else if allowIPv6 {
						list = append(list, ip.String()+"/"+strconv.Itoa(int(v2rayCIDR.GetPrefix())))
					}
				}
			} else if g.MustExist {
				return fmt.Errorf("%s doesn't exist", code), nil
			}
			//runtime.GC()
		}
	}
	return nil, list
}
