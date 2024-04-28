package quickperps

import (
	"bytes"
	"fmt"

	"github.com/tinylib/msgp/msgp"
)

const (
	priceFeedTypeFastPriceFeedV1 = iota
	priceFeedTypeFastPriceFeedV2
)

func encodePriceFeed(pf IFastPriceFeed) []byte {
	if pf == nil {
		return nil
	}

	var (
		priceFeedType byte
		encodable     msgp.Encodable
		buf           = new(bytes.Buffer)
	)
	switch pf := pf.(type) {
	case *FastPriceFeedV1:
		priceFeedType = priceFeedTypeFastPriceFeedV1
		encodable = pf
	case *FastPriceFeedV2:
		priceFeedType = priceFeedTypeFastPriceFeedV2
		encodable = pf
	default:
		panic("invalid IFastPriceFeed concrete type")
	}

	if err := buf.WriteByte(priceFeedType); err != nil {
		panic(fmt.Sprintf("could not encode price feed type: %s", err))
	}

	if err := msgp.Encode(buf, encodable); err != nil {
		panic(fmt.Sprintf("could not encode IFastPriceFeed: %s", err))
	}

	return buf.Bytes()
}

func decodePriceFeed(encoded []byte) IFastPriceFeed {
	if encoded == nil {
		return nil
	}

	var (
		buf           = bytes.NewBuffer(encoded)
		priceFeedType byte
		decodable     msgp.Decodable
		priceFeed     IFastPriceFeed
		err           error
	)
	if priceFeedType, err = buf.ReadByte(); err != nil {
		panic(fmt.Sprintf("could not read price feed type: %s", err))
	}

	switch priceFeedType {
	case priceFeedTypeFastPriceFeedV1:
		v1 := new(FastPriceFeedV1)
		decodable = v1
		priceFeed = v1
	case priceFeedTypeFastPriceFeedV2:
		v2 := new(FastPriceFeedV2)
		decodable = v2
		priceFeed = v2
	default:
		panic(fmt.Sprintf("invalid price feed type %d", priceFeedType))
	}

	if err := msgp.Decode(buf, decodable); err != nil {
		panic(fmt.Sprintf("could not decode IFastPriceFeed: %s", err))
	}

	return priceFeed
}
