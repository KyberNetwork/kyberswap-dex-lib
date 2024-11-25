package erc20balanceslot

import (
	"bytes"
	_ "embed"
	"encoding/gob"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	//go:embed embedded/avalanche
	avalanche []byte
	//go:embed embedded/ethereum
	ethereum []byte
	//go:embed embedded/scroll
	scroll []byte
	//go:embed embedded/arbitrum
	arbitrum []byte
)

// ERC20 balance slots calculated beforehand. This make bootstrapping router-service more convinent.
var embeddedByPrefix = map[valueobject.ChainID][]byte{
	valueobject.ChainIDAvalancheCChain: avalanche,
	valueobject.ChainIDEthereum:        ethereum,
	valueobject.ChainIDScroll:          scroll,
	valueobject.ChainIDArbitrumOne:     arbitrum,
}

func SerializeEmbedded(embedded types.TokenBalanceSlots) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(embedded); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeserializeEmbedded(d []byte) (types.TokenBalanceSlots, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	embedded := make(types.TokenBalanceSlots)
	if err := dec.Decode(&embedded); err != nil {
		return nil, err
	}
	return embedded, nil
}
