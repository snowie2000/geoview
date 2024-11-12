// main
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/snowie2000/geoview/geoip"
	"github.com/snowie2000/geoview/geosite"
)

func main() {
	input := flag.String("input", "", "datafile")
	action := flag.String("action", "geoip", "action: geoip | geosite")
	want := flag.String("list", "", "comma separated site or geo list, e.g. \"cn,jp\" or \"youtube,google\"")
	ipv4 := flag.Bool("ipv4", true, "enable ipv4 output")
	ipv6 := flag.Bool("ipv6", true, "enable ipv6 output")
	regex := flag.Bool("regex", false, "allow regex rules in the geosite result")
	output := flag.String("output", "", "output to file, leave empty to print to console")
	flag.Parse()

	if *input == "" {
		fmt.Printf("Error: Input file empty\nUsage:\n")
		flag.PrintDefaults()
		return
	}

	if *want == "" {
		fmt.Printf("Error: List should not be empty\nUsage:\n")
		flag.PrintDefaults()
		return
	}

	switch *action {
	case "geoip":
		list := strings.Split(*want, ",")
		wantMap := make(map[string]bool)
		for _, v := range list {
			wantMap[strings.ToUpper(strings.TrimSpace(v))] = true
		}
		data := &geoip.GeoIPDatIn{
			URI:  *input,
			Want: wantMap,
		}
		var tp geoip.IPType = 0
		if *ipv4 {
			tp |= geoip.IPv4
		}
		if *ipv6 {
			tp |= geoip.IPv6
		}
		ret, err := data.Extract(tp)
		if err == nil {
			if *output != "" { // output to file
				err = outputToFile(*output, ret)
				if err != nil {
					fmt.Println("Error:", err)
				}
				return
			}
			for _, v := range ret {
				fmt.Println(v)
			}
		} else {
			fmt.Println("Error:", err)
		}
		return

	case "geosite":
		list := strings.Split(*want, ",")
		for i, v := range list {
			list[i] = strings.TrimSpace(v)
		} // remove spaces
		ret, err := geosite.Extract(*input, list, *regex)
		if err == nil {
			if *output != "" { // output to file
				err = outputToFile(*output, ret)
				if err != nil {
					fmt.Println("Error:", err)
				}
				return
			}
			for _, v := range ret {
				fmt.Println(v)
			}
		} else {
			fmt.Println("Error:", err)
		}
	}
}

func outputToFile(file string, lines []string) error {
	return os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0o666)
}
