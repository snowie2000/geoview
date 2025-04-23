package geosite

import (
	"bytes"
	"fmt"
	"github.com/snowie2000/geoview/protohelper"
	"io"
	"os"

	"google.golang.org/protobuf/proto"
)

type V2Site struct {
	codeList map[string]protohelper.CodeIndex
	reader   io.ReadSeekCloser
}

func (v *V2Site) Close() error {
	return v.reader.Close()
}

func (v *V2Site) Codes() (list []string) {
	for key := range v.codeList {
		list = append(list, key)
	}
	return list
}

func (v *V2Site) ReadSites(codes []string, exitOnError bool) ([]GeoSite, error) {
	var geosite GeoSite
	var geositeList []GeoSite

	for _, code := range codes {
		if index, ok := v.codeList[code]; !ok && exitOnError {
			return nil, fmt.Errorf("%s doesn't exist", code)
		} else {
			v.reader.Seek(index.Offset, io.SeekStart)
			buffer := make([]byte, index.Size)
			if _, err := io.ReadFull(v.reader, buffer); err != nil {
				return nil, err
			}
			if err := proto.Unmarshal(buffer, &geosite); err != nil {
				return nil, err
			}
			geositeList = append(geositeList, geosite)
		}
	}
	return geositeList, nil
}

func LoadV2Site(geositeBytes []byte) (*V2Site, error) {
	reader := bytes.NewReader(geositeBytes)
	list := protohelper.CodeListByReader(reader)
	reader.Seek(0, io.SeekStart)
	return &V2Site{
		codeList: list,
		reader:   &protohelper.NopReadSeekCloser{reader},
	}, nil
}

func LoadV2SiteFromFile(filename string) (*V2Site, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	list := protohelper.CodeListByReader(reader)
	reader.Seek(0, io.SeekStart)
	return &V2Site{
		codeList: list,
		reader:   reader,
	}, nil
}
