//go:generate go run ./cmd/generate

package poolsimulatormsgp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"

	"github.com/klauspost/compress/snappy"
	"github.com/tinylib/msgp/msgp"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

var (
	writerPool = sync.Pool{New: func() any { return msgp.NewWriterBuf(nil, nil) }}
	readerPool = sync.Pool{New: func() any { return msgp.NewReaderBuf(nil, nil) }}
)

// EncodePoolSimulatorsMap encode a map from pool ID to IPoolSimulator with Snappy compression
func EncodePoolSimulatorsMap(poolsMap map[string]pool.IPoolSimulator) ([]byte, error) {
	if poolsMap == nil {
		return nil, nil
	}

	buf := new(bytes.Buffer)
	zw := snappy.NewBufferedWriter(buf)
	en := writerPool.Get().(*msgp.Writer)
	defer func() { writerPool.Put(en) }()
	en.Reset(zw)

	err := en.WriteArrayHeader(uint32(len(poolsMap)))
	if err != nil {
		return nil, msgp.WrapError(err, "ArrayHeader")
	}
	for poolID, pool := range poolsMap {
		err = en.WriteString(poolID)
		if err != nil {
			return nil, msgp.WrapError(err, poolID, "poolID")
		}
		dexType, encodable := dispatchPoolSimulator(pool)
		if dexType == "" {
			return nil, msgp.WrapError(errors.New("empty dexType"), poolID, "dexType")
		}
		if encodable == nil {
			return nil, msgp.WrapError(errors.New("empty encodable"), poolID, "pool")
		}
		if hookable, ok := pool.(MsgpHookable); ok {
			if err := hookable.BeforeMsgpEncode(); err != nil {
				return nil, msgp.WrapError(fmt.Errorf("BeforeMsgpEncode() returns an error: %w", err), poolID, "pool")
			}
		}
		err = en.WriteString(dexType)
		if err != nil {
			return nil, msgp.WrapError(err, poolID, "dexType")
		}
		err = encodable.EncodeMsg(en)
		if err != nil {
			return nil, msgp.WrapError(err, poolID, "pool")
		}
	}

	err = en.Flush()
	if err != nil {
		return nil, err
	}
	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodePoolSimulatorsMap decodes an encoded and Snappy compressed map from pool ID to IPoolSimulator
func DecodePoolSimulatorsMap(encoded []byte) (map[string]pool.IPoolSimulator, error) {
	if encoded == nil {
		return nil, nil
	}

	zw := snappy.NewReader(bytes.NewReader(encoded))
	de := readerPool.Get().(*msgp.Reader)
	defer func() { readerPool.Put(de) }()
	de.Reset(zw)

	n, err := de.ReadArrayHeader()
	if err != nil {
		return nil, msgp.WrapError(err, "ArrayHeader")
	}
	poolsMap := make(map[string]pool.IPoolSimulator, int(n))
	for i := uint32(0); i < n; i++ {
		poolID, err := de.ReadString()
		if err != nil {
			return nil, msgp.WrapError(err, poolID, "poolID")
		}
		dexType, err := de.ReadString()
		if err != nil {
			return nil, msgp.WrapError(err, poolID, "dexType")
		}
		pool, decodable := undispatchPoolSimulator(dexType)
		err = decodable.DecodeMsg(de)
		if err != nil {
			return nil, msgp.WrapError(err, poolID, "pool", i)
		}
		if hookable, ok := pool.(MsgpHookable); ok {
			if err := hookable.AfterMsgpDecode(); err != nil {
				return nil, msgp.WrapError(fmt.Errorf("AfterMsgpDecode() returns an error: %w", err), poolID, "pool")
			}
		}
		poolsMap[poolID] = pool
	}

	return poolsMap, nil
}

// EncodePoolSimulator encodes [pool.IPoolSimulator] as the following format
//
// [2-byte len(DexType)] + [DexType] + [encoded msgp.Encodable]
func EncodePoolSimulator(sim pool.IPoolSimulator) []byte {
	if sim == nil {
		return nil
	}

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
	if encoded == nil {
		return nil
	}

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
