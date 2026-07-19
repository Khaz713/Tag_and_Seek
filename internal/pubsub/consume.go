package pubsub

import (
	"bytes"
	"encoding/gob"
)

func DecodeGob[T any](data []byte) (T, error) {
	d := bytes.NewBuffer(data)
	dec := gob.NewDecoder(d)
	var dat T
	err := dec.Decode(&dat)
	if err != nil {
		return dat, err
	}
	return dat, nil
}
