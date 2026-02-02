package codec

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestJSONCodec(t *testing.T) {
	codec := &JSONCodec{}

	// Test encode/decode
	type TestStruct struct {
		Name  string
		Value int
	}

	original := &TestStruct{Name: "test", Value: 42}

	// Encode
	data, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	// Decode
	decoded := &TestStruct{}
	if err := codec.Decode(data, decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	// Verify
	if decoded.Name != original.Name || decoded.Value != original.Value {
		t.Errorf("Mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestProtobufCodec(t *testing.T) {
	codec := &ProtobufCodec{}

	// Create test message
	original := wrapperspb.Int32(42)

	// Encode
	data, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	// Decode
	decoded := &wrapperspb.Int32Value{}
	if err := codec.Decode(data, decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	// Verify
	if decoded.Value != original.Value {
		t.Errorf("Mismatch: got %d, want %d", decoded.Value, original.Value)
	}
}

func TestProtobufCodecInvalidType(t *testing.T) {
	codec := &ProtobufCodec{}

	// Try to encode non-proto message
	_, err := codec.Encode("not a proto message")
	if err == nil {
		t.Error("Expected error for non-proto message")
	}
}

func BenchmarkJSONEncode(b *testing.B) {
	codec := &JSONCodec{}
	data := map[string]interface{}{
		"name":  "benchmark",
		"value": 123,
		"items": []int{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = codec.Encode(data)
	}
}

func BenchmarkJSONDecode(b *testing.B) {
	codec := &JSONCodec{}
	data := map[string]interface{}{
		"name":  "benchmark",
		"value": 123,
	}
	encoded, _ := codec.Encode(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded map[string]interface{}
		_ = codec.Decode(encoded, &decoded)
	}
}

func BenchmarkProtobufEncode(b *testing.B) {
	codec := &ProtobufCodec{}
	msg := wrapperspb.String("benchmark message with some data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = codec.Encode(msg)
	}
}

func BenchmarkProtobufDecode(b *testing.B) {
	codec := &ProtobufCodec{}
	msg := wrapperspb.String("benchmark message")
	data, _ := proto.Marshal(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded := &wrapperspb.StringValue{}
		_ = codec.Decode(data, decoded)
	}
}
