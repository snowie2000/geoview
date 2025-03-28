package geosite

import (
	"fmt"
	"os"
	"strings"

	"github.com/snowie2000/geoview/protohelper"

	E "github.com/sagernet/sing/common/exceptions"
	"google.golang.org/protobuf/proto"
)

func LoadV2Site(geositeBytes []byte, wantList map[string][]string, exitOnError bool) ([]GeoSite, error) {
	var geosite GeoSite
	var geositeList []GeoSite
	for code := range wantList {
		found := protohelper.FindCode(geositeBytes, []byte(strings.ToUpper(code)))
		if found != nil {
			if err := proto.Unmarshal(found, &geosite); err != nil {
				return nil, err
			}
			geositeList = append(geositeList, geosite)
		} else if exitOnError {
			return nil, fmt.Errorf("%s not found", code)
		}
	}
	return geositeList, nil
}

func LoadV2SiteFromFile(filename string, wantList map[string][]string, exitOnError bool) ([]GeoSite, error) {
	geositeBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, E.Cause(err, "failed to load V2Ray GeoSite database")
	}
	return LoadV2Site(geositeBytes, wantList, exitOnError)
}
