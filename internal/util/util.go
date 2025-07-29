package util

import (
	"bytes"
	"encoding/gob"
)

// Serialize returns a []byte representing the passed value
func Serialize(value any) ([]byte, error) {
	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Deserialize will deserialize the passed []byte into the passed ptr any
func Deserialize(payload []byte, ptr any) (err error) {
	return gob.NewDecoder(bytes.NewBuffer(payload)).Decode(ptr)
}
