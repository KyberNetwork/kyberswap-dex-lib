package pack

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Read<Type> receives the []byte encoded data, and index to start to read the <Type>.
// Read<Type> returns the value for <Type>, and next start position to continue reading the encoded data.

func ReadAddress(data []byte, startByte int) (common.Address, int) {
	return common.BytesToAddress(data[startByte : startByte+20]), startByte + 20
}

func ReadBoolean(data []byte, startByte int) (ret bool, endByte int) {
	endByte = startByte + 1
	if data[startByte] == 1 {
		ret = true
	}
	return
}

func ReadUInt8(data []byte, startByte int) (uint8, int) {
	return data[startByte], startByte + 1
}

func ReadUInt24(data []byte, startByte int) (ret UInt24, endByte int) {
	endByte = startByte + 3
	for i := startByte; i < endByte; i++ {
		ret += UInt24(data[i] << (endByte - i - 1))
	}
	return
}

func ReadUInt32(data []byte, startByte int) (ret uint32, endByte int) {
	endByte = startByte + 4
	ret = binary.BigEndian.Uint32(data[startByte:endByte])
	return
}

func ReadUInt96(data []byte, startByte int) (UInt96, int) {
	endByte := startByte + 12
	ret := new(big.Int)
	ret.SetBytes(data[startByte:endByte])
	return ret, endByte
}

func ReadUInt160(data []byte, startByte int) (UInt160, int) {
	endByte := startByte + 20
	ret := new(big.Int)
	ret.SetBytes(data[startByte:endByte])
	return ret, endByte
}

func ReadBigInt(data []byte, startByte int) (ret *big.Int, endByte int) {
	endByte = startByte + 16
	ret = new(big.Int)
	ret.SetBytes(data[startByte:endByte])
	return
}

func ReadBytes(data []byte, startByte int) (ret []byte, endByte int) {
	// Use the first 4 bytes for the bytes length
	length := binary.BigEndian.Uint32(data[startByte : startByte+4])
	endByte = startByte + 4 + int(length)
	ret = data[startByte+4 : endByte]
	return
}

func ReadSliceAddress(data []byte, startByte int) (ret []common.Address, endByte int) {
	length, startByte := ReadUInt8(data, startByte)
	ret = make([]common.Address, length)

	for i := uint8(0); i < length; i++ {
		ret[i], startByte = ReadAddress(data, startByte)
	}
	endByte = startByte
	return
}

func ReadSliceBigInt(data []byte, startByte int) (ret []*big.Int, endByte int) {
	length, startByte := ReadUInt8(data, startByte)
	ret = make([]*big.Int, length)

	for i := uint8(0); i < length; i++ {
		ret[i], startByte = ReadBigInt(data, startByte)
	}
	endByte = startByte
	return
}

func ReadSliceBytes(data []byte, startByte int) (ret [][]byte, endByte int) {
	length, startByte := ReadUInt8(data, startByte)
	ret = make([][]byte, length)

	for i := uint8(0); i < length; i++ {
		ret[i], startByte = ReadBytes(data, startByte)
	}
	endByte = startByte
	return
}
