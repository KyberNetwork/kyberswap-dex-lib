package decode

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode"
)

type DecoderFactory struct {
	config Config
}

type IDecoder interface {
	Decode(data string) (interface{}, error)
}

func NewDecoderFactory(config Config) *DecoderFactory {
	return &DecoderFactory{config: config}
}

func (d *DecoderFactory) GetDecoder() IDecoder {
	if d.config.UseL2Optimize && encode.IsL2EncoderSupportedChains(d.config.ChainID) {
		return NewL2Decoder(d.config)
	}

	return &Decoder{}
}
