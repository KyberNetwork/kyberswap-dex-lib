package ephemeral

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Contract is a contract released by ephemeral-contracts: the file a release writes next to the
// tracker that runs it, {"abi": ..., "bytecode": ...}, parsed. It packs the call that runs the
// contract and decodes what the contract reverts with: the outputs of the ABI's one function.
type Contract struct {
	Name string

	abi      abi.ABI
	bytecode []byte
	result   abi.Arguments
}

// Load parses a released contract file. name is the contract's, for errors.
func Load(name string, raw []byte) (*Contract, error) {
	var doc struct {
		ABI      json.RawMessage `json:"abi"`
		Bytecode string          `json:"bytecode"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("ephemeral: %s file: %w", name, err)
	}
	parsed, err := abi.JSON(bytes.NewReader(doc.ABI))
	if err != nil {
		return nil, fmt.Errorf("ephemeral: %s ABI: %w", name, err)
	}
	if len(parsed.Methods) != 1 {
		return nil, fmt.Errorf("ephemeral: %s ABI has %d functions, not the one result", name, len(parsed.Methods))
	}
	bytecode, err := hexutil.Decode(doc.Bytecode)
	if err != nil {
		return nil, fmt.Errorf("ephemeral: %s bytecode: %w", name, err)
	} else if len(bytecode) == 0 {
		return nil, fmt.Errorf("ephemeral: %s has no bytecode", name)
	}
	c := &Contract{Name: name, abi: parsed, bytecode: bytecode}
	for _, m := range parsed.Methods {
		c.result = m.Outputs
	}
	return c, nil
}

// MustLoad is Load for an embedded file, which cannot be invalid unless the build is broken.
func MustLoad(name string, raw []byte) *Contract {
	c, err := Load(name, raw)
	if err != nil {
		panic(err)
	}
	return c
}

// PackCall is the data of the eth_call that runs the contract: its creation code followed by the
// constructor arguments. Send it with no to address; the node returns the result as the revert
// data, for UnpackCall.
func (c *Contract) PackCall(args ...any) ([]byte, error) {
	packed, err := c.packArgs(args...)
	if err != nil {
		return nil, err
	}
	return append(common.CopyBytes(c.bytecode), packed...), nil
}

// packArgs is the constructor arguments of PackCall, without the creation code.
func (c *Contract) packArgs(args ...any) ([]byte, error) {
	packed, err := c.abi.Pack("", args...)
	if err != nil {
		return nil, fmt.Errorf("ephemeral: packing %s's arguments: %w", c.Name, err)
	}
	return packed, nil
}

// UnpackCall decodes a revert payload into out, a pointer to a struct with a field per result
// output, named after it.
//
// It accepts only the canonical encoding: every integer within its ABI width, and the payload
// exactly what the values encode back to. That is what tells a result from a revert the contract
// passed up from a contract it called. go-ethereum does not check widths other than 8, 16, 32 and
// 64 when decoding, so a tick word of 0x800000 would otherwise read as +8388608, not an int24.
func (c *Contract) UnpackCall(payload []byte, out any) error {
	if len(payload)%32 != 0 {
		return fmt.Errorf("%d-byte payload is not whole words, so not a result", len(payload))
	}
	values, err := c.result.Unpack(payload)
	if err != nil {
		return err
	}
	for i, arg := range c.result {
		if err := checkWidths(arg.Type, reflect.ValueOf(values[i]), arg.Name); err != nil {
			return err
		}
	}
	again, err := c.result.Pack(values...)
	if err != nil {
		return fmt.Errorf("re-encoding the result: %w", err)
	}
	if !bytes.Equal(again, payload) {
		return errors.New("payload is not the canonical encoding of a result")
	}
	return c.result.Copy(out, values)
}

// ContractError is a revert with one of the contract's own errors.
type ContractError struct {
	Name string
	Args []any
}

func (e *ContractError) Error() string {
	if len(e.Args) == 0 {
		return e.Name + "()"
	}
	return fmt.Sprintf("%s%v", e.Name, e.Args)
}

// UnpackRevert decodes a revert with one of the contract's own errors.
func (c *Contract) UnpackRevert(raw []byte) (*ContractError, error) {
	if len(raw) < 4 {
		return nil, fmt.Errorf("%d-byte revert carries no selector", len(raw))
	}
	for name, e := range c.abi.Errors {
		if !bytes.Equal(raw[:4], e.ID[:4]) {
			continue
		}
		args, err := e.Inputs.Unpack(raw[4:])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return &ContractError{Name: name, Args: args}, nil
	}
	return nil, fmt.Errorf("selector %x is none of %s's errors", raw[:4], c.Name)
}

// IsContractError reports whether err is a revert with the contract error name.
func IsContractError(err error, name string) bool {
	revert, ok := errors.AsType[*RevertError](err)
	if !ok {
		return false
	}
	e, ok := revert.Err.(*ContractError)
	return ok && e.Name == name
}

// checkWidths walks a decoded value along its ABI type and checks every intN and uintN fits N bits.
func checkWidths(t abi.Type, v reflect.Value, path string) error {
	switch t.T {
	case abi.IntTy, abi.UintTy:
		switch t.Size {
		case 8, 16, 32, 64, 256: // native, or a full word
			return nil
		}
		n, ok := v.Interface().(*big.Int)
		if !ok {
			return fmt.Errorf("%s: %T for %s", path, v.Interface(), t)
		}
		if fits(n, t.Size, t.T == abi.IntTy) {
			return nil
		}
		return fmt.Errorf("%s: %s does not fit %s", path, n, t)
	case abi.SliceTy, abi.ArrayTy:
		for i := range v.Len() {
			if err := checkWidths(*t.Elem, v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case abi.TupleTy:
		for i, elem := range t.TupleElems {
			name := path + "." + strings.TrimSpace(t.TupleRawNames[i])
			if err := checkWidths(*elem, v.Field(i), name); err != nil {
				return err
			}
		}
	}
	return nil
}

func fits(n *big.Int, bits int, signed bool) bool {
	if !signed {
		return n.Sign() >= 0 && n.BitLen() <= bits
	}
	limit := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
	return n.Cmp(limit) < 0 && n.Cmp(new(big.Int).Neg(limit)) >= 0
}
