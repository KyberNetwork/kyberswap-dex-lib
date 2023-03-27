package repository

import "fmt"

type UnPackMulticallError struct {
	OriginalErr error
}

func NewUnPackMulticallError(originalErr error) error {
	return &UnPackMulticallError{
		OriginalErr: originalErr,
	}
}

func (e UnPackMulticallError) Error() string {
	return fmt.Sprintf("Unpack Multicall Error: %v", e.OriginalErr)
}
