// main
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/snowie2000/geoview/geoip"
	"github.com/snowie2000/geoview/geosite"
	"github.com/snowie2000/geoview/global"
	"github.com/snowie2000/geoview/srs"
)

var (
	version bool
)

const (
	VERSION string = "0.1.1"
)

func main() {
	myflag := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	myflag.StringVar(&global.Input, "input", "", "datafile")
	myflag.StringVar(&global.Datatype, "type", "geoip", "datafile type: geoip | geosite")
	myflag.StringVar(&global.Action, "action", "extract", "action: extract | convert | lookup")
	myflag.StringVar(&global.Want, "list", "", "comma separated site or geo list, e.g. \"cn,jp\" or \"youtube,google\"")
	myflag.BoolVar(&global.Ipv4, "ipv4", true, "enable ipv4 output")
	myflag.BoolVar(&global.Ipv6, "ipv6", true, "enable ipv6 output")
	myflag.BoolVar(&global.Regex, "regex", false, "allow regex rules in the geosite result")
	myflag.StringVar(&global.Output, "output", "", "output to file, leave empty to print to console")
	myflag.StringVar(&global.Target, "value", "", "ip or domain to lookup, required only for lookup action")
	myflag.StringVar(&global.Format, "format", "ruleset", "convert output format. type: ruleset(srs) | quantumultx(qx)")
	myflag.BoolVar(&global.Appendfile, "append", false, "append to existing file instead of overwriting")
	myflag.BoolVar(&global.Lowmem, "lowmem", false, "low memory mode, reduce memory cost by partial file reading")
	myflag.BoolVar(&version, "version", false, "print version")
	myflag.SetOutput(io.Discard)
	myflag.Parse(os.Args[1:])
	myflag.SetOutput(nil)

	if version {
		fmt.Printf("Geoview %s\n", VERSION)
		return
	}

	if global.Input == "" {
		printErrorln("Error: Input file empty\n")
		myflag.Usage()
		return
	}

	switch global.Action {
	case "extract":
		if global.Want == "" {
			printErrorln("Error: List should not be empty\n")
			myflag.Usage()
			return
		}
		extract()
	case "convert":
		if global.Want == "" {
			printErrorln("Error: List should not be empty\n")
			myflag.Usage()
			return
		}
		convert()
	case "lookup":
		if global.Target == "" {
			printErrorln("Error: Target should not be empty\n")
			myflag.Usage()
			return
		}
		lookup()
	default:
		printErrorln("Error: unknown action:", global.Action)
	}
}

func extract() {
	switch global.Datatype {
	case "geoip":
		list := strings.Split(global.Want, ",")
		wantMap := make(map[string]bool)
		for _, v := range list {
			wantMap[strings.ToUpper(strings.TrimSpace(v))] = true
		}
		data := &geoip.GeoIPDatIn{
			URI:  global.Input,
			Want: wantMap,
		}
		var tp geoip.IPType = 0
		if global.Ipv4 {
			tp |= geoip.IPv4
		}
		if global.Ipv6 {
			tp |= geoip.IPv6
		}
		ret, err := data.Extract(tp)
		if err == nil {
			if global.Output != "" { // output to file
				err = outputToFile(global.Output, ret, global.Appendfile)
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
		list := strings.Split(global.Want, ",")
		wantMap := make(map[string][]string)
		for _, v := range list {
			parts := strings.Split(strings.ToLower(v), "@") // attributes are lowercased
			wantMap[strings.ToUpper(parts[0])] = parts[1:]
		}
		ret, err := geosite.Extract(global.Input, wantMap, global.Regex)
		if err == nil {
			if global.Output != "" { // output to file
				err = outputToFile(global.Output, ret, global.Appendfile)
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
	switch global.Datatype {
	case "geoip":
		list := strings.Split(global.Want, ",")
		wantMap := make(map[string]bool)
		for _, v := range list {
			wantMap[strings.ToUpper(strings.TrimSpace(v))] = true
		}
		data := &geoip.GeoIPDatIn{
			URI:  global.Input,
			Want: wantMap,
		}
		var tp geoip.IPType = 0
		if global.Ipv4 {
			tp |= geoip.IPv4
		}
		if global.Ipv6 {
			tp |= geoip.IPv6
		}
		ret, err := data.ToRuleSet(tp)
		if err == nil {
			if global.Output != "" { // output to file
				err = outputRulesetToFile(global.Output, ret)
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
		list := strings.Split(global.Want, ",")
		wantMap := make(map[string][]string)
		for _, v := range list {
			parts := strings.Split(strings.ToLower(v), "@") // attributes are lowercased
			wantMap[strings.ToUpper(parts[0])] = parts[1:]
		}
		// convert to the target format according to the format arg
		switch global.Format {
		case "srs":
			fallthrough
		case "ruleset":
			ret, err := geosite.ToRuleSet(global.Input, wantMap, global.Regex)
			if err == nil {
				if global.Output != "" { // output to file
					err = outputRulesetToFile(global.Output, ret)
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

		case "qx":
			fallthrough
		case "quantumultx":
			ret, err := geosite.ToQuantumultX(global.Input, wantMap)
			if err == nil {
				if global.Output != "" {
					outputToFile(global.Output, ret, global.Appendfile)
				} else {
					for _, v := range ret {
						fmt.Println(v)
					}
				}
			} else {
				printErrorln("Error:", err)
			}
		}
	}
}

func lookup() {
	switch global.Datatype {
	case "geoip":
		data := &geoip.GeoIPDatIn{
			URI: global.Input,
		}
		list := data.FindIP(global.Target)
		for _, code := range list {
			fmt.Println(code)
		}
	case "geosite":
		ret, err := geosite.Lookup(global.Input, global.Target)
		if err == nil {
			for _, code := range ret {
				fmt.Println(code)
			}
		} else {
			printErrorln("Error:", err)
		}
	}
}

func outputRulesetToFile(fileName string, ruleset *srs.PlainRuleSetCompat) error {
	if strings.EqualFold(filepath.Ext(fileName), ".json") {
		//output json
		if global.Appendfile {
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
		return srs.Write(file, ruleset.Options, ruleset.Version)
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
