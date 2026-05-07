package autohand

import (
	"bufio"
	"io"
	"sync"
)

// LineReader reads newline-delimited lines from a stream.
type LineReader struct {
	reader  *bufio.Reader
	lines   chan string
	done    chan struct{}
	closeOnce sync.Once
}

// NewLineReader creates a LineReader from a reader.
func NewLineReader(r io.Reader) *LineReader {
	lr := &LineReader{
		reader: bufio.NewReader(r),
		lines:  make(chan string, 256),
		done:   make(chan struct{}),
	}
	go lr.readLoop()
	return lr
}

func (lr *LineReader) readLoop() {
	defer close(lr.lines)
	for {
		line, err := lr.reader.ReadString('\n')
		if line != "" {
			// Remove trailing newline only if present
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			select {
			case lr.lines <- line:
			case <-lr.done:
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// ReadLine returns the next line, or an error if closed.
func (lr *LineReader) ReadLine() (string, error) {
	select {
	case line, ok := <-lr.lines:
		if !ok {
			return "", io.EOF
		}
		return line, nil
	case <-lr.done:
		return "", io.EOF
	}
}

// Lines returns a channel of lines.
func (lr *LineReader) Lines() <-chan string {
	return lr.lines
}

// Close stops the line reader.
func (lr *LineReader) Close() {
	lr.closeOnce.Do(func() {
		close(lr.done)
	})
}
