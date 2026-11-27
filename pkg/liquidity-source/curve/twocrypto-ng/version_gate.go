package twocryptong

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

// StoredVersion is the version() a stored pool state recorded, or "" (no state, or state stored before the
// trackers read it).
func StoredVersion(extra string) string {
	var e struct{ Version string }
	if json.Unmarshal([]byte(extra), &e) != nil {
		return ""
	}
	return e.Version
}

// AddVersionCalls adds the version-specific reads of a refresh. version() is immutable, so it is read, together
// with the views of every version, only while storedExtra has no version (a new pool, or state stored before
// the trackers read it). Once the version is stored, a refresh reads only that version's views and v21d keeps
// the stored version for Apply: v2.1.0d and v3.0.0 read last_timestamp and the donation views (both
// simulators use them), v3.0.0 also its profit, fee-split and policy views, and any other version none.
//
// policy() exists on v3.0.0 pools only and allowed_extra_profit() on every version but v3.0.0, so a pool of a
// stored version reads the one it has.
func AddVersionCalls(add func(*ethrpc.Call, []any), poolAddress, storedExtra string, v21d *V21dRPC, v3 *V3RPC,
	policy *common.Address, allowedExtraProfit **big.Int) {
	addPolicy := func() {
		add(&ethrpc.Call{ABI: CurveTwocryptoNGABI, Target: poolAddress, Method: poolMethodPolicy}, []any{policy})
	}
	addAllowedExtraProfit := func() {
		add(&ethrpc.Call{ABI: CurveTwocryptoNGABI, Target: poolAddress, Method: poolMethodAllowedExtraProfit},
			[]any{allowedExtraProfit})
	}
	switch version := StoredVersion(storedExtra); version {
	case "":
		addAllowedExtraProfit()
		addPolicy()
		addV21dCalls(add, poolAddress, v21d, true)
		AddV3Calls(add, poolAddress, PreviousPolicy(storedExtra), v3)
	case VersionV21d:
		v21d.Version = version
		addAllowedExtraProfit()
		addV21dCalls(add, poolAddress, v21d, false)
	case VersionV3:
		v21d.Version = version
		addPolicy()
		addV21dCalls(add, poolAddress, v21d, false)
		AddV3Calls(add, poolAddress, PreviousPolicy(storedExtra), v3)
	default:
		v21d.Version = version
		addAllowedExtraProfit()
	}
}
