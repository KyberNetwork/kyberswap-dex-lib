package everlongflamm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// goccyMaxCaseInsensitiveFields is goccy/go-json's allowOptimizeMaxFieldLen (internal/decoder/struct.go): a struct
// with more JSON fields than this is decoded by exact key lookup only, which drops the "DexID" the pool-service
// factories inject (pkg/source/pool/{list,tracker}/factory.go) and every key of the repo's dominant spelling.
const goccyMaxCaseInsensitiveFields = 16

// jsonFieldCount is the number of keys the decoder builds for a struct type.
func jsonFieldCount(t reflect.Type) int {
	n := 0
	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("json"); tag != "-" {
			n++
		}
	}
	return n
}

// TestConfigFieldCount keeps Config within the case-insensitive decoding limit, which every key but the dex id
// depends on: past it only a key's exact tag is matched, so the README's Go field-name spelling and the repo's
// `chainID` / `chainId` variants would be dropped in silence. A new option goes into Gas or another sub-struct.
func TestConfigFieldCount(t *testing.T) {
	t.Parallel()
	require.LessOrEqual(t, jsonFieldCount(reflect.TypeOf(Config{})), goccyMaxCaseInsensitiveFields)

	// The limit itself, on control structs of the same shape as Config's head: at the limit the injected key lands
	// whatever the tag's case, past it only the tag Config uses -- the exact key the factories inject -- still does.
	field := func(i int) reflect.StructField {
		return reflect.StructField{Name: "F" + string(rune('A'+i)), Type: reflect.TypeOf(""),
			Tag: reflect.StructTag(`json:"f` + string(rune('a'+i)) + `"`)}
	}
	dexIDTag, _ := reflect.TypeOf(Config{}).FieldByName("DexID")
	for _, c := range []struct {
		fields int
		tag    reflect.StructTag
		want   string
	}{
		{goccyMaxCaseInsensitiveFields, `json:"dexId"`, DexType},
		{goccyMaxCaseInsensitiveFields + 1, `json:"dexId"`, ""},
		{goccyMaxCaseInsensitiveFields, dexIDTag.Tag, DexType},
		{goccyMaxCaseInsensitiveFields + 1, dexIDTag.Tag, DexType},
	} {
		fields := []reflect.StructField{{Name: "DexID", Type: reflect.TypeOf(""), Tag: c.tag}}
		for i := 1; i < c.fields; i++ {
			fields = append(fields, field(i))
		}
		out := reflect.New(reflect.StructOf(fields))
		require.NoError(t, pool.PropertiesToStruct(map[string]any{"DexID": DexType}, out.Interface()))
		require.Equal(t, c.want, out.Elem().Field(0).String(), "%d fields, %s", c.fields, c.tag)
	}
}

// TestConfigThroughFactories decodes Config the way pool-service does: through the registered lister and tracker
// factories, which inject properties["DexID"] and decode with pool.PropertiesToStruct. Every documented key must
// land, in the exact spelling of its tag and in the Go field-name spelling the README's table uses.
func TestConfigThroughFactories(t *testing.T) {
	t.Parallel()
	factory := c104.Factory.Hex()
	for _, c := range []struct {
		name  string
		props map[string]any
		want  Config
	}{
		{"minimal", map[string]any{"chainID": 8453, "factory": factory},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory}},
		{"tag-spelling", map[string]any{"chainID": 8453, "factory": factory, "leverRouting": true,
			"leverMinEdgeBps": 25, "priceBandMarginBps": 35, "maxSnapshotAgeSec": 120, "quoteDonatedVenues": true,
			"gas": map[string]any{"swapSell": 1_400_000, "leverDown": 4_200_000}},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory, LeverRouting: true,
				LeverMinEdgeBps: ptrU64(25), PriceBandMarginBps: ptrU64(35), MaxSnapshotAgeSec: ptrU64(120),
				QuoteDonatedVenues: true, Gas: GasConfig{SwapSell: 1_400_000, LeverDown: 4_200_000}}},
		{"field-name-spelling", map[string]any{"ChainID": 8453, "Factory": factory, "LeverRouting": true,
			"LeverMinEdgeBps": 0, "PriceBandMarginBps": 0, "PriceAgeMarginSec": 90, "SpreadAgeMarginSec": 15,
			"DebtDriftSec": 0, "MaxSnapshotAgeSec": 300, "Pools": []string{c104.Pool.Hex()},
			"Gas": map[string]any{"SwapSellPass": 7}},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory, LeverRouting: true,
				LeverMinEdgeBps: ptrU64(0), PriceBandMarginBps: ptrU64(0), PriceAgeMarginSec: ptrU64(90),
				SpreadAgeMarginSec: ptrU64(15), DebtDriftSec: ptrU64(0), MaxSnapshotAgeSec: ptrU64(300),
				Pools: []string{c104.Pool.Hex()}, Gas: GasConfig{SwapSellPass: 7}}},
		{"chainid-lowercase-d", map[string]any{"chainId": 8453, "factory": factory},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory}},
		// A configuration that spells the dex id itself does not displace the one the factory injects.
		{"dexid-spelled", map[string]any{"chainID": 8453, "factory": factory, "dexId": DexType},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			lister, err := poollist.Factory(DexType)(DexType, poollist.FactoryParams{Exchange: DexType,
				Properties: c.props})
			require.NoError(t, err)
			require.Equal(t, c.want, *lister.(*PoolsListUpdater).config)

			tracker, err := pooltrack.Factory(DexType)(DexType, pooltrack.FactoryParams{Exchange: DexType,
				Properties: c.props})
			require.NoError(t, err)
			require.Equal(t, c.want, *tracker.(*PoolTracker).config)
		})
	}
}

func ptrU64(v uint64) *uint64 { return &v }
