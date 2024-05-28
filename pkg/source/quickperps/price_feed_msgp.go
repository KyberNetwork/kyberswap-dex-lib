package quickperps

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode/interfacemsgp"
	"github.com/tinylib/msgp/msgp"
)

var priceFeedEncoderHelper = interfacemsgp.NewEncoderHelper[IFastPriceFeed]()

// IFastPriceFeedWrapper is a wrapper of IFastPriceFeed and implements msgp.Encodable, msgp.Decodable, msgp.Marshaler, msgp.Unmarshaler, and msgp.Sizer
type IFastPriceFeedWrapper struct {
	IFastPriceFeed
}

func NewIFastPriceFeedWrapper(priceFeed IFastPriceFeed) *IFastPriceFeedWrapper {
	if priceFeed == nil {
		return nil
	}
	return &IFastPriceFeedWrapper{priceFeed}
}

// EncodeMsg implements msgp.Encodable
func (p *IFastPriceFeedWrapper) EncodeMsg(en *msgp.Writer) (err error) {
	return priceFeedEncoderHelper.EncodeMsg(p.IFastPriceFeed, en)
}

// DecodeMsg implements msgp.Decodable
func (p *IFastPriceFeedWrapper) DecodeMsg(dc *msgp.Reader) (err error) {
	p.IFastPriceFeed, err = priceFeedEncoderHelper.DecodeMsg(dc)
	return
}

// MarshalMsg implements msgp.Marshaler
func (p *IFastPriceFeedWrapper) MarshalMsg(b []byte) (o []byte, err error) {
	return priceFeedEncoderHelper.MarshalMsg(p.IFastPriceFeed, b)
}

// UnmarshalMsg implements msgp.Unmarshaler
func (p *IFastPriceFeedWrapper) UnmarshalMsg(bts []byte) (o []byte, err error) {
	p.IFastPriceFeed, o, err = priceFeedEncoderHelper.UnmarshalMsg(bts)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (p *IFastPriceFeedWrapper) Msgsize() int {
	return priceFeedEncoderHelper.Msgsize(p.IFastPriceFeed)
}

// MarshalJSON marshal embedded interface
func (p *IFastPriceFeedWrapper) MarshalJSON() ([]byte, error) { return json.Marshal(p.IFastPriceFeed) }
