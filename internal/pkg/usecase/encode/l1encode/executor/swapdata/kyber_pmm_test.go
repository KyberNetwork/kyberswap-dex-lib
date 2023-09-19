package swapdata

import (
	"math/big"
	"testing"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestPackKyberRFQ(t *testing.T) {
	amount, _ := new(big.Int).SetString("70000000000000000000", 10)

	type args struct {
		chainID      valueobject.ChainID
		encodingSwap types.EncodingSwap
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{
		{
			name: "it should pack kyberRFQ from encodingSwap correctly",
			args: args{
				chainID: valueobject.ChainIDEthereum,
				encodingSwap: types.EncodingSwap{
					Pool:              "kyber_pmm_0xdac17f958d2ee523a2206206994597c13d831ec7_0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
					TokenIn:           "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
					TokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
					SwapAmount:        amount,
					AmountOut:         big.NewInt(35868163),
					LimitReturnAmount: big.NewInt(0),
					Exchange:          "kyber_pmm",
					PoolLength:        2,
					PoolType:          "kyber_pmm",
					PoolExtra:         "",
					Extra: kyberpmm.RFQExtra{
						RFQContractAddress: "0x9f9D5a8F4a6e1F40835cDB040a7a53B35C1b1400",
						Info:               "31261280130335783461303691170",
						Expiry:             1694677391,
						MakerAsset:         "0xdac17f958d2ee523a2206206994597c13d831ec7",
						TakerAsset:         "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
						Maker:              "0xd198885a138d3b1810a7363dd055c540d521c0c2",
						Taker:              "0x9f9d5a8f4a6e1f40835cdb040a7a53b35c1b1400",
						MakerAmount:        "36038515",
						TakerAmount:        "70000000000000000000",
						Signature:          "0x54afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b",
						Recipient:          "0x631Cf2487C312cF0659F9B6B93EC93dfFeFBf83E",
					},
					Flags:         []types.EncodingSwapFlag{},
					CollectAmount: big.NewInt(0),
					Recipient:     "0x631Cf2487C312cF0659F9B6B93EC93dfFeFBf83E",
				},
			},
			want:    common.FromHex("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000009f9d5a8f4a6e1f40835cdb040a7a53b35c1b140000000000000000000000000000000000000000006502b98f99723e93293d27a2000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7000000000000000000000000defa4e8a7bcba345f687a2f1456f5edd9ce97202000000000000000000000000d198885a138d3b1810a7363dd055c540d521c0c20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000225e773000000000000000000000000000000000000000000000003cb71f51fc55800000000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000003cb71f51fc5580000000000000000000000000000631cf2487c312cf0659f9b6b93ec93dffefbf83e000000000000000000000000000000000000000000000000000000000000004154afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b00000000000000000000000000000000000000000000000000000000000000"),
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PackKyberRFQ(tt.args.chainID, tt.args.encodingSwap)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equalf(t, tt.want, got, "PackKyberRFQ(%v, %v)", tt.args.chainID, tt.args.encodingSwap)
		})
	}
}

func TestUnpackKyberRFQ(t *testing.T) {
	info, _ := new(big.Int).SetString("31261280130335783461303691170", 10)
	takingAmount, _ := new(big.Int).SetString("70000000000000000000", 10)

	type args struct {
		encodedSwap []byte
	}
	tests := []struct {
		name    string
		args    args
		want    KyberRFQ
		wantErr error
	}{
		{
			name: "it should unpack kyberRFQ correctly",
			args: args{
				encodedSwap: common.FromHex("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000009f9d5a8f4a6e1f40835cdb040a7a53b35c1b140000000000000000000000000000000000000000006502b98f99723e93293d27a2000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7000000000000000000000000defa4e8a7bcba345f687a2f1456f5edd9ce97202000000000000000000000000d198885a138d3b1810a7363dd055c540d521c0c20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000225e773000000000000000000000000000000000000000000000003cb71f51fc55800000000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000003cb71f51fc5580000000000000000000000000000631cf2487c312cf0659f9b6b93ec93dffefbf83e000000000000000000000000000000000000000000000000000000000000004154afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b00000000000000000000000000000000000000000000000000000000000000"),
			},
			want: KyberRFQ{
				RFQ: common.HexToAddress("0x9f9D5a8F4a6e1F40835cDB040a7a53B35C1b1400"),
				Order: OrderRFQ{
					Info:          info,
					MakerAsset:    common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
					TakerAsset:    common.HexToAddress("0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202"),
					Maker:         common.HexToAddress("0xd198885a138d3b1810a7363dd055c540d521c0c2"),
					AllowedSender: common.Address{},
					MakingAmount:  big.NewInt(36038515),
					TakingAmount:  takingAmount,
				},
				Signature: common.FromHex("0x54afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b"),
				Amount:    takingAmount,
				Target:    common.HexToAddress("0x631Cf2487C312cF0659F9B6B93EC93dfFeFBf83E"),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnpackKyberRFQ(tt.args.encodedSwap)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equalf(t, tt.want, got, "UnpackKyberRFQ(%v)", tt.args.encodedSwap)
		})
	}
}

func Test_buildKyberRFQ(t *testing.T) {
	info, _ := new(big.Int).SetString("31261280130335783461303691170", 10)
	takingAmount, _ := new(big.Int).SetString("70000000000000000000", 10)

	type args struct {
		swap types.EncodingSwap
	}
	tests := []struct {
		name    string
		args    args
		want    KyberRFQ
		wantErr error
	}{
		{
			name: "it should build kyberRFQ correctly",
			args: args{
				swap: types.EncodingSwap{
					Pool:              "kyber_pmm_0xdac17f958d2ee523a2206206994597c13d831ec7_0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
					TokenIn:           "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
					TokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
					SwapAmount:        takingAmount,
					AmountOut:         big.NewInt(35868163),
					LimitReturnAmount: big.NewInt(0),
					Exchange:          "kyber_pmm",
					PoolLength:        2,
					PoolType:          "kyber_pmm",
					PoolExtra:         "",
					Extra: kyberpmm.RFQExtra{
						RFQContractAddress: "0x9f9D5a8F4a6e1F40835cDB040a7a53B35C1b1400",
						Info:               "31261280130335783461303691170",
						Expiry:             1694677391,
						MakerAsset:         "0xdac17f958d2ee523a2206206994597c13d831ec7",
						TakerAsset:         "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
						Maker:              "0xd198885a138d3b1810a7363dd055c540d521c0c2",
						Taker:              "0x9f9d5a8f4a6e1f40835cdb040a7a53b35c1b1400",
						MakerAmount:        "36038515",
						TakerAmount:        "70000000000000000000",
						Signature:          "0x54afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b",
						Recipient:          "0x631Cf2487C312cF0659F9B6B93EC93dfFeFBf83E",
					},
					Flags:         []types.EncodingSwapFlag{},
					CollectAmount: big.NewInt(0),
					Recipient:     "0x631Cf2487C312cF0659F9B6B93EC93dfFeFBf83E",
				},
			},
			want: KyberRFQ{
				RFQ: common.HexToAddress("0x9f9D5a8F4a6e1F40835cDB040a7a53B35C1b1400"),
				Order: OrderRFQ{
					Info:          info,
					MakerAsset:    common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
					TakerAsset:    common.HexToAddress("0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202"),
					Maker:         common.HexToAddress("0xd198885a138d3b1810a7363dd055c540d521c0c2"),
					AllowedSender: common.Address{},
					MakingAmount:  big.NewInt(36038515),
					TakingAmount:  takingAmount,
				},
				Signature: common.FromHex("0x54afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b"),
				Amount:    takingAmount,
				Target:    common.HexToAddress("0x631Cf2487C312cF0659F9B6B93EC93dfFeFBf83E"),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildKyberRFQ(tt.args.swap)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equalf(t, tt.want, got, "buildKyberRFQ(%v)", tt.args.swap)
		})
	}
}

func Test_packKyberRFQ(t *testing.T) {
	info, _ := new(big.Int).SetString("31261280130335783461303691170", 10)
	takingAmount, _ := new(big.Int).SetString("70000000000000000000", 10)

	type args struct {
		kyberRFQ KyberRFQ
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{
		{
			name: "it should pack kyberRFQ correctly",
			args: args{
				kyberRFQ: KyberRFQ{
					RFQ: common.HexToAddress("0x9f9D5a8F4a6e1F40835cDB040a7a53B35C1b1400"),
					Order: OrderRFQ{
						Info:          info,
						MakerAsset:    common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
						TakerAsset:    common.HexToAddress("0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202"),
						Maker:         common.HexToAddress("0xd198885a138d3b1810a7363dd055c540d521c0c2"),
						AllowedSender: common.Address{},
						MakingAmount:  big.NewInt(36038515),
						TakingAmount:  takingAmount,
					},
					Signature: common.FromHex("0x54afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b"),
					Amount:    takingAmount,
					Target:    common.HexToAddress("0x631Cf2487C312cF0659F9B6B93EC93dfFeFBf83E"),
				},
			},
			want:    common.FromHex("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000009f9d5a8f4a6e1f40835cdb040a7a53b35c1b140000000000000000000000000000000000000000006502b98f99723e93293d27a2000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7000000000000000000000000defa4e8a7bcba345f687a2f1456f5edd9ce97202000000000000000000000000d198885a138d3b1810a7363dd055c540d521c0c20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000225e773000000000000000000000000000000000000000000000003cb71f51fc55800000000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000003cb71f51fc5580000000000000000000000000000631cf2487c312cf0659f9b6b93ec93dffefbf83e000000000000000000000000000000000000000000000000000000000000004154afff8646ff13782804c7cc1caa13ca381152cb1dd537dd9bb8797c616fae6655a09a1e3c6c87d60db38e2a68b414ebc4bbe6e4bd1ca694ed13d15e8f4a35871b00000000000000000000000000000000000000000000000000000000000000"),
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := packKyberRFQ(tt.args.kyberRFQ)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equalf(t, tt.want, got, "packKyberRFQ(%v)", tt.args.kyberRFQ)
		})
	}
}
