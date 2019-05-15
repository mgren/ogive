package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"github.com/awnumar/memguard"
	"github.com/minio/sio"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
)

// GetGCM returns a new AES GCM cipher with optional custom nonce size
func GetGCM(key *memguard.LockedBuffer, size int) (gcm cipher.AEAD, err error) {
	var c cipher.Block

	c, err = aes.NewCipher(key.Buffer())
	if err != nil {
		return
	}

	if size > 0 {
		return cipher.NewGCMWithNonceSize(c, size)
	}

	return cipher.NewGCM(c)
}

// GetCryptWriter returns a new io.WriteCloser that will decrypt data written to it.
// Plaintext will be saved under the specified directory with chosen filename.
//
// This writer has 2 bytes already written to it in order to initialize the underlying AES instance.
func GetCryptWriter(key *memguard.LockedBuffer, dir, fname string) (w io.WriteCloser, err error) {
	var f os.FileInfo
	f, err = os.Stat(dir)
	if err != nil {
		return
	}
	if !f.Mode().IsDir() {
		err = errors.New(dir + " is not a directory.")
		return
	}

	dstFname := filepath.Join(dir, fname)
	var dst *os.File

	dst, err = os.OpenFile(dstFname, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return
	}

	w, err = sio.DecryptWriter(dst, sio.Config{Key: key.Buffer()})

	var n int
	n, err = w.Write([]byte{sio.Version20, sio.AES_256_GCM})
	if n != 2 {
		err = errors.New("Invalid write length " + string(n))
	}

	return
}

// GetCryptReader returns a new io.Reader that reads and encrypts the specified filename
func GetCryptReader(key *memguard.LockedBuffer, fname string) (r io.Reader, s int, err error) {
	var src *os.File
	src, err = os.Open(fname)
	if err != nil {
		return
	}

	var f os.FileInfo
	f, err = src.Stat()
	if err != nil {
		return
	}

	s = int(f.Size())

	// Assume block device
	if s == 0 {
		// Ignoring the error is ok here, continue without size
		s, _ = unix.IoctlGetInt(int(src.Fd()), unix.BLKGETSIZE64)
	}

	r, err = sio.EncryptReader(src, sio.Config{
		MinVersion:   sio.Version20,
		MaxVersion:   sio.Version20,
		CipherSuites: []byte{sio.AES_256_GCM},
		Key:          key.Buffer(),
	})

	return
}
