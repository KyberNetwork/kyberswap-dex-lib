package stonkbrokersfunv2

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// A launch has no address of its own: it trades through its lane's pad, and the
// pool key is "<pad>_<launchId>". Routing a pad log therefore means decoding
// that id out of it.
var padEventIDs = map[common.Hash]struct{}{
	// Trades: move vQuote/vToken.
	abiutil.MustABIEvent(PadABI, "SafeBuy").ID:     {},
	abiutil.MustABIEvent(PadABI, "SafeSell").ID:    {},
	abiutil.MustABIEvent(PadABI, "LaunchArmed").ID: {},
	// The three terminal transitions isTerminal parks the pool on.
	abiutil.MustABIEvent(PadABI, "CurveClosed").ID:   {},
	abiutil.MustABIEvent(PadABI, "LaunchBonded").ID:  {},
	abiutil.MustABIEvent(PadABI, "LaunchAborted").ID: {},
}

type EventParserConfig struct {
	DexID string `json:"dexID,omitempty"`
}

type EventParser struct {
	config *EventParserConfig
}

var _ = poolfactory.RegisterFactoryC(DexType, NewEventParser)

func NewEventParser(config *EventParserConfig) *EventParser {
	return &EventParser{config: config}
}

func (p *EventParser) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	if len(log.Topics) < 2 {
		return nil, nil
	}
	if _, ok := padEventIDs[log.Topics[0]]; !ok {
		return nil, nil
	}

	// All six events index the launch id first.
	launchID := new(big.Int).SetBytes(log.Topics[1][:])
	if !launchID.IsUint64() {
		return nil, nil
	}

	return []string{PoolAddress(hexutil.Encode(log.Address[:]), launchID.Uint64())}, nil
}

func (p *EventParser) DecodePoolCreated(_ types.Log) (*entity.Pool, error) {
	return nil, nil
}

func (p *EventParser) IsEventSupported(_ common.Hash) bool {
	return false
}
