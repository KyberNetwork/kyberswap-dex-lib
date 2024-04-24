package msgpencode

import (
	"encoding/binary"
	"math/big"
	"unsafe"
)

// floatExported's structure is the same as big.Float:
type floatExported struct {
	Prec uint32
	Mode big.RoundingMode
	Acc  big.Accuracy
	Form byte
	Neg  bool
	Mant []big.Word
	Exp  int32
}

func init() {
	if unsafe.Sizeof(big.Float{}) != unsafe.Sizeof(floatExported{}) {
		panic("Sizeof(floatExported) is not equal to Sizeof(big.Float)")
	}
}

func exportFloat(bf *big.Float) *floatExported { return (*floatExported)(unsafe.Pointer(bf)) }

func lenFloat(x *big.Float) int {
	y := exportFloat(x)
	return 4 /* Prec */ +
		1 /* Mode */ +
		1 /* Acc */ +
		1 /* Form */ +
		1 /* Neg */ +
		(len(y.Mant) * 8) /* Mant */ +
		4 /* Exp */
}

func EncodeFloat(x *big.Float) []byte {
	if x == nil {
		return nil
	}

	y := exportFloat(x)
	b := make([]byte, lenFloat(x))
	binary.BigEndian.PutUint32(b[:4], y.Prec)
	b[4] = byte(y.Mode)
	b[5] = *(*byte)(unsafe.Pointer(&y.Acc))
	b[6] = byte(y.Form)
	if y.Neg {
		b[7] = 1
	} else {
		b[7] = 0
	}
	for i, w := range y.Mant {
		binary.BigEndian.PutUint64(b[8+(i*8):], uint64(w))
	}
	exp := *(*uint32)(unsafe.Pointer(&y.Exp))
	binary.BigEndian.PutUint32(b[8+(len(y.Mant)*8):], exp)

	return b
}

func DecodeFloat(b []byte) *big.Float {
	if b == nil {
		return nil
	}

	z := new(big.Float)
	y := exportFloat(z)

	y.Prec = binary.BigEndian.Uint32(b[:4])
	y.Mode = big.RoundingMode(b[4])
	y.Acc = big.Accuracy(*(*int8)(unsafe.Pointer(&b[5])))
	y.Form = b[6]
	y.Neg = b[7] == 1
	y.Mant = make([]big.Word, (len(b)-12)/8)
	for i := range y.Mant {
		y.Mant[i] = big.Word(binary.BigEndian.Uint64(b[8+(i*8):]))
	}
	exp := binary.BigEndian.Uint32(b[8+(len(y.Mant)*8):])
	y.Exp = *(*int32)(unsafe.Pointer(&exp))

	return z
}