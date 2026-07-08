package pamm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
)

const (
	testPriorityUpdateRegistry = "0xda7afeed01fe625cf15d187a19f94b45f00b8c5f"
	testPriorityUpdateTarget   = "0xfE3d12b21d2602868223E83149bdBbFB5D11e185"
)

func TestPriorityUpdateLaneSlot(t *testing.T) {
	slot := priorityUpdateLaneSlot(common.HexToAddress(testPriorityUpdateTarget), 0)

	assert.Equal(
		t,
		common.HexToHash("0x3d9784f10b231c263263dafaa16836e1b955778891780d7013ea3fd12ccd046f"),
		slot,
	)
}

func TestExtractPammBlockTimestamp(t *testing.T) {
	cfg := &Config{
		LensAddress:          testPriorityUpdateTarget,
		PriorityUpdateRegistry: testPriorityUpdateRegistry,
	}
	slot := priorityUpdateLaneSlot(common.HexToAddress(cfg.LensAddress), priorityUpdateLaneIndex)
	overrides := map[common.Address]gethclient.OverrideAccount{
		common.HexToAddress(cfg.PriorityUpdateRegistry): {
			StateDiff: map[common.Hash]common.Hash{
				slot: common.HexToHash("0x6a229cf302000000000000000000000000000000000000000000000000000000"),
			},
		},
	}

	assert.Equal(t, uint64(1780653299), extractPammBlockTimestamp(overrides, cfg))
}

func TestExtractPammBlockTimestampMissingPUR(t *testing.T) {
	cfg := &Config{
		LensAddress:          testPriorityUpdateTarget,
		PriorityUpdateRegistry: testPriorityUpdateRegistry,
	}

	assert.Zero(t, extractPammBlockTimestamp(nil, cfg))
	assert.Zero(t, extractPammBlockTimestamp(map[common.Address]gethclient.OverrideAccount{}, cfg))
}
