package erc20balanceslot

import (
	"bytes"
	_ "embed"
	"encoding/gob"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	//go:embed preloaded/avalanche
	avalanche []byte
	//go:embed preloaded/ethereum
	ethereum []byte
	//go:embed preloaded/scroll
	scroll []byte
	//go:embed preloaded/arbitrum
	arbitrum []byte
)

// ERC20 balance slots calculated beforehand. This make bootstrapping router-service more convinent.
var preloadedByPrefix = map[valueobject.ChainID][]byte{
	valueobject.ChainIDAvalancheCChain: avalanche,
	valueobject.ChainIDEthereum:        ethereum,
	valueobject.ChainIDScroll:          scroll,
	valueobject.ChainIDArbitrumOne:     arbitrum,
}

func SerializePreloaded(preloaded types.TokenBalanceSlots) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(preloaded); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeserializePreloaded(d []byte) (types.TokenBalanceSlots, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	preloaded := make(types.TokenBalanceSlots)
	if err := dec.Decode(&preloaded); err != nil {
		return nil, err
	}
	return preloaded, nil
}
