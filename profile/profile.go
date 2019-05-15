package profile

import (
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/awnumar/memguard"
	"github.com/mgren/ogive/input"
	"golang.org/x/crypto/argon2"
	"io/ioutil"
)

const version = uint32(1)
const magic = "OGPROF"

// Open reads the profile file from provided location and returns decrypted InnerData.
func Open(fname string) (in *InnerData, err error) {
	var pwd, derived *memguard.LockedBuffer
	var od *OuterData

	pwd, err = input.GetMaskedInput("Enter password", "", "", 64, 8)
	if err != nil {
		return
	}
	defer pwd.Destroy()

	var data []byte
	data, err = ioutil.ReadFile(fname)
	if err != nil {
		return
	}

	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err = dec.Decode(&od)
	if err != nil {
		return
	}

	if od.Magic != magic || od.Version != version {
		err = errors.New("Unsupported or corrupted profile file.")
		return
	}

	derived, err = memguard.NewImmutableFromBytes(argon2.Key(pwd.Buffer(), od.Salt, 3, 32*1024, 4, 32))
	if err != nil {
		return
	}
	defer derived.Destroy()

	return od.unpack(derived)
}

// Save takes InnerData, encrypts it with the provided key and saves under the selected filename.
func Save(key *memguard.LockedBuffer, in *InnerData, fname string) (err error) {
	var salt, derived *memguard.LockedBuffer
	salt, err = memguard.NewImmutableRandom(32)
	if err != nil {
		return
	}

	defer salt.Destroy()
	derived, err = memguard.NewImmutableFromBytes(argon2.Key(key.Buffer(), salt.Buffer(), 3, 32*1024, 4, 32))
	if err != nil {
		return
	}

	od := OuterData{Magic: magic, Version: version, Salt: salt.Buffer()}

	err = od.pack(in, derived)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err = enc.Encode(&od)
	if err != nil {
		return
	}

	return ioutil.WriteFile(fname, buf.Bytes(), 0600)
}

// NewInner creates a mew InnerData instance with a randomly generated master key.
func NewInner() (in *InnerData, err error) {
	var i InnerData
	in = &i
	in.Key, err = memguard.NewImmutableRandom(32)
	return
}
