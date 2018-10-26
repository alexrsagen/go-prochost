package main

import (
	"io"
	"sync"
)

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

// AsyncMultiReader is a Reader that's the logical concatenation of
// the provided input readers. They're read sequentially. Once all
// inputs have returned EOF, Read will return EOF.
type AsyncMultiReader struct {
	readers      []io.Reader
	readersMutex sync.Mutex
	continueAt   int
}

// AddReaders adds readers to the AsyncMultiReader
func (t *AsyncMultiReader) AddReaders(readers ...io.Reader) {
	t.readersMutex.Lock()
	defer t.readersMutex.Unlock()

	t.readers = append(t.readers, readers...)
}

func (t *AsyncMultiReader) Read(p []byte) (int, error) {
	var err error
	var n, nn int
	var errReaders []int

	t.readersMutex.Lock()
	defer t.readersMutex.Unlock()

	for i := t.continueAt; i < len(t.readers); i++ {
		nn, err = t.readers[i].Read(p)
		n += nn
		if n == len(p) {
			t.continueAt = i
			return n, nil
		}

		if err != nil {
			errReaders = append(errReaders, i)
		}
	}
	for i := 0; i < t.continueAt; i++ {
		nn, err = t.readers[i].Read(p)
		n += nn
		if n == len(p) {
			t.continueAt = i
			return n, nil
		}
	}
	t.continueAt = 0

	for i := len(errReaders) - 1; i >= 0; i-- {
		t.readers = append(t.readers[:errReaders[i]], t.readers[errReaders[i]+1:]...)
	}
	if len(t.readers) == 0 {
		return n, io.EOF
	}

	return n, nil
}
