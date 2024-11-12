## GeoView

Extract information from geoip and geosite files from Project X and Sing-box.

### Usage

```
Usage:
  -action string
        action: geoip | geosite (default "geoip")
  -input string
        datafile
  -ipv4
        enable ipv4 output (default true)
  -ipv6
        enable ipv6 output (default true)
  -list string
        comma separated site or geo list, e.g. "cn,jp" or "youtube,google"
  -output string
        output to file, leave empty to print to console
  -regex
        allow regex rules in the geosite result
```

### Examples

#### Extract IP ranges of China and Japan from geoip.dat

```bash
./geoview -action geoip -input geoip.dat -list cn,jp -output cn_jp.txt
```

### Extract domain list of gfw from geosite.dat

```bash
./geoview -action geosite -input geosite.dat -list gfw -output gfw.txt
```

Regex rules of geosite are ignored by default.