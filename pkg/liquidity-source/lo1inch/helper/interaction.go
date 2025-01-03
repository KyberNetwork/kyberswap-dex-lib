package helper

import (
	"errors"
	"strings"
)

// Interaction represents an interaction with a contract
type Interaction struct {
	Target Address
	Data   string
}

func NewInteraction(target Address, data string) (*Interaction, error) {
	if !isHexBytes(data) {
		return nil, errors.New("interaction data must be valid hex bytes")
	}
	return &Interaction{
		Target: target,
		Data:   data,
	}, nil
}

func DecodeInteraction(bytes string) (*Interaction, error) {
	iter := NewBytesIter(bytes)
	return NewInteraction(
		Address(iter.NextUint160()),
		iter.Rest(),
	)
}

func (i *Interaction) Encode() string {
	return i.Target.ToString() + strings.TrimPrefix(i.Data, "0x")
}
