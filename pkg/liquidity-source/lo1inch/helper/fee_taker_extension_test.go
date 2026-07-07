//nolint:testpackage
package helper

import (
	"math/big"
	"testing"

	util "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestCalculateTakingAmount(t *testing.T) {
	testCase := []struct {
		name         string
		feeTakerEx   FeeTakerExtension
		takerAddress common.Address
		takingAmount *big.Int
		expected     *big.Int
	}{
		{
			name: "normal case, taker is in white list",
			feeTakerEx: FeeTakerExtension{
				Address: common.HexToAddress("0x1"),
				Fees: Fees{
					Resolver: ResolverFee{
						Receiver:          common.HexToAddress("0x2"),
						Fee:               1,
						WhitelistDiscount: 0,
					},
				},
				feeCalculator: FeeCalculator{
					fees: Fees{
						Resolver: ResolverFee{
							Receiver:          common.HexToAddress("0x2"),
							Fee:               100,
							WhitelistDiscount: 0,
						},
					},
					whitelist: Whitelist{
						Addresses: []util.AddressHalf{util.HalfAddressFromAddress(common.HexToAddress("0x3"))},
					},
				},
			},
			takerAddress: common.HexToAddress("0x3"),
			takingAmount: big.NewInt(100_000_000),
			expected:     big.NewInt(101_000_000),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			takingAmount := tc.feeTakerEx.GetTakingAmount(tc.takerAddress, tc.takingAmount)
			assert.True(t, tc.expected.Cmp(takingAmount) == 0)
		})
	}
}

func TestCalculateResolverFee(t *testing.T) {
	testCase := []struct {
		name         string
		feeTakerEx   FeeTakerExtension
		takerAddress common.Address
		resolverFee  *big.Int
		expected     *big.Int
	}{
		{
			name: "normal case, taker is in white list",
			feeTakerEx: FeeTakerExtension{
				Address: common.HexToAddress("0x1"),
				Fees: Fees{
					Resolver: ResolverFee{
						Receiver:          common.HexToAddress("0x2"),
						Fee:               1,
						WhitelistDiscount: 0,
					},
				},
				feeCalculator: FeeCalculator{
					fees: Fees{
						Resolver: ResolverFee{
							Receiver: common.HexToAddress("0x2"),
							Fee:      100,
						},
					},
					whitelist: Whitelist{
						Addresses: []util.AddressHalf{util.HalfAddressFromAddress(common.HexToAddress("0x3"))},
					},
				},
			},
			takerAddress: common.HexToAddress("0x3"),
			resolverFee:  big.NewInt(100_000_000),
			expected:     big.NewInt(1_000_000),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			takingAmount := tc.feeTakerEx.GetResolverFee(tc.takerAddress, tc.resolverFee)
			assert.True(t, tc.expected.Cmp(takingAmount) == 0)
		})
	}
}

func TestCalculateIntegratorFee(t *testing.T) {
	testCase := []struct {
		name          string
		feeTakerEx    FeeTakerExtension
		takerAddress  common.Address
		integratorFee *big.Int
		expected      *big.Int
	}{
		{
			name: "normal case, taker is in white list",
			feeTakerEx: FeeTakerExtension{
				Address: common.HexToAddress("0x1"),
				Fees: Fees{
					Integrator: IntegratorFee{
						Integrator: common.HexToAddress("0x2"),
						Protocol:   common.HexToAddress("0x3"),
						Fee:        1,
						Share:      100,
					},
				},
				feeCalculator: FeeCalculator{
					fees: Fees{
						Integrator: IntegratorFee{
							Integrator: common.HexToAddress("0x2"),
							Protocol:   common.HexToAddress("0x3"),
							Fee:        500,
							Share:      1000,
						},
					},
					whitelist: Whitelist{
						Addresses: []util.AddressHalf{util.HalfAddressFromAddress(common.HexToAddress("0x4"))},
					},
				},
			},
			takerAddress:  common.HexToAddress("0x4"),
			integratorFee: big.NewInt(100_000_000),
			expected:      big.NewInt(500_000),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			integrator := tc.feeTakerEx.GetIntegratorFee(tc.takerAddress, tc.integratorFee)
			assert.True(t, tc.expected.Cmp(integrator) == 0)
		})
	}
}

