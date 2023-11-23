package aevm

import "github.com/KyberNetwork/aevm/common"

// AVAXNormalizeStateKey reference: https://github.com/ava-labs/coreth/blob/e3f354ececa0c91e7c85ab592faf03d0e9c62d33/core/state/state_object.go#L560
func AVAXNormalizeStateKey(key *common.Hash) {
	key[0] &= 0xfe
}
