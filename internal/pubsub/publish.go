package pubsub

import (
	"bytes"
	"encoding/gob"
)

func EncodeGob[T any](v T) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
