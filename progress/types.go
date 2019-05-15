package progress

import (
	"io"
)

// Reader extends io.Reader interface with a byte counter
type Reader struct {
	// Reader is the underlying io.Reader to which all read calls are proxied.
	io.Reader

	// totalProgress indicates the total number of bytes read from reader.
	totalProgress int
}

// Writer extends io.Writer interface with a byte counter
type Writer struct {
	// Writer is the underlying io.Writer to which all read calls are proxied.
	io.Writer

	// totalProgress indicates the total number of bytes written to writer.
	totalProgress int
}

// ProgressReporter is an interface implemented by progress-tracking readers and writers
type ProgressReporter interface {
	// GetProgress returns the totalProgress of the r/w interface
	GetProgress() int
}
