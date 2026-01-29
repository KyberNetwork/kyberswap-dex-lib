package someswapv2

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestConfig_UnmarshalProvidedJSON(t *testing.T) {
	t.Parallel()

	const input = `{
  "factory": "0xF4B30295EA24938d9705E30F88e144140422BAa3",
  "router": "0x7153f466a8DE3ee8bb7196F8b4c615aD06F4b175",
  "quoter": "0xb41079B0681F4C6f554a50c56d931fd6D13E62eb",
  "lpFeeManager": "0x440374B4a44987f070AAF24Ff5a237b99681D44B",
  "liquidityLocker": "0x8f749e223E9A13FB140D061740b16f57D85A1DD5",
  "coreModule": "0x361F68Dd5a3C5b2e3ababb3191a740B83d345dB2",
  "permissionsRegistry": "0x320c4D1c74279D5A5e8D38C929C5798d48ccF75a"
}`

	var cfg Config
	require.NoError(t, json.Unmarshal([]byte(input), &cfg))

	t.Logf("someswapv2 config: factory=%s router=%s quoter=%s lpFeeManager=%s liquidityLocker=%s coreModule=%s permissionsRegistry=%s",
		cfg.Factory, cfg.Router, cfg.Quoter, cfg.LPFeeManager, cfg.LiquidityLocker, cfg.CoreModule, cfg.PermissionsRegistry)

	require.Equal(t, "0xF4B30295EA24938d9705E30F88e144140422BAa3", cfg.Factory)
	require.Equal(t, "0x7153f466a8DE3ee8bb7196F8b4c615aD06F4b175", cfg.Router)
	require.Equal(t, "0xb41079B0681F4C6f554a50c56d931fd6D13E62eb", cfg.Quoter)
	require.Equal(t, "0x440374B4a44987f070AAF24Ff5a237b99681D44B", cfg.LPFeeManager)
	require.Equal(t, "0x8f749e223E9A13FB140D061740b16f57D85A1DD5", cfg.LiquidityLocker)
	require.Equal(t, "0x361F68Dd5a3C5b2e3ababb3191a740B83d345dB2", cfg.CoreModule)
	require.Equal(t, "0x320c4D1c74279D5A5e8D38C929C5798d48ccF75a", cfg.PermissionsRegistry)
}

func TestFactoryABI_ParsesAndHasExpectedEntries(t *testing.T) {
	t.Parallel()

	require.Equal(t, "someswap-v2", DexType)

	require.Contains(t, factoryABI.Methods, factoryMethodAllPairsLength)
	require.Contains(t, factoryABI.Events, factoryEventPairCreated)

	t.Logf("factory method %s selector: %s", factoryMethodAllPairsLength, hexutil.Encode(factoryABI.Methods[factoryMethodAllPairsLength].ID))
	t.Logf("factory event %s topic0: %s", factoryEventPairCreated, factoryABI.Events[factoryEventPairCreated].ID.Hex())
}

