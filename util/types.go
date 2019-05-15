package util

import (
	"io"
)

// WriterAtFake is used to implement fake compliance with the io.WriteAt interface while also validating the first two bytes for sio
type WriterAtFake struct {
	// w is the underlying io.Writer that WriterAtFake proxies writes to.
	w io.Writer

	// f is a flag used to indicate whether the writer had received no writes (true) or had its first byte wtitten (false).
	f *bool
}
