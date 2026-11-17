package ticks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTicksBasedStateBlockOf(t *testing.T) {
	assert.Equal(t, uint64(95), TicksBasedStateBlockOf(`{"liquidity":1,"sB":95}`))
	assert.Zero(t, TicksBasedStateBlockOf(`{"liquidity":1}`), "an Extra from before the field")
	assert.Zero(t, TicksBasedStateBlockOf(``), "no Extra")
	assert.Zero(t, TicksBasedStateBlockOf(`not json`))
}

func TestTicksBasedStateBlockForMeta(t *testing.T) {
	assert.Equal(t, TicksBasedStateBlock{StateBlock: 120}, TicksBasedStateBlock{StateBlock: 120}.ForMeta(100), "R above H")
	assert.Equal(t, TicksBasedStateBlock{StateBlock: 100}, TicksBasedStateBlock{StateBlock: 90}.ForMeta(100), "never below H")
	assert.Equal(t, TicksBasedStateBlock{}, TicksBasedStateBlock{}.ForMeta(100), "unknown stays unknown")
}
