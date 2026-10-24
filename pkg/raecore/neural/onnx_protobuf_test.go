package neural

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func buildTestONNXProtobufPayload(floats []float32) []byte {
	raw := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(raw[i*4:(i+1)*4], math.Float32bits(f))
	}
	// Field 9 (raw_data), wire type 2 (length-delimited): tag = (9<<3)|2 = 74 (0x4a)
	var msg []byte
	msg = append(msg, 0x4a, byte(len(raw)))
	msg = append(msg, raw...)
	return msg
}

func TestONNXBinaryProtobufRoundTripAndForwardPass(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "mini_bge_small.onnx")
	payload := buildTestONNXProtobufPayload([]float32{
		0.15, -0.25, 0.35, 0.45, -0.55, 0.65, 0.75, -0.85,
	})
	if err := os.WriteFile(tmpFile, payload, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	parsed := ParseONNXProtobufTensors(payload)
	if len(parsed) != 1 || len(parsed[0].FloatData) != 8 {
		t.Fatalf("expected 1 parsed tensor with 8 floats, got %+v", parsed)
	}

	engine := NewONNXRuntimeEngine("bge-small-en-v1.5-onnx", 64)
	count, err := engine.LoadONNXBinaryFile(tmpFile)
	if err != nil || count != 1 {
		t.Fatalf("LoadONNXBinaryFile failed: count=%d, err=%v", count, err)
	}
	vec := engine.ForwardPass("RAE-Suite-go v5.3.0 binary ONNX protobuf verification")
	if len(vec) != 64 {
		t.Fatalf("expected 64-d embedding, got %d", len(vec))
	}
}

func TestONNXProtobufWireTypesAndRuntimeEdgeCases(t *testing.T) {
	// Test wireType 0 (varint), wireType 1 (64-bit), wireType 5 (32-bit float), nested wireType 2, and invalid wireType
	var wireBuf []byte
	// Field 1, wireType 0 (varint): tag = (1<<3)|0 = 0x08, val = 42
	wireBuf = append(wireBuf, 0x08, 42)
	// Field 2, wireType 1 (64-bit): tag = (2<<3)|1 = 0x11, 8 bytes
	wireBuf = append(wireBuf, 0x11, 1, 2, 3, 4, 5, 6, 7, 8)
	// Field 3, wireType 5 (32-bit float): tag = (3<<3)|5 = 0x1d, 4 bytes for 0.5f
	var fBytes [4]byte
	binary.LittleEndian.PutUint32(fBytes[:], math.Float32bits(0.5))
	wireBuf = append(wireBuf, 0x1d)
	wireBuf = append(wireBuf, fBytes[:]...)
	// Nested message in field 7, wireType 2: tag = (7<<3)|2 = 0x3a
	nested := buildTestONNXProtobufPayload([]float32{0.1, 0.2, 0.3, 0.4})
	wireBuf = append(wireBuf, 0x3a, byte(len(nested)))
	wireBuf = append(wireBuf, nested...)

	tensors := ParseONNXProtobufTensors(wireBuf)
	if len(tensors) < 2 {
		t.Fatalf("expected scalar_float and nested onnx_initializer tensors, got %+v", tensors)
	}

	// Test missing file and empty protobuf file in LoadONNXBinaryFile
	eng := NewONNXRuntimeEngine("default-dim", 0)
	if eng.Dimension != 64 {
		t.Fatalf("expected default dimension 64, got %d", eng.Dimension)
	}
	if _, err := eng.LoadONNXBinaryFile("/non/existent/file.onnx"); err == nil {
		t.Fatalf("expected error loading non-existent ONNX file")
	}
	emptyFile := filepath.Join(t.TempDir(), "empty.onnx")
	_ = os.WriteFile(emptyFile, []byte{0x08, 0x01}, 0o644)
	if cnt, err := eng.LoadONNXBinaryFile(emptyFile); err != nil || cnt != 0 {
		t.Fatalf("expected 0 tensors from empty file, got %d (%v)", cnt, err)
	}

	// LayerNorm empty & eps <= 0, ForwardPass empty text, EmbedBatch
	if LayerNorm(nil, 0) != nil {
		t.Fatalf("expected nil from LayerNorm(nil)")
	}
	normed := LayerNorm([]float64{1.0, 2.0, 3.0}, 0)
	if len(normed) != 3 {
		t.Fatalf("expected 3 elements from LayerNorm")
	}
	padVec := eng.ForwardPass("   ")
	if len(padVec) != 64 {
		t.Fatalf("expected 64-d vector for whitespace input")
	}
	batchVecs := eng.EmbedBatch([]string{"first text", "second text"})
	if len(batchVecs) != 2 || len(batchVecs[0]) != 64 {
		t.Fatalf("unexpected EmbedBatch output")
	}
}

