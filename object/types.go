package object

import (
	"github.com/awnumar/memguard"
	"time"
)

// ResponseObject is an ogive-friendly representation of a HEAD result on a stored file
type ResponseObject struct {
	// Restore indicates object restore status
	Restore string

	// Size is the objects size as indicated by Content-Length
	Size int

	// LastModified is the file creation date as indicated by Last-Modified
	LastModified time.Time

	// Nonce is the unique nonce used for key derivation
	Nonce []byte

	// Name is the original unencrypted filename
	Name string

	// Key is the unique derived key
	Key *memguard.LockedBuffer
}

// RequestObject is an ogive-friendly representation of object metadata needed to prepare a PUT request
type RequestObject struct {
	// Nonce is the unique nonce used for key derivation
	Nonce []byte

	// Name is the encrypted filename, represented as AWS-key-safe version of base64
	// (no padding =, / replaced with . and + replaced with -)
	// see Characters That Might Require Special Handling
	// https://docs.aws.amazon.com/AmazonS3/latest/dev/UsingMetadata.html
	Name string

	// Key is the unique derived key
	Key *memguard.LockedBuffer
}
