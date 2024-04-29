//go:generate go run ./cmd/generate

package poolsimulatormsgp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/tinylib/msgp/msgp"
)

var (
	poolSimulatorTypes = make(map[string]reflect.Type)
)

// RegisterPoolSimulator registers the concrete types of an IPoolSimulator and its discriminator string.
// This function is not thread-safe and should be only call in init().
func RegisterPoolSimulator(dexType string, sim pool.IPoolSimulator) {
	poolSimulatorTypes[dexType] = reflect.ValueOf(sim).Elem().Type()
}

func dispatchRegisteredPoolSimulator(sim pool.IPoolSimulator) (dexName string, encodable msgp.Encodable) {
	typ := reflect.ValueOf(sim).Elem().Type()
	for name, t := range poolSimulatorTypes {
		if typ == t {
			dexName = name
			encodable = reflect.ValueOf(sim).Interface().(msgp.Encodable)
			break
		}
	}
	return
}

func undispatchRegisteredPoolSimulator(dexName string) (sim pool.IPoolSimulator, decodable msgp.Decodable) {
	for name, typ := range poolSimulatorTypes {
		if dexName == name {
			impl := reflect.New(typ)
			sim = impl.Interface().(pool.IPoolSimulator)
			decodable = impl.Interface().(msgp.Decodable)
		}
	}
	return
}

// EncodePoolSimulator encodes [pool.IPoolSimulator] as the following format
//
// [2-byte len(DexType)] + [DexType] + [encoded msgp.Encodable]
func EncodePoolSimulator(sim pool.IPoolSimulator) []byte {
	dexType, encodable := dispatchPoolSimulator(sim)
	if dexType == "" {
		panic("empty dexType")
	}
	if encodable == nil {
		panic("nil encodable")
	}

	var (
		dexTypeLenBuf = [2]byte{}
		buf           = new(bytes.Buffer)
	)

	// truncate too long dexType
	if len(dexType) > math.MaxUint16 {
		dexType = dexType[:math.MaxUint16]
	}
	binary.BigEndian.PutUint16(dexTypeLenBuf[:], uint16(len(dexType)))
	if _, err := buf.Write(dexTypeLenBuf[:]); err != nil {
		panic(fmt.Sprintf("could not write len(DexType): %s", err))
	}

	if _, err := buf.Write([]byte(dexType)); err != nil {
		panic(fmt.Sprintf("could not write DexType: %s", err))
	}

	if err := msgp.Encode(buf, encodable); err != nil {
		panic(fmt.Sprintf("could not encode msgp.Encodable: %s", err))
	}

	return buf.Bytes()
}

// DecodePoolSimulator decodes an encoded [pool.IPoolSimulator] as the following format
//
// [2-byte len(DexType)] + [DexType] + [encoded msgp.Encodable]
func DecodePoolSimulator(encoded []byte) pool.IPoolSimulator {
	var (
		buf           = bytes.NewBuffer(encoded)
		dexTypeLenBuf = [2]byte{}
		dexTypeLen    uint16
		dexTypeBytes  []byte
		dexType       string
	)

	if _, err := buf.Read(dexTypeLenBuf[:]); err != nil {
		panic(fmt.Sprintf("could not read len(DexType): %s", err))
	}
	dexTypeLen = binary.BigEndian.Uint16(dexTypeLenBuf[:])

	dexTypeBytes = make([]byte, int(dexTypeLen))
	if _, err := buf.Read(dexTypeBytes); err != nil {
		panic(fmt.Sprintf("could not read DexType: %s", err))
	}
	dexType = string(dexTypeBytes)

	if dexType == "" {
		panic("empty DexType")
	}

	sim, decodable := undispatchPoolSimulator(dexType)
	if err := msgp.Decode(buf, decodable); err != nil {
		panic(fmt.Sprintf("could not decode msgp.Decodable: %s", err))
	}

	if hookable, ok := sim.(MsgpHookable); ok {
		if err := hookable.AfterMsgpDecode(); err != nil {
			panic(fmt.Sprintf("AfterDecode() returns an error: %s", err))
		}
	}

	return sim
}
