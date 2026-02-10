package pools

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
)

type (
	swappedEvent struct {
		sqrtRatioAfter *uint256.Int
		tickAfter      int32
		liquidityAfter *uint256.Int
	}
	positionUpdatedEvent struct {
		liquidityDelta *int256.Int
		lower          int32
		upper          int32
	}
)

func parseSwappedEventIfMatching(data []byte, poolKey IPoolKey) (*swappedEvent, error) {
	if len(data) < 116 {
		return nil, nil
	}

	poolId := data[20:52]
	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return nil, fmt.Errorf("computing expected pool id: %w", err)
	}
	if !bytes.Equal(poolId, expectedPoolId) {
		return nil, nil
	}

	return &swappedEvent{
		sqrtRatioAfter: math.FloatSqrtRatioToFixed(new(uint256.Int).SetBytes(data[84:96])),
		tickAfter:      int32(binary.BigEndian.Uint32(data[96:100])),
		liquidityAfter: new(uint256.Int).SetBytes(data[100:116]),
	}, nil
}

func parsePositionUpdatedEventIfMatching(data []byte, poolKey IPoolKey) (*positionUpdatedEvent, error) {
	values, err := abis.PositionUpdatedEvent.Inputs.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("unpacking event data: %w", err)
	}

	poolId, ok := values[1].([32]byte)
	if !ok {
		return nil, errors.New("failed to parse poolId")
	}

	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return nil, fmt.Errorf("computing expected pool id: %w", err)
	}

	if !bytes.Equal(expectedPoolId, poolId[:]) {
		return nil, nil
	}

	liquidityDelta, ok := values[3].(*big.Int)
	if !ok {
		return nil, errors.New("failed to parse liquidityDelta")
	}

	if liquidityDelta.Sign() == 0 {
		return nil, nil
	}

	params, ok := values[2].([32]byte)
	if !ok {
		return nil, errors.New("failed to parse positionId")
	}

	return &positionUpdatedEvent{
		liquidityDelta: int256.MustFromBig(liquidityDelta),
		lower:          int32(binary.BigEndian.Uint32(params[24:28])),
		upper:          int32(binary.BigEndian.Uint32(params[28:32])),
	}, nil
}
