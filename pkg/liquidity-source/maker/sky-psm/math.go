package skypsm

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/spark"
)

const (
	OneToOne  = "convertOneToOne"
	ToSUSDS   = "convertToSUsds"
	FromSUSDS = "convertFromSUsds"
)

var (
	ErrInvalidArgs = errors.New("invalid args")
)

var FuncMap = map[string]func(amount *uint256.Int, args ...*uint256.Int) (*uint256.Int, error){
	OneToOne: func(amount *uint256.Int, args ...*uint256.Int) (*uint256.Int, error) {
		if len(args) != 2 {
			return nil, ErrInvalidArgs
		}
		// amount * convertAssetPrecision / assetPrecision
		var r uint256.Int
		r.Set(amount).Mul(&r, args[0]).Div(&r, args[1])
		return &r, nil
	},
	ToSUSDS: func(amount *uint256.Int, args ...*uint256.Int) (*uint256.Int, error) {
		if len(args) != 3 {
			return nil, ErrInvalidArgs
		}
		// amount * 1e27 / rate * susdsPrecision / assetPrecision
		var r uint256.Int
		r.Set(amount).Mul(&r, spark.RAY).
			Div(&r, args[0]).
			Mul(&r, args[1]).
			Div(&r, args[2])
		return &r, nil
	},
	FromSUSDS: func(amount *uint256.Int, args ...*uint256.Int) (*uint256.Int, error) {
		if len(args) != 3 {
			return nil, ErrInvalidArgs
		}
		// amount * rate  / 1e27 * assetPrecision / susdsPrecision
		var r uint256.Int
		r.Set(amount).Mul(&r, args[0]).
			Div(&r, spark.RAY).
			Mul(&r, args[2]).
			Div(&r, args[3])
		return &r, nil
	},
}
