package protohelper

import "io"

type ReadSeekTracker struct {
	io.ReadSeeker
	pos int64
}

func NewReadSeeker(r io.ReadSeeker) *ReadSeekTracker {
	return &ReadSeekTracker{r, 0}
}

func (r *ReadSeekTracker) Read(p []byte) (n int, err error) {
	n, err = r.ReadSeeker.Read(p)
	r.pos += int64(n)
	return n, err
}

func (r *ReadSeekTracker) Seek(offset int64, whence int) (int64, error) {
	n, err := r.ReadSeeker.Seek(offset, whence)
	if err == nil {
		switch whence {
		case io.SeekStart:
			r.pos = offset
		case io.SeekCurrent:
			r.pos += offset
		case io.SeekEnd:
			r.pos = -999999 // can not track seekend
		}
	}
	return n, err
}

func (r *ReadSeekTracker) Offset() int64 {
	return r.pos
}

type NopReadSeekCloser struct {
	io.ReadSeeker
}

func (r *NopReadSeekCloser) Close() error { return nil }
