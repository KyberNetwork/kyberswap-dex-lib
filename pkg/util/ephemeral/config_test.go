package ephemeral

import (
	"encoding/json"
	"testing"

	gojson "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLensConfigKeysSitBesideTheDexs: a dex's Config embeds LensConfig, and its lens keys are read
// from beside the dex's own, by encoding/json and by goccy/go-json alike.
func TestLensConfigKeysSitBesideTheDexs(t *testing.T) {
	type dexConfig struct {
		DexID string `json:"dexID"`
		LensConfig
	}
	raw := []byte(`{"dexID":"d","useLens":true,"lensGas":5,"lensGuard":{"cooldownSecs":3}}`)
	for name, unmarshal := range map[string]func([]byte, any) error{
		"encoding/json": json.Unmarshal, "goccy/go-json": gojson.Unmarshal,
	} {
		var cfg dexConfig
		require.NoError(t, unmarshal(raw, &cfg), name)
		assert.Equal(t, "d", cfg.DexID, name)
		assert.True(t, cfg.UseLens, name)
		assert.EqualValues(t, 5, cfg.LensGas, name)
		assert.EqualValues(t, 3, cfg.LensGuard.CooldownSecs, name)
	}
}
