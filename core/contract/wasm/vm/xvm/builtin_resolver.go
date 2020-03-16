package xvm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"unsafe"

	"github.com/xuperchain/xuperchain/core/xvm/exec"
	"github.com/xuperchain/xuperchain/core/xvm/runtime/emscripten"
)

func touint32(n int32) uint32 {
	return *(*uint32)(unsafe.Pointer(&n))
}

func hashFunc(name string) hash.Hash {
	switch name {
	case "sha256":
		return sha256.New()
	default:
		return nil
	}
}

func xvmHash(ctx exec.Context,
	nameptr uint32,
	inputptr uint32, inputlen uint32,
	outputptr uint32, outputlen uint32) uint32 {

	codec := exec.NewCodec(ctx)
	name := codec.CString(nameptr)
	input := codec.Bytes(inputptr, inputlen)
	output := codec.Bytes(outputptr, outputlen)

	hasher := hashFunc(name)
	if hasher == nil {
		exec.ThrowMessage(fmt.Sprintf("hash %s not found", name))
	}
	hasher.Write(input)
	out := hasher.Sum(nil)
	copy(output, out[:])
	return 0
}

type codec interface {
	Encode(in []byte) []byte
	Decode(in []byte) ([]byte, error)
}

func getCodec(name string) codec {
	switch name {
	case "hex":
		return hexCodec{}
	default:
		return nil
	}
}

type hexCodec struct{}

func (h hexCodec) Encode(in []byte) []byte {
	out := make([]byte, hex.EncodedLen(len(in)))
	hex.Encode(out, in)
	return out
}
func (h hexCodec) Decode(in []byte) ([]byte, error) {
	out := make([]byte, hex.DecodedLen(len(in)))
	_, err := hex.Decode(out, in)
	return out, err
}

func xvmEncode(ctx exec.Context,
	nameptr uint32,
	inputptr uint32, inputlen uint32,
	outputpptr uint32, outputLenPtr uint32) uint32 {

	codec := exec.NewCodec(ctx)
	name := codec.CString(nameptr)
	input := codec.Bytes(inputptr, inputlen)

	c := getCodec(name)
	if c == nil {
		exec.ThrowMessage(fmt.Sprintf("codec %s not found", name))
	}
	out := c.Encode(input)
	memptr, err := emscripten.Malloc(ctx, len(out))
	if err != nil {
		exec.ThrowError(err)
	}
	mem := codec.Bytes(memptr, uint32(len(out)))
	copy(mem, out)
	codec.SetUint32(outputpptr, memptr)
	codec.SetUint32(outputLenPtr, uint32(len(out)))
	return 0
}

func xvmDecode(ctx exec.Context,
	nameptr uint32,
	inputptr uint32, inputlen uint32,
	outputpptr uint32, outputLenPtr uint32) uint32 {

	codec := exec.NewCodec(ctx)
	name := codec.CString(nameptr)
	input := codec.Bytes(inputptr, inputlen)

	c := getCodec(name)
	if c == nil {
		exec.ThrowMessage(fmt.Sprintf("codec %s not found", name))
	}
	out, err := c.Decode(input)
	if err != nil {
		return 1
	}

	memptr, err := emscripten.Malloc(ctx, len(out))
	if err != nil {
		exec.ThrowError(err)
	}
	mem := codec.Bytes(memptr, uint32(len(out)))
	copy(mem, out)
	codec.SetUint32(outputpptr, memptr)
	codec.SetUint32(outputLenPtr, uint32(len(out)))
	return 0
}

var builtinResolver = exec.MapResolver(map[string]interface{}{
	"env._xvm_hash":   xvmHash,
	"env._xvm_encode": xvmEncode,
	"env._xvm_decode": xvmDecode,
})