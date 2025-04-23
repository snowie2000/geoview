// helper
package protohelper

import (
	"io"
	"log"
)

type CodeIndex struct {
	Name   string
	Offset int64
	Size   int64
}

func FindCode(data, code []byte) []byte {
	codeL := len(code)
	if codeL == 0 {
		return nil
	}
	for {
		dataL := len(data)
		if dataL < 2 {
			return nil
		}
		x, y := decodeVarint(data[1:])
		if x == 0 && y == 0 {
			return nil
		}
		headL, bodyL := 1+y, int(x)
		dataL -= headL
		if dataL < bodyL {
			return nil
		}
		data = data[headL:]
		if int(data[1]) == codeL {
			for i := 0; i < codeL && data[2+i] == code[i]; i++ {
				if i+1 == codeL {
					return data[:bodyL]
				}
			}
		}
		if dataL == bodyL {
			return nil
		}
		// log.Println(bodyL, data[1])
		data = data[bodyL:]
	}
}

func FindCodeByReader(data io.ReadSeeker, code []byte) []byte {
	codeL := len(code)
	if codeL == 0 {
		return nil
	}
	var (
		header []byte = make([]byte, 30)
		count  int
		err    error
	)
	for {
		if count, err = io.ReadAtLeast(data, header, 2); err != nil {
			return nil
		}
		x, y := decodeVarint(header[1:])
		if x == 0 && y == 0 {
			return nil
		}
		headL, bodyL := 1+y, int(x)
		data.Seek(int64(headL-count), io.SeekCurrent)
		if _, err = io.ReadFull(data, header[:2]); err != nil {
			return nil
		}
		size := header[1]
		if int(size) == codeL {
			c := make([]byte, size)
			_, err = io.ReadFull(data, c)
			for i := 0; i < codeL && c[i] == code[i]; i++ {
				if i+1 == codeL {
					body := make([]byte, bodyL)
					data.Seek(-int64(size)-2, io.SeekCurrent)
					if _, err = io.ReadFull(data, body); err == nil {
						return body
					} else {
						return nil
					}
				}
			}
		} else {
			size = 0 // do not include size in the seek
		}
		data.Seek(int64(bodyL-2-int(size)), io.SeekCurrent)
	}
}

func CodeListByReader(data io.ReadSeeker) (list map[string]CodeIndex) {
	var (
		header  []byte = make([]byte, 30)
		count   int
		err     error
		tracked = NewReadSeeker(data)
	)
	list = make(map[string]CodeIndex)

	tracked.Seek(0, io.SeekStart)
	for {
		if count, err = io.ReadAtLeast(tracked, header, 2); err != nil {
			return
		}
		x, y := decodeVarint(header[1:])
		if x == 0 && y == 0 {
			return
		}
		headL, bodyL := 1+y, int(x)
		tracked.Seek(int64(headL-count), io.SeekCurrent)
		if _, err = io.ReadFull(tracked, header[:2]); err != nil {
			return
		}
		size := header[1]
		if size > 0 {
			code := make([]byte, size)
			_, err = io.ReadFull(tracked, code)
			list[string(code)] = CodeIndex{
				Name:   string(code),
				Size:   int64(bodyL),
				Offset: tracked.Offset() - int64(size) - 2,
			}
		}
		// log.Println(bodyL, size)
		tracked.Seek(int64(bodyL-2-int(size)), io.SeekCurrent)
	}
	return
}

func CodeList(data []byte) (list [][]byte) {
	for {
		dataL := len(data)
		if dataL < 2 {
			return
		}
		x, y := decodeVarint(data[1:])
		if x == 0 && y == 0 {
			return
		}
		headL, bodyL := 1+y, int(x)
		dataL -= headL
		if dataL < bodyL {
			return
		}
		data = data[headL:]
		size := data[1]
		if size > 0 {
			list = append(list, data[2:size+2])
		}
		if dataL == bodyL {
			return
		}
		data = data[bodyL:]
	}
	return
}

func decodeVarint(buf []byte) (x uint64, n int) {
	for shift := uint(0); shift < 64; shift += 7 {
		if n >= len(buf) {
			log.Println(n, len(buf), "out of length")
			return 0, 0
		}
		b := uint64(buf[n])
		n++
		x |= (b & 0x7F) << shift
		if (b & 0x80) == 0 {
			return x, n
		}
	}

	// The number is too large to represent in a 64-bit value.
	return 0, 0
}
