package aevm_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/core/aevm"
)

func TestPackERC20ApproveCall(t *testing.T) {
	for _, param := range []struct {
		addr     common.Address
		amountIn *big.Int
	}{
		{
			common.HexToAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
			big.NewInt(0),
		},
		{
			common.HexToAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
			new(big.Int).SetUint64(math.MaxUint64),
		},
		{
			common.HexToAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
			new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), // 2^256 - 1
		},
	} {
		expected, err := abis.ERC20.Pack("approve", param.addr, param.amountIn)
		require.NoError(t, err)

		actual, err := aevm.PackERC20ApproveCall(param.addr, param.amountIn)
		require.NoError(t, err)

		require.Equal(t, expected, actual)
	}
}

func TestPackERC20TransferCall(t *testing.T) {
	for _, param := range []struct {
		addr   common.Address
		amount *big.Int
	}{
		{
			common.HexToAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
			big.NewInt(0),
		},
		{
			common.HexToAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
			new(big.Int).SetUint64(math.MaxUint64),
		},
		{
			common.HexToAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
			new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), // 2^256 - 1
		},
	} {
		expected, err := abis.ERC20.Pack("transfer", param.addr, param.amount)
		require.NoError(t, err)

		actual, err := aevm.PackERC20TransferCall(param.addr, param.amount)
		require.NoError(t, err)

		require.Equal(t, expected, actual)
	}
}
