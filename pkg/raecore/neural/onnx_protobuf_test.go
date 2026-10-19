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
