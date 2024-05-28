package ibasepool

import (
	"encoding/json"
	"math/big"

	"github.com/tinylib/msgp/msgp"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode/interfacemsgp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var encoderHelper = interfacemsgp.NewEncoderHelper[ICurveBasePool]()

// ICurveBasePool is the interface for curve base pool inside a meta pool
// It can be:
// 1. base/plain pool
// 2. plain oracle pool
// 3. lending pool
// 4. or even meta pool
// At the moment, our code can only support base/plain pool and plain oracle pool
type ICurveBasePool interface {
	GetInfo() pool.PoolInfo
	GetTokenIndex(address string) int
	// return both vPrice and D
	GetVirtualPrice() (vPrice *big.Int, D *big.Int, err error)
	// if `dCached` is nil then will be recalculated
	GetDy(i int, j int, dx *big.Int, dCached *big.Int) (*big.Int, *big.Int, error)
	CalculateTokenAmount(amounts []*big.Int, deposit bool) (*big.Int, error)
	CalculateWithdrawOneCoin(tokenAmount *big.Int, i int) (*big.Int, *big.Int, error)
	AddLiquidity(amounts []*big.Int) (*big.Int, error)
	RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error)
}

// RegisterICurveBasePoolImpl registers the concrete types of an ICurveBasePool.
// This function is not thread-safe and should be only call in init().
func RegisterICurveBasePoolImpl(base ICurveBasePool) {
	if err := encoderHelper.RegisterType(base); err != nil {
		panic(err)
	}
}

// ICurveBasePoolWrapper is a wrapper of ICurveBasePool and implements msgp.Encodable, msgp.Decodable, msgp.Marshaler, msgp.Unmarshaler, and msgp.Sizer
type ICurveBasePoolWrapper struct {
	ICurveBasePool
}

func NewICurveBasePoolWrapper(base ICurveBasePool) *ICurveBasePoolWrapper {
	if base == nil {
		return nil
	}
	return &ICurveBasePoolWrapper{base}
}

// EncodeMsg implements msgp.Encodable
func (p *ICurveBasePoolWrapper) EncodeMsg(en *msgp.Writer) (err error) {
	return encoderHelper.EncodeMsg(p.ICurveBasePool, en)
}

// DecodeMsg implements msgp.Decodable
func (p *ICurveBasePoolWrapper) DecodeMsg(dc *msgp.Reader) (err error) {
	p.ICurveBasePool, err = encoderHelper.DecodeMsg(dc)
	return
}

// MarshalMsg implements msgp.Marshaler
func (p *ICurveBasePoolWrapper) MarshalMsg(b []byte) (o []byte, err error) {
	return encoderHelper.MarshalMsg(p.ICurveBasePool, b)
}

// UnmarshalMsg implements msgp.Unmarshaler
func (p *ICurveBasePoolWrapper) UnmarshalMsg(bts []byte) (o []byte, err error) {
	p.ICurveBasePool, o, err = encoderHelper.UnmarshalMsg(bts)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (p *ICurveBasePoolWrapper) Msgsize() int {
	return encoderHelper.Msgsize(p.ICurveBasePool)
}

// MarshalJSON marshal embedded interface
func (p *ICurveBasePoolWrapper) MarshalJSON() ([]byte, error) { return json.Marshal(p.ICurveBasePool) }
