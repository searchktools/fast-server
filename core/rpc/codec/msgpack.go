package codec

import (
	"bytes"
	"encoding/gob"
)

// MsgPackCodec implements MessagePack-like encoding using Go's gob for now
// TODO: Replace with actual msgpack library for better performance
type MsgPackCodec struct{}

func (c *MsgPackCodec) Encode(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *MsgPackCodec) Decode(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}

func (c *MsgPackCodec) Name() string {
	return "msgpack"
}
