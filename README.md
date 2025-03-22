## GeoView

Extract information from geoip and geosite files from Project X and Sing-box.

## Usage

```
Usage of geoview:
  -action string
        action: extract | convert | lookup (default "extract")
  -append
        append to existing file instead of overwriting
  -format string
        convert output format. type: ruleset(srs) | quantumultx(qx) | json | geosite | geoip (default "ruleset")
  -input string
        datafile
  -ipv4
        enable ipv4 output (default true)
  -ipv6
        enable ipv6 output (default true)
  -list string
        comma separated site or geo list, e.g. "cn,jp" or "youtube,google"
  -lowmem
        low memory mode, reduce memory cost by partial file reading
  -output string
        output to file, leave empty to print to console
  -regex
        allow regex rules in the geosite result
  -type string
        datafile type: geoip | geosite (default "geoip")
  -value string
        ip or domain to lookup, required only for lookup action
  -version
        print version
```

## Examples

#### Extract IP ranges of China and Japan from geoip.dat

```bash
./geoview -type geoip -input geoip.dat -list cn,jp -output cn_jp.txt
```

#### Extract domain list of gfw from geosite.dat

```bash
./geoview -type geosite -input geosite.dat -list gfw -output gfw.txt
```

-------

## Lookup IPs and Domains

The `-action lookup` flag will search for your target ip or domain in the geoip or geosite file and output all the list codes that contain the desired IP or domain, including all possible domain attributes (no support for sing-box geosite)

#### Lookup an IP address
```
./geoview.exe -input geoip.dat -type geoip -action lookup -value 1.1.1.1
AU
CLOUDFLARE
```

#### Lookup a domain
```
./geoview.exe -input geosite.dat -type geosite -action lookup -value samsung
TLD-!CN
PRIVATE
CATEGORY-COMPANIES
SAMSUNG
GEOLOCATION-!CN
```

```
./geoview.exe -input geosite.dat -type geosite -action lookup -value xp.apple.com
APPLE
APPLE@cn
APPLE-CN
CATEGORY-COMPANIES
CATEGORY-COMPANIES@cn
GEOLOCATION-!CN
APPLE-UPDATE
```

## Convert into other formats
The following conversions are supported 
- srs ruleset for singbox (*default)
- filter for QuantumultX
- converting from geosite to a subset of geosite
- converting from geoip to a subset of geoip

Format can be set by `-format` flag. abbr. is also accepted, such as `qx` for `quantumultx`

#### Extract domain list of medium and convert into sing-box ruleset JSON

```bash
./geoview -type geosite -action convert -input geosite.dat -list medium -output medium.json
```

#### Extract domain list of medium and convert into sing-box ruleset binary
```bash
./geoview -type geosite -action convert -input geosite.dat -list medium -output medium.srs
```

#### Extract domain list of medium and convert into QuantumultX filter set
```bash
./geoview -type geosite -action convert -input geosite.dat -list medium -output medium.conf -format qx
```

#### Extract domain list of medium and convert into a new `Geosite.dat` to remove memory consumption
```bash
./geoview -type geosite -action convert -input geosite.dat -list medium -output medium.dat -format geosite
```

#### Extract IPs of China and Japan and convert into a new `Geoip.dat` to remove memory consumption
```bash
./geoview -type geoip -action convert -input geoip.dat -list CN,JP -output cnjp.dat -format geoip
```

* Regex rules of geosite are ignored by default.

* When using `-append=true` to ruleset and the output format is JSON, existing rules will be kept and new rules will be appended.

* When converting geo files to ruleset, the output format is determined by `-format` flag. The format is always `JSON` if `output` is not specified for ruleset conversion.

* Binary ruleset conversion doesn't support appending, it always creates a new file.

## Low memory mode
By adding `-lowmem` to the command, the program will read the file partially to reduce memory usage. This is useful when execute on devices with limited memory.

## Compile for OpenWrt

Download the latest Openwrt source and clone this repository to the package directory.

```bash
git clone https://github.com/snowie2000/geoview.git package/geoview
```

Follow the steps below to compile the package.

```bash
./scripts/feeds update -a
./scripts/feeds install -a

make menuconfig

Network  ---> Web Servers/Proxies  ---> <*> geoview

make package/geoview/{clean,compile} V=s
```

To compile the golang projects under `Linux/arm64`, you need to manually install golang and set external golang environment in the `make menuconfig` menu.

-----

## Credit

Great thanks to project [geoip](https://github.com/Loyalsoldier/geoip) and [geo](https://github.com/MetaCubeX/geo)