func TestGetProtocolFee(t *testing.T) {
	testCase := []struct {
		name         string
		feeTakerEx   FeeTakerExtension
		takerAddress common.Address
		protocolFee  *big.Int
		expected     *big.Int
	}{
		{
			name: "normal case, taker is in white list",
			feeTakerEx: FeeTakerExtension{
				Address: common.HexToAddress("0x1"),
				Fees: Fees{
					Integrator: IntegratorFee{
						Integrator: common.HexToAddress("0x2"),
						Protocol:   common.HexToAddress("0x3"),
						Fee:        1,
						Share:      100,
					},
				},
				feeCalculator: FeeCalculator{
					fees: Fees{
						Resolver: ResolverFee{
							Receiver: common.HexToAddress("0x2"),
							Fee:      100,
						},
						Integrator: IntegratorFee{
							Integrator: common.HexToAddress("0x2"),
							Protocol:   common.HexToAddress("0x3"),
							Fee:        500,
							Share:      1000,
						},
					},
					whitelist: Whitelist{
						Addresses: []util.AddressHalf{util.HalfAddressFromAddress(common.HexToAddress("0x4"))},
					},
				},
			},
			takerAddress: common.HexToAddress("0x4"),
			protocolFee:  big.NewInt(100_000_000),
			expected:     big.NewInt(500_000 + 1_000_000),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			protocolFee := tc.feeTakerEx.GetProtocolFee(tc.takerAddress, tc.protocolFee)
			assert.True(t, tc.expected.Cmp(protocolFee) == 0)
		})
	}
}

// order: 0x1bf7bba4b4140c5a02431c5854c5170f2cc15599c7dade20aa099ca0037d1cf7
// tx_hash: 0x1c988bf44efa704617494fbd0dfca5e1049801af1df6aa8809896a526724431f
func TestNewFeeTakerExtension(t *testing.T) {
	extension, err := DecodeExtension("0x000000d400000072000000720000007200000072000000390000000000000000c0dfdb9e7a392c3dbbe7c6fbe8fbc1789c9fe05e00000001f43203b09498030ae3416b66dc74db31d09524fa87b1f7d18bd45f0b94f54a968fc0dfdb9e7a392c3dbbe7c6fbe8fbc1789c9fe05e00000001f43203b09498030ae3416b66dc74db31d09524fa87b1f7d18bd45f0b94f54a968fc0dfdb9e7a392c3dbbe7c6fbe8fbc1789c9fe05e00000000000000000000000000000000000000000090cbe4bdd538d6e9b379bff5fe72c3d67a521de500000001f43203b09498030ae3416b66dc74db31d09524fa87b1f7d18bd45f0b94f54a968f")
	assert.NoError(t, err)
	feeTakerExtension, err := NewFeeTakerFromExtension(extension)
	assert.NoError(t, err)
	resolver := common.HexToAddress("0xBEEf02961503351625926Ea9a11AE13B29F5c555")
	originTakingAmount := big.NewInt(24565559816)
	assert.Equal(t, new(big.Int).SetInt64(24688387616), feeTakerExtension.GetTakingAmount(resolver, originTakingAmount), "taking amount should be 100000001")
}

func TestDecodeExtension(t *testing.T) {
	extension, err := DecodeExtension("0x000000e800000072000000720000007200000072000000390000000000000000c0dfdb9e7a392c3dbbe7c6fbe8fbc1789c9fe05e00000001f43203b09498030ae3416b66dc74db31d09524fa87b1f7d18bd45f0b94f54a968fc0dfdb9e7a392c3dbbe7c6fbe8fbc1789c9fe05e00000001f43203b09498030ae3416b66dc74db31d09524fa87b1f7d18bd45f0b94f54a968fc0dfdb9e7a392c3dbbe7c6fbe8fbc1789c9fe05e01000000000000000000000000000000000000000090cbe4bdd538d6e9b379bff5fe72c3d67a521de509cc0a79dfef324587c6c9bc814d3a5a072e71de00000001f43203b09498030ae3416b66dc74db31d09524fa87b1f7d18bd45f0b94f54a968f")
	assert.NoError(t, err)
	feeTakerExtension, err := NewFeeTakerFromExtension(extension)
	assert.NoError(t, err)
	t.Log(feeTakerExtension)
}
