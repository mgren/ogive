package profile

import (
	"github.com/awnumar/memguard"
)

// InnerData is the actual profile data, stored in an encrypted format
type InnerData struct {
	// Key is the master key used to encrypt files at rest
	Key *memguard.LockedBuffer

	// AWSKeyId ia the AWS Key ID used to upload/download files
	AWSKeyId *memguard.LockedBuffer

	// AWSSecret is the AWS Secret associated with the AWS Key ID
	AWSSecret *memguard.LockedBuffer

	// BucketName is the S3 bucket name
	BucketName string

	// Endpoint is the S3 endpoint to connect to
	Endpoint string

	// Region is the AWS region in which the S3 bucket is located
	Region string
}

// OuterData is a wrapper for InnerData that holds information needed to perform
// encryption and decryption of the stored information
type OuterData struct {
	// Magic is a string indicator of this indeed being an ogive struct
	Magic string

	// Version ia the profile file version, in case the spec changes
	Version uint32

	// Salt is the salt value used for password derivation to unseal the InnerData
	Salt []byte

	// Inner is the encrypted, serialized representation of InnerData
	Inner []byte
}
