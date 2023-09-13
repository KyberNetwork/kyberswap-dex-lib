package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var testData = []struct {
	encodingSwap types.EncodingSwap
	packingData  TraderJoeV2
	packedData   string
}{
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/c33c4fd26d1e6b41557bde9e7fbc6702e9454ed2/foundry-tests/executor/Executor.TraderJoeV2.t.sol#L72-L78
	{
		encodingSwap: types.EncodingSwap{
			Exchange:      valueobject.ExchangeTraderJoeV21,
			Recipient:     "0x185a4dc360CE69bDCceE33b3784B0282f7961aea",
			Pool:          "0xD446eb1660F766d533BeCeEf890Df7A69d26f7d1",
			TokenIn:       "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
			TokenOut:      "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
			CollectAmount: big.NewInt(100000000),
		},
		packingData: TraderJoeV2{
			Recipient:           common.HexToAddress("0x185a4dc360CE69bDCceE33b3784B0282f7961aea"),
			Pool:                common.HexToAddress("0xD446eb1660F766d533BeCeEf890Df7A69d26f7d1"),
			TokenIn:             common.HexToAddress("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"),
			TokenOut:            common.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			PackedCollectAmount: big.NewInt(100000000),
		},
		packedData: "000000000000000000000000185a4dc360ce69bdccee33b3784b0282f7961aea000000000000000000000000d446eb1660f766d533beceef890df7a69d26f7d1000000000000000000000000b97ef9ef8734c71904d8002f8b6bc66dd9c48a6e000000000000000000000000b31f66aa3c1e785363f0875a1b74e27b85fd66c70000000000000000000000000000000000000000000000000000000005f5e100",
	},
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/c33c4fd26d1e6b41557bde9e7fbc6702e9454ed2/foundry-tests/executor/Executor.TraderJoeV2.t.sol#L200-L208
	{
		encodingSwap: types.EncodingSwap{
			Exchange:      valueobject.ExchangeTraderJoeV20,
			Recipient:     "0x87EB2F90d7D0034571f343fb7429AE22C1Bd9F72",
			Pool:          "0xB5352A39C11a81FE6748993D586EC448A01f08b5",
			TokenIn:       "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
			TokenOut:      "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
			CollectAmount: big.NewInt(0),
		},
		packingData: TraderJoeV2{
			Recipient: common.HexToAddress("0x87EB2F90d7D0034571f343fb7429AE22C1Bd9F72"),
			Pool:      common.HexToAddress("0xB5352A39C11a81FE6748993D586EC448A01f08b5"),
			TokenIn:   common.HexToAddress("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"),
			TokenOut:  common.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"),
			PackedCollectAmount: func() *big.Int {
				amount, _ := new(big.Int).SetString("57896044618658097711785492504343953926634992332820282019728792003956564819968", 10)
				return amount
			}(),
		},
		packedData: "00000000000000000000000087eb2f90d7d0034571f343fb7429ae22c1bd9f72000000000000000000000000b5352a39c11a81fe6748993d586ec448a01f08b5000000000000000000000000b97ef9ef8734c71904d8002f8b6bc66dd9c48a6e000000000000000000000000b31f66aa3c1e785363f0875a1b74e27b85fd66c78000000000000000000000000000000000000000000000000000000000000000",
	},
}

func TestBuildTraderJoeV2(t *testing.T) {
	t.Parallel()

	for idx, data := range testData {
		t.Run(fmt.Sprintf("it should build correctly %d", idx), func(t *testing.T) {
			result, err := buildTraderJoeV2(data.encodingSwap)
			assert.NoError(t, err)
			assert.EqualValues(t, data.packingData, result)
		})
	}
}

func TestPackTraderJoeV2(t *testing.T) {
	t.Parallel()

	for idx, data := range testData {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packTraderJoeV2(data.packingData)
			assert.NoError(t, err)
			assert.Equal(t, data.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackTraderJoeV2(t *testing.T) {
	t.Parallel()

	for idx, data := range testData {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackTraderJoeV2(common.Hex2Bytes(data.packedData))
			assert.NoError(t, err)
			assert.EqualValues(t, data.packingData, result)
		})
	}
}
