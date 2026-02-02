package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// ProtobufCodec implements Protocol Buffers encoding/decoding
type ProtobufCodec struct{}

func (c *ProtobufCodec) Encode(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("value must implement proto.Message interface, got %T", v)
	}
	return proto.Marshal(msg)
}

func (c *ProtobufCodec) Decode(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("value must implement proto.Message interface, got %T", v)
	}
	return proto.Unmarshal(data, msg)
}

func (c *ProtobufCodec) Name() string {
	return "protobuf"
}
