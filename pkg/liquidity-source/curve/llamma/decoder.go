package llamma

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type CurveLlammaResult struct {
	ActiveBand  *int256.Int
	MinBand     *int256.Int
	MaxBand     *int256.Int
	PriceOracle *uint256.Int
	DynamicFee  *uint256.Int
	AdminFee    *uint256.Int
	Bands       []Band
}

const (
	wordSize = 32
)

var (
	twoPow253       = bignumber.NewBig("0x2000000000000000000000000000000000000000000000000000000000000000")
	twoPow254       = bignumber.NewBig("0x4000000000000000000000000000000000000000000000000000000000000000")
	twoPow254Minus1 = bignumber.NewBig("0x3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	twoPow256       = new(big.Int).Lsh(big.NewInt(1), 256)
)

func decode(data []byte) (*CurveLlammaResult, error) {
	if len(data) < 7*wordSize {
		return nil, ErrNotEnoughData
	}

	var result CurveLlammaResult
	var offset = 0

	result.ActiveBand, offset = readInt256(data, offset)
	_, offset = readBigInt256(data, offset) // skip p_oracle_up
	result.MinBand, offset = readInt256(data, offset)
	result.MaxBand, offset = readInt256(data, offset)
	result.PriceOracle, offset = readUInt256(data, offset)
	result.DynamicFee, offset = readUInt256(data, offset)
	result.AdminFee, offset = readUInt256(data, offset)

	var bandValue *big.Int
	for offset < len(data) {
		bandValue, offset = readBigInt256(data, offset)

		index := new(big.Int).And(bandValue, twoPow254Minus1)
		if index.Cmp(twoPow253) >= 0 {
			index.Sub(index, twoPow254)
		}

		bandX := new(uint256.Int)
		bandY := new(uint256.Int)
		if bandValue.Bit(255) == 1 {
			bandX, offset = readUInt256(data, offset)
		}
		if bandValue.Bit(254) == 1 {
			bandY, offset = readUInt256(data, offset)
		}

		result.Bands = append(result.Bands, Band{
			Index: index.Int64(),
			BandX: bandX,
			BandY: bandY,
		})
	}

	str, _ := json.Marshal(result)
	fmt.Println("RE", string(str))

	return &result, nil
}

func readInt256(data []byte, offset int) (ret *int256.Int, endByte int) {
	endByte = offset + 32
	bytesVal := data[offset:endByte]
	retBI := new(big.Int).SetBytes(bytesVal)
	if (bytesVal[0] & 0x80) != 0 {
		retBI.Sub(retBI, twoPow256)
	}
	ret = int256.MustFromBig(retBI)
	return
}

func readUInt256(data []byte, offset int) (ret *uint256.Int, endByte int) {
	endByte = offset + 32
	ret = new(uint256.Int)
	ret.SetBytes(data[offset:endByte])
	return
}

func readBigInt256(data []byte, offset int) (ret *big.Int, endByte int) {
	endByte = offset + 32
	ret = new(big.Int)
	ret.SetBytes(data[offset:endByte])
	return
}
