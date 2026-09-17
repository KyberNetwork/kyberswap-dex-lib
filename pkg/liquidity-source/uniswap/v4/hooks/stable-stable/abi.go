package stablestable

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var (
	// stableStableHookABI covers the legacy hook deployment (HookAddresses[0]), whose
	// feeConfig()/FeeConfigUpdated still carry an explicit on-chain logK.
	stableStableHookABI = lo.Must(abi.JSON(bytes.NewReader(stableStableHookABIJson)))
	// stableStableHookV2ABI covers newer hook deployments, whose feeConfig() dropped
	// logK; it's now derived off-chain from k (see deriveLogK).
	stableStableHookV2ABI = lo.Must(abi.JSON(bytes.NewReader(stableStableHookV2ABIJson)))
)
