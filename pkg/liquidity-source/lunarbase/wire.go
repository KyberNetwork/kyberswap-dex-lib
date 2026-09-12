package lunarbase

import (
	"fmt"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/KyberNetwork/msgpack/v5/msgpcode"
)

// EncodeMsgpack uses named fields for the complete simulator. Older Kyber
// decoders accept maps and skip unknown names even with ForceAsArray enabled.
// Keep this method on PoolSimulator: placing it on embedded Extra would promote
// it and accidentally serialize only Extra instead of the complete simulator.
func (s *PoolSimulator) EncodeMsgpack(enc *msgpack.Encoder) error {
	if s == nil {
		return enc.EncodeNil()
	}
	if s.Extra == nil || s.StaticExtra == nil {
		return fmt.Errorf("lunarbase wire: missing simulator metadata")
	}
	version := uint8(2)
	if s.SnapshotComplete {
		version = 3
	}
	type wireField struct {
		name  string
		value any
	}
	fields := []wireField{
		{"_lunarbaseWire", version},
		{"Info", s.Info}, {"reserves", s.reserves}, {"chainID", s.chainID},
		{"SqrtPriceX96", s.SqrtPriceX96}, {"FeeAskX24", s.FeeAskX24}, {"FeeBidX24", s.FeeBidX24},
		{"LatestUpdateBlock", s.LatestUpdateBlock}, {"Paused", s.Paused}, {"BlockDelay", s.BlockDelay},
		{"ConcentrationK", s.ConcentrationK}, {"MaxPunishmentX24", s.MaxPunishmentX24}, {"HasNative", s.HasNative},
		{"BlockHash", s.BlockHash}, {"ConcentrationModel", s.ConcentrationModel},
		{"requiresRPCRefresh", s.requiresRPCRefresh},
	}
	if s.SnapshotComplete {
		fields = append(fields, wireField{"SnapshotComplete", true})
	}
	if err := enc.EncodeMapLen(len(fields)); err != nil {
		return err
	}
	for _, field := range fields {
		if err := enc.EncodeString(field.name); err != nil {
			return err
		}
		if err := enc.Encode(field.value); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsgpack accepts the upstream 12-field array and named v2/v3 maps. Old
// frames cannot supply a canonical hash; preserve that absence rather than
// inventing one. A legacy K=0/max=0 frame cannot identify its pricing model.
func (s *PoolSimulator) DecodeMsgpack(dec *msgpack.Decoder) error {
	code, err := dec.PeekCode()
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		return fmt.Errorf("lunarbase wire: nil simulator")
	}
	var next PoolSimulator
	next.Extra = &Extra{}
	next.StaticExtra = &StaticExtra{}
	legacy := false
	if msgpcode.IsFixedArray(code) || code == msgpcode.Array16 || code == msgpcode.Array32 {
		n, err := dec.DecodeArrayLen()
		if err != nil {
			return err
		}
		if n != 12 {
			return fmt.Errorf("lunarbase wire: unsupported legacy array length %d", n)
		}
		fields := []any{&next.Info, &next.reserves, &next.chainID, &next.SqrtPriceX96, &next.FeeAskX24, &next.FeeBidX24, &next.LatestUpdateBlock, &next.Paused, &next.BlockDelay, &next.ConcentrationK, &next.MaxPunishmentX24, &next.HasNative}
		for _, field := range fields {
			if err := dec.Decode(field); err != nil {
				return err
			}
		}
		legacy = true
	} else {
		n, err := dec.DecodeMapLen()
		if err != nil {
			return err
		}
		seen := make(map[string]bool, n)
		var version uint64
		for i := 0; i < n; i++ {
			name, err := dec.DecodeString()
			if err != nil {
				return err
			}
			if seen[name] {
				return fmt.Errorf("lunarbase wire: duplicate field %s", name)
			}
			seen[name] = true
			var field any
			switch name {
			case "_lunarbaseWire":
				field = &version
			case "Info":
				field = &next.Info
			case "reserves":
				field = &next.reserves
			case "chainID":
				field = &next.chainID
			case "SqrtPriceX96":
				field = &next.SqrtPriceX96
			case "FeeAskX24":
				field = &next.FeeAskX24
			case "FeeBidX24":
				field = &next.FeeBidX24
			case "LatestUpdateBlock":
				field = &next.LatestUpdateBlock
			case "Paused":
				field = &next.Paused
			case "BlockDelay":
				field = &next.BlockDelay
			case "ConcentrationK":
				field = &next.ConcentrationK
			case "MaxPunishmentX24":
				field = &next.MaxPunishmentX24
			case "HasNative":
				field = &next.HasNative
			case "BlockHash":
				field = &next.BlockHash
			case "ConcentrationModel":
				field = &next.ConcentrationModel
			case "SnapshotComplete":
				field = &next.SnapshotComplete
			case "requiresRPCRefresh":
				field = &next.requiresRPCRefresh
			default:
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}
			if err := dec.Decode(field); err != nil {
				return err
			}
		}
		if version != 0 && version != 2 && version != 3 {
			return fmt.Errorf("lunarbase wire: unsupported version %d", version)
		}
		for _, name := range []string{"Info", "reserves", "chainID", "SqrtPriceX96", "FeeAskX24", "FeeBidX24", "LatestUpdateBlock", "Paused", "BlockDelay", "ConcentrationK", "MaxPunishmentX24", "HasNative"} {
			if !seen[name] {
				return fmt.Errorf("lunarbase wire: missing field %s", name)
			}
		}
		if version >= 2 && (!seen["BlockHash"] || !seen["ConcentrationModel"] || !seen["requiresRPCRefresh"]) {
			return fmt.Errorf("lunarbase wire: missing version 2 metadata")
		}
		if version == 3 && !seen["SnapshotComplete"] {
			return fmt.Errorf("lunarbase wire: missing version 3 metadata")
		}
		legacy = !seen["ConcentrationModel"]
	}
	if len(next.reserves) != 2 || next.reserves[0] == nil || next.reserves[1] == nil || next.SqrtPriceX96 == nil || len(next.Info.Tokens) != 2 || len(next.Info.Reserves) != 2 {
		return fmt.Errorf("lunarbase wire: incomplete simulator state")
	}
	if legacy {
		if next.ConcentrationK == 0 && next.MaxPunishmentX24 == 0 {
			next.requiresRPCRefresh = true
		}
		next.ConcentrationModel = next.ConcentrationK > 0
	}
	*s = next
	return nil
}
