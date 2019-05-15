package object

import (
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/awnumar/memguard"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mgren/ogive/crypt"
	"golang.org/x/crypto/argon2"
	"regexp"
	"strings"
)

// Parse translates the output of an s3 HeadObject command into a robust ogive archive file representation
// retrieving information such as original filename, unique file nonce, or the derived key (if possible).
//
// The reason this function requires an existing instance of cipher.AEAD is that it can be reused between
// multiple object instances (in case of list command), which is more efficient than creating it every time.
func Parse(res *s3.HeadObjectOutput, key *string, gcm cipher.AEAD, master *memguard.LockedBuffer) (o ResponseObject, err error) {
	if *res.ContentType != "application/x-ogive" {
		err = errors.New("Invalid content-type " + *res.ContentType)
		return
	}

	pattern := regexp.MustCompile("ongoing-request=\\\"(false|true)\\\"")

	o.Restore = "?????"

	if res.StorageClass != nil && *res.StorageClass == "DEEP_ARCHIVE" {
		o.Restore = "DEEPS"
	}

	if res.Restore != nil {
		match := pattern.FindStringSubmatch(*res.Restore)
		if match[1] == "true" {
			o.Restore = "RECOV"
		} else if match[1] == "false" {
			o.Restore = "READY"
		} else {
			o.Restore = "?????"
		}
	}

	o.Size = int(*res.ContentLength)
	o.LastModified = *res.LastModified

	if key == nil || gcm == nil {
		return
	}

	base := strings.Replace(strings.Replace(*key, ".", "/", -1), "-", "+", -1)
	var cryptName, name []byte

	cryptName, err = base64.RawStdEncoding.DecodeString(base)
	if err != nil {
		return
	}

	o.Nonce, err = hex.DecodeString(*res.Metadata["Nonce"])
	if err != nil {
		return
	}
	if len(o.Nonce) != 32 {
		err = errors.New("Malformed nonce " + *res.Metadata["Nonce"])
		return
	}

	name, err = gcm.Open(nil, o.Nonce, cryptName, nil)
	if err != nil {
		return
	}

	o.Name = string(name)

	if master == nil {
		return
	}

	o.Key, err = memguard.NewImmutableFromBytes(argon2.Key(master.Buffer(), o.Nonce, 3, 32*1024, 4, 32))

	return
}

// Prepare is the inverse of Parse. It generates a unique nonce and encrypts the filename.
func Prepare(master *memguard.LockedBuffer, fname string) (o RequestObject, err error) {
	var buf *memguard.LockedBuffer
	buf, err = memguard.NewImmutableRandom(32)
	if err != nil {
		return
	}

	// Assignment to o.Nonce must be done after this KDF (at least in GO 1.11.10). Otherwise, hilarity ensues:
	// If o.Nonce is used as salt parameter instead of buf.Buffer() the GC will prematurely decide to free
	// the memory region for memguard container which will then be immediately reused for o.Key...
	// o.Nonce and o.Key.Buffer() will refer to the same address and return identical data.
	o.Key, err = memguard.NewImmutableFromBytes(argon2.Key(master.Buffer(), buf.Buffer(), 3, 32*1024, 4, 32))
	if err != nil {
		return
	}

	o.Nonce = buf.Buffer()

	// Use bare AES for filename, to save on sio overhead
	// Override default GCM nonce size, since a single nonce is shared between file content and file name
	gcm, err := crypt.GetGCM(master, 32)
	master.Destroy()
	if err != nil {
		return
	}

	encryptedBase := gcm.Seal(nil, o.Nonce, []byte(fname), nil)

	o.Name = strings.Replace(strings.Replace(base64.RawStdEncoding.EncodeToString(encryptedBase), "/", ".", -1), "+", "-", -1)

	if len(o.Name) > 1024 {
		err = errors.New("Storage filename too long.")
	}

	return
}
