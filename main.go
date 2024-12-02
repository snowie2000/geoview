// main
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/snowie2000/geoview/geoip"
	"github.com/snowie2000/geoview/geosite"
	"github.com/snowie2000/geoview/srs"
)

var (
	input      string
	datatype   string
	action     string
	want       string
	ipv4       bool
	ipv6       bool
	regex      bool
	output     string
	target     string
	appendfile bool
)

func main() {
	flag.StringVar(&input, "input", "", "datafile")
	flag.StringVar(&datatype, "type", "geoip", "datafile type: geoip | geosite")
	flag.StringVar(&action, "action", "extract", "action: extract | convert | lookup")
	flag.StringVar(&want, "list", "", "comma separated site or geo list, e.g. \"cn,jp\" or \"youtube,google\"")
	flag.BoolVar(&ipv4, "ipv4", true, "enable ipv4 output")
	flag.BoolVar(&ipv6, "ipv6", true, "enable ipv6 output")
	flag.BoolVar(&regex, "regex", false, "allow regex rules in the geosite result")
	flag.StringVar(&output, "output", "", "output to file, leave empty to print to console")
	flag.StringVar(&target, "value", "", "ip or domain to lookup, required only for lookup action")
	flag.BoolVar(&appendfile, "append", false, "append to existing file instead of overwriting")
	flag.Parse()

	if input == "" {
		printErrorln("Error: Input file empty\nUsage:\n")
		flag.PrintDefaults()
		return
	}

	switch action {
	case "extract":
		if want == "" {
			printErrorln("Error: List should not be empty\nUsage:\n")
			flag.PrintDefaults()
			return
		}
		extract()
	case "convert":
		if want == "" {
			printErrorln("Error: List should not be empty\nUsage:\n")
			flag.PrintDefaults()
			return
		}
		convert()
	case "lookup":
		if target == "" {
			printErrorln("Error: Target should not be empty\nUsage:\n")
			flag.PrintDefaults()
			return
		}
		lookup()
	default:
		printErrorln("Error: unknown action:", action)
	}
}

func extract() {
	switch datatype {
	case "geoip":
		list := strings.Split(want, ",")
		wantMap := make(map[string]bool)
		for _, v := range list {
			wantMap[strings.ToUpper(strings.TrimSpace(v))] = true
		}
		data := &geoip.GeoIPDatIn{
			URI:  input,
			Want: wantMap,
		}
		var tp geoip.IPType = 0
		if ipv4 {
			tp |= geoip.IPv4
		}
		if ipv6 {
			tp |= geoip.IPv6
		}
		ret, err := data.Extract(tp)
		if err == nil {
			if output != "" { // output to file
				err = outputToFile(output, ret, appendfile)
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
		list := strings.Split(want, ",")
		for i, v := range list {
			list[i] = strings.TrimSpace(v)
		} // remove spaces
		ret, err := geosite.Extract(input, list, regex)
		if err == nil {
			if output != "" { // output to file
				err = outputToFile(output, ret, appendfile)
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

func convert() {
	switch datatype {
	case "geoip":
		list := strings.Split(want, ",")
		wantMap := make(map[string]bool)
		for _, v := range list {
			wantMap[strings.ToUpper(strings.TrimSpace(v))] = true
		}
		data := &geoip.GeoIPDatIn{
			URI:  input,
			Want: wantMap,
		}
		var tp geoip.IPType = 0
		if ipv4 {
			tp |= geoip.IPv4
		}
		if ipv6 {
			tp |= geoip.IPv6
		}
		ret, err := data.ToRuleSet(tp)
		if err == nil {
			if output != "" { // output to file
				err = outputRulesetToFile(output, ret)
				if err != nil {
					printErrorln("Error:", err)
				}
				return
			}
			// output json to stdout
			stdjson := json.NewEncoder(os.Stdout)
			if err = stdjson.Encode(*ret); err != nil {
				printErrorln("Error:", err, ret)
			}
		} else {
			printErrorln("Error:", err)
		}
		return

	case "geosite":
		list := strings.Split(want, ",")
		for i, v := range list {
			list[i] = strings.TrimSpace(v)
		} // remove spaces
		ret, err := geosite.ToRuleSet(input, list, regex)
		if err == nil {
			if output != "" { // output to file
				err = outputRulesetToFile(output, ret)
				if err != nil {
					printErrorln("Error:", err)
				}
				return
			}
			// output json to stdout
			stdjson := json.NewEncoder(os.Stdout)
			if err = stdjson.Encode(*ret); err != nil {
				printErrorln("Error:", err, ret)
			}
		} else {
			printErrorln("Error:", err)
		}
	}
}

func lookup() {
	switch datatype {
	case "geoip":
		data := &geoip.GeoIPDatIn{
			URI: input,
		}
		list := data.FindIP(target)
		for _, code := range list {
			fmt.Println(code)
		}
	case "geosite":
		printErrorln("Not implemented")
	}
}

func outputRulesetToFile(fileName string, ruleset *srs.PlainRuleSetCompat) error {
	if strings.EqualFold(filepath.Ext(fileName), ".json") {
		//output json
		if appendfile {
			// incremental json generation
			file, err := os.OpenFile(fileName, os.O_RDONLY, 0666)
			if err == nil {
				// read old rules
				var oldset srs.PlainRuleSetCompat
				decoder := json.NewDecoder(file)
				decoder.Decode(&oldset)
				file.Close()
				ruleset.Options.Rules = append(oldset.Options.Rules, ruleset.Options.Rules...) // append new rules to the old ones
			}
		}
		file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		return encoder.Encode(*ruleset) // generate new json
	} else {
		file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		defer file.Close()
		return srs.Write(file, ruleset.Options, ruleset.Version != 1)
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
