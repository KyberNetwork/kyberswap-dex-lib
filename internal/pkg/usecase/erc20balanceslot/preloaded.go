package erc20balanceslot

import (
	"bytes"
	_ "embed"
	"encoding/gob"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

//go:embed preloaded/avalanche
var avalanche []byte

//go:embed preloaded/ethereum
var ethereum []byte

// ERC20 balance slots calculated beforehand. This make bootstrapping router-service more convinent.
var preloadedByPrefix = map[string][]byte{
	"avalanche": avalanche,
	"ethereum":  ethereum,
}

func SerializePreloaded(preloaded entity.TokenBalanceSlots) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(preloaded); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeserializePreloaded(d []byte) (entity.TokenBalanceSlots, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	preloaded := make(entity.TokenBalanceSlots)
	if err := dec.Decode(&preloaded); err != nil {
		return nil, err
	}
	return preloaded, nil
}
