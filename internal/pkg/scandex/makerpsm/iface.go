package makerpsm

import (
	"context"
)

type IPSMReader interface {
	Read(ctx context.Context, address string) (*PSM, error)
}

type IVatReader interface {
	Read(ctx context.Context, address string, ilk [32]byte) (*Vat, error)
}
