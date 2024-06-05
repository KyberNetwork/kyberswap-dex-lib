package msgpack

import (
	"io"
	"reflect"
	"sync"

	"github.com/KyberNetwork/msgpack/v5"
	uniswapentities "github.com/daoleno/uniswap-sdk-core/entities"
)

var (
	uniswapentitiesBaseCurrencyType = reflect.TypeFor[uniswapentities.BaseCurrency]()
	uniswapentitiesEtherType        = reflect.TypeFor[uniswapentities.Ether]()
	uniswapentitiesNativeType       = reflect.TypeFor[uniswapentities.Native]()
	uniswapentitiesTokenType        = reflect.TypeFor[uniswapentities.Token]()
)

var encoderPool = sync.Pool{
	New: func() any {
		en := msgpack.NewEncoder(nil)
		return en
	},
}

func NewEncoder(w io.Writer) *msgpack.Encoder {
	en := encoderPool.Get().(*msgpack.Encoder)
	en.Reset(w)
	en.IncludeUnexported(true)
	return en
}

func PutEncoder(en *msgpack.Encoder) {
	encoderPool.Put(en)
}
