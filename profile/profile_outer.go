package profile

import (
	"github.com/awnumar/memguard"
	"github.com/mgren/ogive/crypt"
)

func (od *OuterData) pack(in *InnerData, key *memguard.LockedBuffer) error {
	gcm, err := crypt.GetGCM(key, 0)
	if err != nil {
		return err
	}

	// No need to protect this, but it saves us importing rand
	nonce, err := memguard.NewImmutableRandom(gcm.NonceSize())
	if err != nil {
		return err
	}

	marshal, err := in.marshalBinaryLocked()
	if err != nil {
		return err
	}

	od.Inner = gcm.Seal(nonce.Buffer(), nonce.Buffer(), marshal.Buffer(), nil)
	marshal.Destroy()
	return nil
}

func (od *OuterData) unpack(key *memguard.LockedBuffer) (*InnerData, error) {
	gcm, err := crypt.GetGCM(key, 0)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, data := od.Inner[:nonceSize], od.Inner[nonceSize:]

	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}

	inner := InnerData{}
	locked, err := memguard.NewImmutableFromBytes(plain)
	if err != nil {
		return nil, err
	}

	err = inner.unmarshalBinaryLocked(locked)
	return &inner, err
}
