package main

import (
	"sync"
)

// LineBufferedWriter provides a line-buffered FIFO writer
// useful for buffering the output lines of a process
type LineBufferedWriter struct {
	buf      []string
	bufMutex sync.Mutex
	cur      int
}

// ReadLines returns a copy of the internal line buffer
func (r *LineBufferedWriter) ReadLines() []string {
	r.bufMutex.Lock()
	defer r.bufMutex.Unlock()

	var tmp []string
	tmp = make([]string, len(r.buf))
	copy(tmp, r.buf)

	return tmp
}

func (r *LineBufferedWriter) Write(p []byte) (int, error) {
	r.bufMutex.Lock()
	defer r.bufMutex.Unlock()
	var pos, npos, nnpos int
	var line []byte

	if len(r.buf) == 0 {
		return len(p), nil
	}

	for {
		if pos == len(p) {
			break
		}

		// If the last line is terminated, get an empty buffer line
		if len(r.buf[r.cur]) > 0 && r.buf[r.cur][len(r.buf[r.cur])-1] == '\n' {
			if r.cur == len(r.buf)-1 {
				r.buf = append(r.buf[1:], "")
			} else {
				r.cur++
			}
		}

		// Find next newline
		nnpos = 0
		npos = pos
		for ; pos < len(p); pos++ {
			if p[pos] == '\n' {
				pos++ // Move to after newline
				nnpos = pos
				break
			}
		}
		if nnpos == 0 {
			nnpos = len(p)
		}

		// Create line of size(previous contents) + size(new contents)
		line = make([]byte, len(r.buf[r.cur])+nnpos-npos)

		// Read previous contents into line
		copy(line, []byte(r.buf[r.cur]))

		// Read range npos to nnpos from p into line after previous contents
		copy(line[len(r.buf[r.cur]):], p[npos:nnpos])

		r.buf[r.cur] = string(line)
	}

	return len(p), nil
}
