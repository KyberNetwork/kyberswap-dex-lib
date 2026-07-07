package decode

import (
	"encoding/binary"
	"errors"
	"math/big"
)

var ErrOutOfData = errors.New("out of data")

type BytesIterator struct {
	data []byte
}

func NewBytesIterator(data []byte) *BytesIterator {
	return &BytesIterator{data: data}
}

func (bi *BytesIterator) RemainingData() []byte {
	return bi.data
}

func (bi *BytesIterator) HasMore() bool {
	return len(bi.data) > 0
}

func (bi *BytesIterator) NextBytes(length int) ([]byte, error) {
	if len(bi.data) < length {
		return nil, ErrOutOfData
	}

	result := bi.data[:length]
	bi.data = bi.data[length:]

	return result, nil
}

func (bi *BytesIterator) NextUint8() (uint8, error) {
	result, err := bi.NextBytes(1) // nolint: gomnd
	if err != nil {
		return 0, err
	}

	return result[0], nil
}

func (bi *BytesIterator) NextUint16() (uint16, error) {
	result, err := bi.NextBytes(2) // nolint: gomnd
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(result), nil
}

func (bi *BytesIterator) NextUint24() (uint32, error) {
	result, err := bi.NextBytes(3) // nolint: gomnd
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(append([]byte{0}, result...)), nil
}

func (bi *BytesIterator) NextUint32() (uint32, error) {
	result, err := bi.NextBytes(4) // nolint: gomnd
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(result), nil
}

func (bi *BytesIterator) NextUint64() (uint64, error) {
	result, err := bi.NextBytes(8) // nolint: gomnd
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(result), nil
}

func (bi *BytesIterator) NextUint160() (*big.Int, error) {
	result, err := bi.NextBytes(20) // nolint: gomnd
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(result), nil
}

func (bi *BytesIterator) NextUint256() (*big.Int, error) {
	result, err := bi.NextBytes(32) // nolint: gomnd
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(result), nil
}
