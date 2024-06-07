package msgpack

import (
	"reflect"
	"unsafe"

	"github.com/KyberNetwork/msgpack/v5"
	uniswapentities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
)

/*
	uniswapentities.{Token, Native, Ether} contain cyclic pointer alias (self.BaseCurrency.currency = self).
	So we have to ignore the alias while encoding/decoding and set it after decoding to break the cycle. Otherwise, stackoverflow error will occur.
*/

type (
	// Currency is identical to uniswapentities.Currency
	Currency interface {
		IsNative() bool
		IsToken() bool
		ChainId() uint
		Decimals() uint
		Symbol() string
		Name() string
		Equal(other uniswapentities.Currency) bool
		Wrapped() *uniswapentities.Token
	}

	// BaseCurrency 's structure is identical to uniswapentities.BaseCurrency
	BaseCurrency struct {
		currency Currency `msgpack:"-"` // break the cycle
		isNative bool
		isToken  bool
		chainId  uint
		decimals uint
		symbol   string
		name     string
	}

	// token2's structure is identical to uniswapentities.Token
	token2 struct {
		*BaseCurrency
		Address common.Address
	}

	// native2's structure is identical to uniswapentities.Native
	native2 struct {
		*BaseCurrency
		wrapped *uniswapentities.Token
	}

	// ether2's structure is identical to uniswapentities.Ether
	ether2 struct {
		*BaseCurrency
	}
)

func reinterpretTokenAsToken2(t *uniswapentities.Token) *token2 {
	return (*token2)(unsafe.Pointer(t))
}
func reinterpretToken2AsToken(t *token2) *uniswapentities.Token {
	return (*uniswapentities.Token)(unsafe.Pointer(t))
}

func reinterpretNativeAsNative2(t *uniswapentities.Native) *native2 {
	return (*native2)(unsafe.Pointer(t))
}
func reinterpretNative2AsNative(t *native2) *uniswapentities.Native {
	return (*uniswapentities.Native)(unsafe.Pointer(t))
}

func reinterpretEtherAsEther2(t *uniswapentities.Ether) *ether2 {
	return (*ether2)(unsafe.Pointer(t))
}
func reinterpretEther2AsEther(t *ether2) *uniswapentities.Ether {
	return (*uniswapentities.Ether)(unsafe.Pointer(t))
}

func exportValue(v reflect.Value) reflect.Value {
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func init() {
	msgpack.Register(
		uniswapentities.Token{},
		func(en *msgpack.Encoder, v reflect.Value) error {
			var token uniswapentities.Token
			if v.CanInterface() {
				token = v.Interface().(uniswapentities.Token)
			} else {
				token = exportValue(v).Interface().(uniswapentities.Token)
			}
			// encode as token2 to break the cycle
			reinterpreted := *reinterpretTokenAsToken2(&token)
			return en.EncodeValue(reflect.ValueOf(reinterpreted))
		},
		func(de *msgpack.Decoder, v reflect.Value) error {
			reinterpreted := new(token2)
			// decode as token2 to break the cycle
			if err := de.DecodeValue(reflect.ValueOf(reinterpreted).Elem()); err != nil {
				return err
			}
			// set the pointer alias to reconstruct the cycle
			// v is allocated before this function so we can alias to it
			if v.Addr().CanInterface() {
				reinterpreted.BaseCurrency.currency = v.Addr().Interface().(*uniswapentities.Token)
			} else {
				reinterpreted.BaseCurrency.currency = exportValue(v).Addr().Interface().(*uniswapentities.Token)
			}
			// convert back to uniswapentities.Token
			token := *reinterpretToken2AsToken(reinterpreted)
			if v.CanSet() {
				v.Set(reflect.ValueOf(token))
			} else {
				exportValue(v).Set(reflect.ValueOf(token))
			}
			return nil
		},
	)
	msgpack.Register(
		uniswapentities.Ether{},
		func(en *msgpack.Encoder, v reflect.Value) error {
			var ether uniswapentities.Ether
			if v.CanInterface() {
				ether = v.Interface().(uniswapentities.Ether)
			} else {
				ether = exportValue(v).Interface().(uniswapentities.Ether)
			}
			// encode as ether2 to break the cycle
			reinterpreted := *reinterpretEtherAsEther2(&ether)
			return en.EncodeValue(reflect.ValueOf(reinterpreted))
		},
		func(de *msgpack.Decoder, v reflect.Value) error {
			reinterpreted := new(ether2)
			// decode as token2 to break the cycle
			if err := de.DecodeValue(reflect.ValueOf(reinterpreted).Elem()); err != nil {
				return err
			}
			// set the pointer alias to reconstruct the cycle
			// v is allocated before this function so we can alias to it
			if v.Addr().CanInterface() {
				reinterpreted.BaseCurrency.currency = v.Addr().Interface().(*uniswapentities.Ether)
			} else {
				reinterpreted.BaseCurrency.currency = exportValue(v).Addr().Interface().(*uniswapentities.Ether)
			}
			// convert back to uniswapentities.Ether
			ether := *reinterpretEther2AsEther(reinterpreted)
			if v.CanSet() {
				v.Set(reflect.ValueOf(ether))
			} else {
				exportValue(v).Set(reflect.ValueOf(ether))
			}
			return nil
		},
	)
	msgpack.Register(
		uniswapentities.Native{},
		func(en *msgpack.Encoder, v reflect.Value) error {
			var native uniswapentities.Native
			if v.CanInterface() {
				native = v.Interface().(uniswapentities.Native)
			} else {
				native = exportValue(v).Interface().(uniswapentities.Native)
			}
			// encode as native2 to break the cycle
			reinterpreted := *reinterpretNativeAsNative2(&native)
			return en.EncodeValue(reflect.ValueOf(reinterpreted))
		},
		func(de *msgpack.Decoder, v reflect.Value) error {
			reinterpreted := new(native2)
			// decode as native2 to break the cycle
			if err := de.DecodeValue(reflect.ValueOf(reinterpreted).Elem()); err != nil {
				return err
			}
			// set the pointer alias to reconstruct the cycle
			// v is allocated before this function so we can alias to it
			if v.Addr().CanInterface() {
				reinterpreted.BaseCurrency.currency = v.Addr().Interface().(*uniswapentities.Native)
			} else {
				reinterpreted.BaseCurrency.currency = exportValue(v).Addr().Interface().(*uniswapentities.Native)
			}
			// convert back to uniswapentities.Native
			native := *reinterpretNative2AsNative(reinterpreted)
			if v.CanSet() {
				v.Set(reflect.ValueOf(native))
			} else {
				exportValue(v).Set(reflect.ValueOf(native))
			}
			return nil
		},
	)
}
