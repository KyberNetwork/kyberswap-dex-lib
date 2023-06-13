package abi

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestGenMethodID(t *testing.T) {
	testCases := []struct {
		rawName    string
		types      []string
		expectedId string
	}{
		{
			rawName:    "executeUniSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0xd0796174",
		},
		{
			rawName:    "executeStableSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x234c8880",
		},
		{
			rawName:    "executeCurveSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0xcbf622d3",
		},
		{
			rawName:    "executeUniV3ProMMSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x9e8aa935",
		},
		{
			rawName:    "executeBalV2Swap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0xa5d0728f",
		},
		{
			rawName:    "executeDODOSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x8a5da90f",
		},
		{
			rawName:    "executeGMXSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0xad2261ff",
		},
		{
			rawName:    "executeSynthetixSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x9f054463",
		},
		{
			rawName:    "executePSMSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x78c3ad1f",
		},
		{
			rawName:    "executeWrappedstETHSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x583e56d7",
		},
		{
			rawName:    "executeKyberDMMSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x4944c815",
		},
		{
			rawName:    "executeVelodromeSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0xa067e24b",
		},
		{
			rawName:    "executePlatypusSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0xbfe1b858",
		},
		{
			rawName:    "executeMuteSwitchSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0xfe3868f2",
		},
		{
			rawName:    "executeSyncSwap",
			types:      []string{"uint256", "bytes", "uint256"},
			expectedId: "0x22939f03",
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("it should gen method id correctly %d", idx), func(t *testing.T) {
			id := GenMethodID(tc.rawName, tc.types)

			assert.EqualValues(t, common.HexToHash(tc.expectedId), common.BytesToHash(id[:]))
		})
	}
}
