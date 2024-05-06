//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Asset Gas
//msgp:ignore Extra Metadata SubgraphPool SubgraphAsset AssetSubgraph UnderlyingTokenSubgraph
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package wombat

import "math/big"

type Extra struct {
	Paused        bool             `json:"paused"`
	HaircutRate   *big.Int         `json:"haircutRate"`
	AmpFactor     *big.Int         `json:"ampFactor"`
	StartCovRatio *big.Int         `json:"startCovRatio"`
	EndCovRatio   *big.Int         `json:"endCovRatio"`
	AssetMap      map[string]Asset `json:"assetMap"`
}

type Asset struct {
	IsPause                 bool     `json:"isPause"`
	Address                 string   `json:"address"`
	Cash                    *big.Int `json:"cash"`
	Liability               *big.Int `json:"liability"`
	UnderlyingTokenDecimals uint8    `json:"underlyingTokenDecimals"`
	RelativePrice           *big.Int `json:"relativePrice"`
}

type Gas struct {
	Swap int64
}

type Metadata struct {
	LastCreateTime uint64 `json:"lastCreateTime"`
}

type SubgraphPool struct {
	ID               string          `json:"id"`
	Assets           []AssetSubgraph `json:"assets"`
	CreatedTimestamp string          `json:"createdTimestamp"`
}

type SubgraphAsset struct {
	Assets []AssetSubgraph `json:"assets"`
}

type AssetSubgraph struct {
	ID              string                  `json:"id"`
	UnderlyingToken UnderlyingTokenSubgraph `json:"underlyingToken"`
	IsPaused        bool                    `json:"isPaused"`
}

type UnderlyingTokenSubgraph struct {
	ID       string `json:"id"`
	Decimals uint8  `json:"decimals"`
}
