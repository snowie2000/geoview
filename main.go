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
	appendfile := flag.Bool("append", false, "append to existing file instead of overwriting")
	flag.Parse()

	if *input == "" {
		printErrorln("Error: Input file empty\nUsage:\n")
		flag.PrintDefaults()
		return
	}

	if *want == "" {
		printErrorln("Error: List should not be empty\nUsage:\n")
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
				err = outputToFile(*output, ret, *appendfile)
				if err != nil {
					printErrorln("Error:", err)
				}
				return
			}
			for _, v := range ret {
				fmt.Println(v)
			}
		} else {
			printErrorln("Error:", err)
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
				err = outputToFile(*output, ret, *appendfile)
				if err != nil {
					printErrorln("Error:", err)
				}
				return
			}
			for _, v := range ret {
				fmt.Println(v)
			}
		} else {
			printErrorln("Error:", err)
		}
	}
}

func outputToFile(fileName string, lines []string, appendfile bool) error {
	var (
		file *os.File
		err  error
	)
	if appendfile {
		file, err = os.OpenFile(fileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	} else {
		file, err = os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	}
	if err != nil {
		return err
	}
	defer file.Close()
	needsNewline := false

	if appendfile {
		fileInfo, err := os.Stat(fileName)
		if err != nil {
			return err
		}

		if fileInfo.Size() > 0 {
			// Read the last byte to check if it's a newline
			buf := make([]byte, 1)
			_, err := file.ReadAt(buf, fileInfo.Size()-1)
			if err != nil {
				return err
			}
			if buf[0] != '\n' {
				needsNewline = true
			}
		}
	}

	// If needed, write a newline before appending
	if needsNewline {
		if _, err = file.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = file.WriteString(strings.Join(lines, "\n"))
	return err
}

func printErrorln(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
}

func printErrorf(str string, args ...any) {
	fmt.Fprintf(os.Stderr, str, args...)
}
