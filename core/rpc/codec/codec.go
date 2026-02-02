package codec

import (
	"encoding/json"
	"errors"
)

var (
	ErrUnsupportedCodec = errors.New("unsupported codec")
)

// Codec defines the interface for encoding/decoding RPC messages
type Codec interface {
	// Encode encodes a value to bytes
	Encode(v interface{}) ([]byte, error)

	// Decode decodes bytes to a value
	Decode(data []byte, v interface{}) error

	// Name returns the codec name
	Name() string
}

// CodecType represents the codec type
type CodecType byte

const (
	CodecJSON     CodecType = 0x01
	CodecMsgPack  CodecType = 0x02
	CodecProtobuf CodecType = 0x03
)

// GetCodec returns a codec by type
func GetCodec(typ CodecType) (Codec, error) {
	switch typ {
	case CodecJSON:
		return &JSONCodec{}, nil
	case CodecMsgPack:
		return &MsgPackCodec{}, nil
	case CodecProtobuf:
		return &ProtobufCodec{}, nil
	default:
		return nil, ErrUnsupportedCodec
	}
}

// JSONCodec implements JSON encoding/decoding
type JSONCodec struct{}

func (c *JSONCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c *JSONCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (c *JSONCodec) Name() string {
	return "json"
}
