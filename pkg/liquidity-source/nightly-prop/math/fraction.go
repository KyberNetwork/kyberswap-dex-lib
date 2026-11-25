package math

import (
	"github.com/KyberNetwork/msgpack/v5"
	"github.com/holiman/uint256"
)

// maxRaw is FractionMath.MAX_RAW: type(uint104).max, the raw value of a level's upper edge.
var maxRaw = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(one, 104), 1)

// Fraction is the pool's Fraction: the progress through one price level, from 0 at the lower edge
// to maxRaw at the upper edge. Its raw value always fits uint104.
type Fraction struct {
	raw uint256.Int
}

var (
	// Lower is FractionMath.LOWER, a level's lower edge.
	Lower = Fraction{}
	// Upper is FractionMath.UPPER, a level's upper edge.
	Upper = Fraction{raw: *maxRaw}
)

// FractionFromRaw is FractionMath.fromRaw, with SafeCast.toUint104's check: a raw value above
// uint104 is ErrOverflow.
func FractionFromRaw(raw *uint256.Int) (Fraction, error) {
	if raw.Gt(maxRaw) {
		return Fraction{}, ErrOverflow
	}
	return Fraction{raw: *raw}, nil
}

// Raw is FractionMath.toRaw.
func (f Fraction) Raw() *uint256.Int {
	return new(uint256.Int).Set(&f.raw)
}

// IsLower is FractionMath.isLower.
func (f Fraction) IsLower() bool {
	return f.raw.IsZero()
}

// IsUpper is FractionMath.isUpper.
func (f Fraction) IsUpper() bool {
	return f.raw.Eq(maxRaw)
}

// MoveUp is FractionMath.moveUp: it moves toward the upper edge by amount / depth of the remaining
// distance, f' = f + (maxRaw - f) * amount / depth, and rounding applies to the distance moved.
// The caller supplies amount <= depth and a nonzero depth. Otherwise it returns the error of the
// contract's revert.
func (f Fraction) MoveUp(amount, depth *uint256.Int, rounding Rounding) (Fraction, error) {
	distance, err := Sub(maxRaw, &f.raw)
	if err != nil {
		return Fraction{}, err
	}
	moved, err := MulDiv(&distance, amount, depth, rounding)
	if err != nil {
		return Fraction{}, err
	}
	raw, err := Add(&f.raw, &moved)
	if err != nil {
		return Fraction{}, err
	}
	return FractionFromRaw(&raw)
}

// MoveDown is FractionMath.moveDown: it moves toward the lower edge by amount / depth of the
// remaining distance, f' = f - f * amount / depth, and rounding applies to the distance moved.
// The caller supplies amount <= depth and a nonzero depth. Otherwise it returns the error of the
// contract's revert.
func (f Fraction) MoveDown(amount, depth *uint256.Int, rounding Rounding) (Fraction, error) {
	moved, err := MulDiv(&f.raw, amount, depth, rounding)
	if err != nil {
		return Fraction{}, err
	}
	raw, err := Sub(&f.raw, &moved)
	if err != nil {
		return Fraction{}, err
	}
	return FractionFromRaw(&raw)
}

// MarshalJSON encodes the raw value as a decimal string, as uint256.Int does.
func (f Fraction) MarshalJSON() ([]byte, error) {
	return f.raw.MarshalJSON()
}

// UnmarshalJSON decodes a raw value with FractionFromRaw's check.
func (f *Fraction) UnmarshalJSON(input []byte) error {
	var raw uint256.Int
	if err := raw.UnmarshalJSON(input); err != nil {
		return err
	}
	return f.set(&raw)
}

// EncodeMsgpack encodes the raw value as msgpack encodes a uint256.Int. Without it, msgpack skips
// the unexported raw value, and a decoded fraction is the lower edge.
func (f Fraction) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.Encode(&f.raw)
}

// DecodeMsgpack decodes a raw value with FractionFromRaw's check.
func (f *Fraction) DecodeMsgpack(dec *msgpack.Decoder) error {
	var raw uint256.Int
	if err := dec.Decode(&raw); err != nil {
		return err
	}
	return f.set(&raw)
}

func (f *Fraction) set(raw *uint256.Int) error {
	decoded, err := FractionFromRaw(raw)
	if err != nil {
		return err
	}
	*f = decoded
	return nil
}
