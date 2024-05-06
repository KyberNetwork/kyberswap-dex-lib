//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PriceFeedEnum

package madmex

import "fmt"

// PriceFeedEnum is the sum type of *FastPriceFeedV1 and *FastPriceFeedV2.
// This struct means for marshaling/unmarshaling.
type PriceFeedEnum struct {
	V1 *FastPriceFeedV1
	V2 *FastPriceFeedV2
}

func (f *PriceFeedEnum) get() IFastPriceFeed {
	if f.V1 != nil {
		return f.V1
	}
	if f.V2 != nil {
		return f.V2
	}
	return nil
}

func (f *PriceFeedEnum) set(pf IFastPriceFeed) error {
	switch pf := pf.(type) {
	case *FastPriceFeedV1:
		f.V1 = pf
		return nil
	case *FastPriceFeedV2:
		f.V2 = pf
		return nil
	default:
		return fmt.Errorf("invalid IFastPriceFeed concrete type")
	}
}
