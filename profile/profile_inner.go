package profile

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/awnumar/memguard"
	"reflect"
)

func (id *InnerData) marshalBinaryLocked() (buf *memguard.LockedBuffer, err error) {
	defer id.wipe()
	var head, body bytes.Buffer
	var cnt int
	v := reflect.ValueOf(id).Elem()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.CanInterface() {
			t := f.Interface()
			switch t := t.(type) {
			case *memguard.LockedBuffer:
				err = binary.Write(&head, binary.LittleEndian, uint32(t.Size()))
				if err != nil {
					return nil, err
				}

				cnt, err = body.Write(t.Buffer())
				if err != nil {
					return nil, err
				}
				if cnt != t.Size() {
					return nil, errors.New("Incorrect number of bytes written.")
				}

				t.Destroy()
			case string:
				err = binary.Write(&head, binary.LittleEndian, uint32(len(t)))
				if err != nil {
					return nil, err
				}

				cnt, err = body.Write([]byte(t))
				if err != nil {
					return nil, err
				}
				if cnt != len(t) {
					return nil, errors.New("Incorrect number of bytes written.")
				}
			default:
				return nil, errors.New("Incorrect field type.")
			}
		}
	}

	// bytes.Join copies the data, so create two LockedBuffers to zero out underlying memory regions
	headLocked, err := memguard.NewImmutableFromBytes(head.Bytes())
	bodyLocked, err := memguard.NewImmutableFromBytes(body.Bytes())

	return memguard.Concatenate(headLocked, bodyLocked)
}

func (id *InnerData) unmarshalBinaryLocked(data *memguard.LockedBuffer) error {
	v, total := reflect.ValueOf(id).Elem(), uint32(24) // four bytes per value times six fields
	data.MakeMutable()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.CanInterface() {
			size := binary.LittleEndian.Uint32(data.Buffer()[4*i : 4*(1+i)])
			if size == 0 {
				return errors.New("Corruped profile file.")
			}
			t := f.Interface()
			switch t.(type) {
			case *memguard.LockedBuffer:
				b, err := memguard.NewImmutableFromBytes(data.Buffer()[total : total+size])
				if err != nil {
					return err
				}
				f.Set(reflect.ValueOf(b))
				total += size
			case string:
				f.SetString(string(data.Buffer()[total : total+size]))
				total += size
			default:
				return errors.New("Incorrect field type.")
			}
		}
	}

	data.Destroy()
	return nil
}

func (id *InnerData) wipe() {
	v := reflect.ValueOf(id).Elem()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.CanInterface() {
			t := f.Interface()
			// We don't care about string data here
			switch t := t.(type) {
			case *memguard.LockedBuffer:
				t.Destroy()
			default:
			}
		}
	}
}
