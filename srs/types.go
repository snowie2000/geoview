package srs

import (
	"time"

	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/json"
)

type Listable[T any] []T

func (l Listable[T]) MarshalJSON() ([]byte, error) {
	arrayList := []T(l)
	if len(arrayList) == 1 {
		return json.Marshal(arrayList[0])
	}
	return json.Marshal(arrayList)
}

func (l *Listable[T]) UnmarshalJSON(content []byte) error {
	err := json.UnmarshalDisallowUnknownFields(content, (*[]T)(l))
	if err == nil {
		return nil
	}
	var singleItem T
	newError := json.UnmarshalDisallowUnknownFields(content, &singleItem)
	if newError != nil {
		return E.Errors(err, newError)
	}
	*l = []T{singleItem}
	return nil
}

type DNSQueryType uint16

func (t DNSQueryType) String() string {
	return F.ToString(uint16(t))
}

func (t DNSQueryType) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint16(t))
}

func (t *DNSQueryType) UnmarshalJSON(bytes []byte) error {
	var valueNumber uint16
	err := json.Unmarshal(bytes, &valueNumber)
	if err == nil {
		*t = DNSQueryType(valueNumber)
		return nil
	}
	var valueString string
	err = json.Unmarshal(bytes, &valueString)
	if err == nil {
		// queryType, loaded := mDNS.StringToType[valueString]
		// if loaded {
		// 	*t = DNSQueryType(queryType)
		// 	return nil
		// }
	}
	return E.New("unknown DNS query type: ", string(bytes))
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal((time.Duration)(d).String())
}

func (d *Duration) UnmarshalJSON(bytes []byte) error {
	var value string
	err := json.Unmarshal(bytes, &value)
	if err != nil {
		return err
	}
	duration, err := ParseDuration(value)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}
