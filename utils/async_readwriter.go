package utils

/*
import (
	"bytes"
	"errors"
	"io"
	"log"
)

var (
	ErrAsyncReadWriterClosed = errors.New("async readWriter closed")
)

// see https://go.dev/blog/pipelines for good pipelining

func MultiReaderAsync(readers ...io.Reader) io.Reader {
	// TO DO how to get back the error ???
	channel := make(chan []byte, 256)
	// errChan := make(chan error, 1)
	for _, reader := range readers {
		reader := reader
		go func() { ReadLinesInto(reader, channel) }()
	}
	return ReadFrom(channel)
}

// ReadLinesInto: read lines from the reader and push them into the channel
// If the reader reports any non-EOF error, it is returned.
// If the reader ends without a newline, io.ErrUnexpectedEOF is returned.
func ReadLinesInto(reader io.Reader, channel chan<- []byte) error {
	buffer := make([]byte, 1024)
	free_idx := 0
	for {
		// read from reader
		n, err := reader.Read(buffer[free_idx:])
		if err == io.EOF {
			// empty buffer
			if free_idx+n > 0 {
				channel <- buffer[:free_idx+n]
				if buffer[free_idx+n-1] == '\n' {
					return nil
				} else {
					return io.ErrUnexpectedEOF
				}
			}
			return nil
		} else if err != nil {
			return err
		}

		// find '\n'
		lo := 0
		for hi := 0; hi < n; hi++ {
			if buffer[hi] == '\n' {
				// yield line
				channel <- buffer[lo : hi+1]
				lo = hi + 1
			}
		}

		// copy remaining bytes to the beginning of the buffer
		if lo < n {
			copy(buffer, buffer[lo:n])
		}
		free_idx = n - lo
	}
}

// ReadFrom: empty the channel and provide a reader to read from it
// NB: the channel must be externally closed if this function is expected to return
func ReadFrom(channel <-chan []byte) io.Reader {
	buffer := &bytes.Buffer{}
	for section := range channel {
		buffer.Write(section)
	}

	var c chan error
	for {
		select {
		case err := <-c:
			log.Fatal(err)
			return nil
		case part, ok := <-channel:
			if !ok {
				return buffer
			}
			buffer.Write(part)
		}
	}
}

// AsyncReadWriter: a multi-producer, single-consumer ReadWriter
// Usage:
// 	rw := NewAsyncReadWriter()
// 	1 or more producer can call Write() on the readWriter
// 	1 consumer can call Read() on the readWriter
//  when all producers are done, exactly 1 call Close() on the readWriter
type AsyncReadWriter struct {
	channel chan []byte   // const
	done    chan struct{} // const
	buffer  []byte
}

// IsClosed: returns true if the readWriter is closed (thread-safe)
func (ar *AsyncReadWriter) IsClosed() bool {
	// nil check: an invalid readWriter is considered closed
	if ar == nil || ar.done == nil {
		return true
	}

	// check for closed channel
	select {
	case <-ar.done:
		return true
	default:
		return false
	}
}

// Write: push an item to the channel (thread-safe)
func (ar *AsyncReadWriter) Write(p []byte) (int, error) {
	if ar.IsClosed() {
		return 0, ErrAsyncReadWriterClosed
	}
	ar.channel <- p
	return len(p), nil
}

// Read: read from the channel until the channel is closed (NOT thread-safe)
func (ar *AsyncReadWriter) Read(p []byte) (int, error) {
	// nil check
	if ar == nil {
		return 0, io.EOF
	}

	// 1. empty buffer
	if len(ar.buffer) > 0 {
		n := copy(p, ar.buffer)
		ar.buffer = ar.buffer[n:]
		return n, nil
	}

	// 2. read from channel
	item, ok := <-ar.channel
	if !ok {
		return 0, io.EOF
	}

	// copy item to p and buffer
	n := copy(p, item)
	ar.buffer = item[n:]
	return n, nil
}

// mark the readWriter as closed: no more items will be pushed (NOT thread-safe)
// NB: closing a closed readWriter returns an error
func (ar *AsyncReadWriter) Close() error {
	if ar.IsClosed() {
		return ErrAsyncReadWriterClosed
	}
	close(ar.done)
	return nil
}

func NewAsyncReadWriter() *AsyncReadWriter {
	return NewAsyncReadWriterWithCapacity(16)
}

func NewAsyncReadWriterWithCapacity(capacity int) *AsyncReadWriter {
	return &AsyncReadWriter{
		channel: make(chan []byte, capacity),
		done:    make(chan struct{}),
		buffer:  nil,
	}
}

func AssertInterfaces() {
	var _ io.Reader = &AsyncReadWriter{}
	var _ io.Writer = &AsyncReadWriter{}
	var _ io.ReadWriter = &AsyncReadWriter{}

	var rw *AsyncReadWriter = nil
	rw.IsClosed()
}
*/
