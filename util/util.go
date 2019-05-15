package util

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mgren/ogive/profile"
	"github.com/minio/sio"
	"io"
	"os"
	"path/filepath"
)

// GetDefaultProfileLoc returns the default ogive profile location
func GetDefaultProfileLoc() string {
	return filepath.Join(os.Getenv("HOME"), ".ogive")
}

// SizeIEC transforms a bytesize into an approximate (1 decimal place) IEC-compliant representation.
// It supports sizes up to 1000 TiB which is more than plenty for S3 maximum of 5 TB
func SizeIEC(b int64) string {
	unit, div, exp := int64(1024), int64(1024), 0

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGT"[exp])
}

// WriteAt is a dummy positional writer method. It ignores the offset and writes into the original Writer sequentially.
// This implementation is ogive-specific and omits first two bytes, making sure they are 0x20 0x00.
// See ogive/cmd/get.go source code for an explanation.
func (w WriterAtFake) WriteAt(p []byte, offset int64) (s int, err error) {
	if offset == 0 && *w.f {
		if p[0] != byte(sio.Version20) || p[1] != byte(sio.AES_256_GCM) {
			return 2, fmt.Errorf("Wrong start of header %x %x", p[0], p[1])
		}
		*w.f = false
		s, err = w.w.Write(p[2:])
		s += 2
		return
	}
	return w.w.Write(p)
}

// NewWriterAtFake wraps an io.Writer into a dummy io.WriterAt interface.
func NewWriterAtFake(w io.Writer) WriterAtFake {
	f := true
	return WriterAtFake{w, &f}
}

// GetSession uses ogive profile data to create a new AWS session.
func GetSession(i *profile.InnerData) *session.Session {
	defer i.AWSKeyId.Destroy()
	defer i.AWSSecret.Destroy()
	return session.New(&aws.Config{
		Region:      &i.Region,
		Credentials: credentials.NewStaticCredentials(string(i.AWSKeyId.Buffer()), string(i.AWSSecret.Buffer()), ""),
		Endpoint:    &i.Endpoint,
	})
}

// GetPartSize returns the part size for multipart upload.
// For files less than 500 MiB the part size is 5 MiB
// For files between 500 MiB and 5 000 MiB part size grows dynamically to create a 100-part upload
// For files between 5 000 MiB and 50 000 MiB the part size is 50 MiB and part count increases
// For files more than 50 000 MiB part count is 10 000 and part size starts to grow again
func GetPartSize(size int64) int64 {
	partSize := size / 100
	def := int64(50 << 20)

	if partSize < s3manager.MinUploadPartSize {
		return s3manager.MinUploadPartSize
	}

	if partSize > def {
		partSize = size / s3manager.MaxUploadParts
		if partSize < def {
			return def
		}

		return partSize
	}

	return partSize
}

// Fail prints out the error, additional message and exits the program with code 1 while also zeroing all memguard buffers.
func Fail(err error, msg string) {
	if awserr, ok := err.(awserr.Error); ok {
		fmt.Fprintln(os.Stderr, awserr)
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Fprintln(os.Stderr, msg+" Exiting...")
	memguard.SafeExit(1)
}
