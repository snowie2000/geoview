package geosite

import (
	"os"

	E "github.com/sagernet/sing/common/exceptions"
	"google.golang.org/protobuf/proto"
)

func LoadV2Site(geositeBytes []byte) ([]*GeoSite, error) {
	var geositeList GeoSiteList
	if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
		return nil, err
	}
	return geositeList.Entry, nil
}

func LoadV2SiteFromFile(filename string) ([]*GeoSite, error) {
	geositeBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, E.Cause(err, "failed to load V2Ray GeoSite database")
	}
	return LoadV2Site(geositeBytes)
}
