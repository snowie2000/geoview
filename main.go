// main
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/snowie2000/geoview/geoip"
	"github.com/snowie2000/geoview/geosite"
	"github.com/snowie2000/geoview/global"
	"github.com/snowie2000/geoview/memory"
	"github.com/snowie2000/geoview/protohelper"
	"github.com/snowie2000/geoview/srs"
	"google.golang.org/protobuf/proto"
	"io"
	"os"
	"strings"
)

var (
	version  bool
	strict   bool
	exitCode = 0
)

const (
	VERSION string = "0.2.1"
)

func main() {
	//t := time.Now()
	//defer func() {
	//	elapsed := time.Since(t)
	//	fmt.Println("finished in", elapsed)
	//}()

	memory.SetDynamicMemoryLimit(0.80)

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
	myflag.StringVar(&global.Format, "format", "ruleset", "convert output format. type: ruleset(srs) | quantumultx(qx) | json | geosite | geoip")
	myflag.BoolVar(&global.Appendfile, "append", false, "append to existing file instead of overwriting")
	myflag.BoolVar(&global.Lowmem, "lowmem", true, "low memory mode, reduce memory cost by partial file reading")
	myflag.BoolVar(&version, "version", false, "print version")
	myflag.BoolVar(&strict, "strict", true, "strict mode, non-existent code will result in an error")
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
			listCodes()
		} else {
			extract()
		}
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
	//elapsed := time.Since(t)
	//fmt.Println("finished in", elapsed)
	os.Exit(exitCode)
}

// list all stored codes in the database
func listCodes() {
	switch global.Datatype {
	case "geoip":
		file, err := os.Open(global.Input)
		if err != nil {
			printErrorln("Can't open input file")
			return
		}
		defer file.Close()
		list := protohelper.CodeListByReader(file)
		fmt.Println("Available codes:")
		for _, code := range list {
			fmt.Println(code.Name)
		}

	case "geosite":
		fileContent, err := os.ReadFile(global.Input)
		if err != nil {
			printErrorln("Can't open input file")
			return
		}
		// load as sing-box db
		if _, codes, err := geosite.LoadSingSite(fileContent); err == nil {
			fmt.Println("Available codes:")
			for _, code := range codes {
				fmt.Println(string(code))
			}
			return
		}
		// load as v2ray db
		codes := protohelper.CodeList(fileContent)
		fmt.Println("Available codes:")
		for _, code := range codes {
			fmt.Println(string(code))
		}
		return
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
			URI:       global.Input,
			Want:      wantMap,
			MustExist: strict,
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
		gsreader := geosite.NewGeositeHandler(global.Input, strict, global.Lowmem)
		ret, err := gsreader.Extract(wantMap, global.Regex)
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
			URI:       global.Input,
			Want:      wantMap,
			MustExist: strict,
		}
		var tp geoip.IPType = 0
		if global.Ipv4 {
			tp |= geoip.IPv4
		}
		if global.Ipv6 {
			tp |= geoip.IPv6
		}
		// convert to the target format according to the format arg
		switch global.Format {
		case "json": // ruleset json
			fallthrough
		case "srs":
			fallthrough
		case "ruleset": //ruleset binary
			ret, err := data.ToRuleSet(tp)
			if err == nil {
				if global.Output != "" { // output to file
					err = outputRulesetToFile(global.Output, ret, global.Format)
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
		case "geoip":
			if global.Output == "" {
				printErrorln("Error: Output file for geoip conversion is required")
				return
			}
			list := strings.Split(global.Want, ",")
			wantMap := make(map[string]bool)
			for _, v := range list {
				wantMap[strings.ToUpper(strings.TrimSpace(v))] = true
			}
			data := &geoip.GeoIPDatIn{
				URI:       global.Input,
				Want:      wantMap,
				MustExist: strict,
			}
			ret, err := data.ToGeoIP()
			if err == nil {
				protoBytes, err := proto.Marshal(ret)
				if err == nil {
					err = os.WriteFile(global.Output, protoBytes, 0644)
				}
				if err != nil {
					printErrorln("Error:", err)
				}
			} else {
				printErrorln("Error:", err)
			}
		case "qx":
			fallthrough
		case "quantumultx":
			ret, err := data.ToQuantumultX(tp)
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
		default:
			printErrorln("Error: converting from", global.Datatype, "to", global.Format, "is not supported")
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
		case "json": // ruleset json
			fallthrough
		case "srs":
			fallthrough
		case "ruleset": //ruleset binary
			gsreader := geosite.NewGeositeHandler(global.Input, strict, global.Lowmem)
			ret, err := gsreader.ToRuleSet(wantMap, global.Regex)
			if err == nil {
				if global.Output != "" { // output to file
					err = outputRulesetToFile(global.Output, ret, global.Format)
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
		case "geosite":
			if global.Output == "" {
				printErrorln("Error: Output file for geosite conversion is required")
				return
			}
			gsreader := geosite.NewGeositeHandler(global.Input, strict, global.Lowmem)
			ret, err := gsreader.ToGeosite(wantMap)
			if err == nil {
				protoBytes, err := proto.Marshal(ret)
				if err == nil {
					err = os.WriteFile(global.Output, protoBytes, 0644)
				}
				if err != nil {
					printErrorln("Error:", err)
				}
			} else {
				printErrorln("Error:", err)
			}
		case "qx":
			fallthrough
		case "quantumultx":
			gsreader := geosite.NewGeositeHandler(global.Input, strict, global.Lowmem)
			ret, err := gsreader.ToQuantumultX(wantMap)
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
		default:
			printErrorln("Error: converting from", global.Datatype, "to", global.Format, "is not supported")
		}
	}
}

func lookup() {
	switch global.Datatype {
	case "geoip":
		data := &geoip.GeoIPDatIn{
			URI:       global.Input,
			MustExist: strict,
		}
		list := data.FindIP(global.Target)
		for _, code := range list {
			fmt.Println(code)
		}
	case "geosite":
		gsreader := geosite.NewGeositeHandler(global.Input, strict, global.Lowmem)
		ret, err := gsreader.Lookup(global.Target)
		if err == nil {
			for _, code := range ret {
				fmt.Println(code)
			}
		} else {
			printErrorln("Error:", err)
		}
	}
}

func outputRulesetToFile(fileName string, ruleset *srs.PlainRuleSetCompat, format string) error {
	if strings.EqualFold(format, "json") {
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
	exitCode = 1
}

func printErrorf(str string, args ...any) {
	fmt.Fprintf(os.Stderr, str, args...)
	exitCode = 1
}
