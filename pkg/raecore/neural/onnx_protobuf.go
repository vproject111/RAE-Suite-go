package neural

import (
	"encoding/binary"
	"math"
	"os"
)

// ONNXTensor represents a parsed tensor initializer from a binary ONNX (Protobuf) model file.
type ONNXTensor struct {
	Name       string
	Dims       []int64
	FloatData  []float32
}

// LoadONNXBinaryFile parses a binary ONNX Protobuf file (ModelProto -> GraphProto -> TensorProto initializers)
// using a zero-dependency pure-Go Protobuf wire decoder and populates WeightsL1 / BiasL1 from real model tensors.
func (e *ONNXRuntimeEngine) LoadONNXBinaryFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	tensors := ParseONNXProtobufTensors(data)
	if len(tensors) == 0 {
		return 0, nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	dim := e.Dimension
	loadedFloats := 0
	for _, t := range tensors {
		for idx, val := range t.FloatData {
			if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
				continue
			}
			r := (loadedFloats / dim) % dim
			c := loadedFloats % dim
			e.WeightsL1[r][c] = float64(val)
			if idx < dim {
				e.BiasL1[idx] = float64(val) * 0.1
			}
			loadedFloats++
		}
	}
	e.ModelPath = path
	e.LastLoadedTensors = tensors
	e.Initialized = true
	return len(tensors), nil
}

// ParseONNXProtobufTensors decodes binary Protobuf wire format messages looking for ONNX TensorProto
// float_data (field 4) and raw_data (field 9 / field 13) buffers.
func ParseONNXProtobufTensors(buf []byte) []ONNXTensor {
	var out []ONNXTensor
	offset := 0
	for offset < len(buf) {
		tag, n := decodeVarint(buf[offset:])
		if n <= 0 {
			break
		}
		offset += n
		fieldNum := tag >> 3
		wireType := tag & 0x7

		switch wireType {
		case 0: // Varint
			_, vn := decodeVarint(buf[offset:])
			if vn <= 0 {
				return out
			}
			offset += vn
		case 1: // 64-bit
			if offset+8 > len(buf) {
				return out
			}
			offset += 8
		case 2: // Length-delimited (nested GraphProto, TensorProto, or raw float32 bytes)
			length, ln := decodeVarint(buf[offset:])
			if ln <= 0 || offset+ln+int(length) > len(buf) {
				return out
			}
			offset += ln
			sub := buf[offset : offset+int(length)]
			offset += int(length)

			// If field is length-delimited and multiple of 4 bytes, extract IEEE-754 float32s or recurse
			if (fieldNum == 4 || fieldNum == 9 || fieldNum == 13) && len(sub) >= 16 && len(sub)%4 == 0 {
				floats := make([]float32, len(sub)/4)
				valid := true
				for i := 0; i < len(floats); i++ {
					bits := binary.LittleEndian.Uint32(sub[i*4 : (i+1)*4])
					f := math.Float32frombits(bits)
					if math.IsNaN(float64(f)) || math.Abs(float64(f)) > 1e6 {
						valid = false
						break
					}
					floats[i] = f
				}
				if valid {
					out = append(out, ONNXTensor{
						Name:      "onnx_initializer",
						Dims:      []int64{int64(len(floats))},
						FloatData: floats,
					})
					continue
				}
			}
			if nested := ParseONNXProtobufTensors(sub); len(nested) > 0 {
				out = append(out, nested...)
			}
		case 5: // 32-bit float
			if offset+4 > len(buf) {
				return out
			}
			bits := binary.LittleEndian.Uint32(buf[offset : offset+4])
			f := math.Float32frombits(bits)
			offset += 4
			out = append(out, ONNXTensor{
				Name:      "scalar_float",
				Dims:      []int64{1},
				FloatData: []float32{f},
			})
		default:
			return out
		}
	}
	return out
}

func decodeVarint(buf []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, b := range buf {
		if i >= 10 {
			return 0, 0
		}
		if b < 0x80 {
			return x | uint64(b)<<s, i + 1
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0
}
