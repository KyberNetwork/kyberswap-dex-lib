package ilyris

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func poolCreatedData(swapFeeBps uint64, policy, marketGuard, pool string) []byte {
	data := make([]byte, 128)
	big.NewInt(int64(swapFeeBps)).FillBytes(data[0:32])
	copy(data[32+12:64], common.HexToAddress(policy).Bytes())
	copy(data[64+12:96], common.HexToAddress(marketGuard).Bytes())
	copy(data[96+12:128], common.HexToAddress(pool).Bytes())
	return data
}

func TestDecodePoolCreatedTable(t *testing.T) {
	topic := common.HexToHash(poolCreatedTopic)
	factory := common.HexToAddress("0x4A943A11a6fFBF8D204Df4d5A080Ca741697ca33")
	poolAddr := "0x90d0950065c567b9324a08a9aae8a28890fbab16"
	tokenX := "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	tokenY := "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
	indexed := []common.Hash{
		topic,
		common.BytesToHash(common.HexToAddress(tokenX).Bytes()),
		common.BytesToHash(common.HexToAddress(tokenY).Bytes()),
		common.BigToHash(big.NewInt(10)),
	}

	f := NewPoolFactory(DexType)

	tests := []struct {
		name    string
		log     types.Log
		wantErr bool
		want    string
	}{
		{
			name: "pool is data word 3 not the factory",
			log: types.Log{
				Address: factory,
				Topics:  indexed,
				Data:    poolCreatedData(30, "0x1111111111111111111111111111111111111111", "0xDd74981476f81c8e45e962Af6DF886a3c5788816", poolAddr),
			},
			want: poolAddr,
		},
		{
			name: "short data",
			log: types.Log{
				Address: factory,
				Topics:  indexed,
				Data:    make([]byte, 96),
			},
			wantErr: true,
		},
		{
			name: "zero pool address",
			log: types.Log{
				Address: factory,
				Topics:  indexed,
				Data:    poolCreatedData(30, "0x1111111111111111111111111111111111111111", "0xDd74981476f81c8e45e962Af6DF886a3c5788816", common.Address{}.Hex()),
			},
			wantErr: true,
		},
		{
			name: "missing indexed topics",
			log: types.Log{
				Address: factory,
				Topics:  []common.Hash{topic},
				Data:    poolCreatedData(30, "0x1111111111111111111111111111111111111111", "0xDd74981476f81c8e45e962Af6DF886a3c5788816", poolAddr),
			},
			wantErr: true,
		},
		{
			name: "wrong topic",
			log: types.Log{
				Address: factory,
				Topics: []common.Hash{
					common.HexToHash("0xdead"),
					indexed[1], indexed[2], indexed[3],
				},
				Data: poolCreatedData(30, "0x1111111111111111111111111111111111111111", "0xDd74981476f81c8e45e962Af6DF886a3c5788816", poolAddr),
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := f.DecodePoolCreated(tc.log)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !strings.EqualFold(got.Address, tc.want) {
				t.Fatalf("address %s, want %s", got.Address, tc.want)
			}
			if strings.EqualFold(got.Address, factory.Hex()) {
				t.Fatal("keyed as the factory")
			}
			if got.Tokens[0].Address != tokenX || got.Tokens[1].Address != tokenY {
				t.Fatalf("tokens %s %s", got.Tokens[0].Address, got.Tokens[1].Address)
			}

			// ABI unpack of the same data must agree with the data[96:128] word.
			vals, err := factoryABI.Unpack("PoolCreated", tc.log.Data)
			if err != nil {
				t.Fatalf("abi unpack: %v", err)
			}
			unpacked, ok := vals[3].(common.Address)
			if !ok {
				t.Fatalf("pool field type %T", vals[3])
			}
			if !strings.EqualFold(unpacked.Hex(), got.Address) {
				t.Fatalf("abi pool %s != decoded %s", unpacked.Hex(), got.Address)
			}
			if common.BytesToAddress(tc.log.Data[96:128]) != unpacked {
				t.Fatal("data[96:128] disagrees with abi word 3")
			}
		})
	}
}
