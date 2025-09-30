package pools

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"golang.org/x/crypto/sha3"
)

var (
	addressType, _      = abi.NewType("address", "address", nil)
	bytes32Type, _      = abi.NewType("bytes32", "bytes32", nil)
	poolKeyABIArguments = abi.Arguments{
		{Type: addressType},
		{Type: addressType},
		{Type: bytes32Type},
	}
)

type PoolKey struct {
	Token0 common.Address `json:"token0"`
	Token1 common.Address `json:"token1"`
	Config PoolConfig     `json:"config"`

	stringId string
	numId    []byte
}

type AbiPoolKey struct {
	Token0 common.Address `json:"token0"`
	Token1 common.Address `json:"token1"`
	Config common.Hash    `json:"config"`
}

func NewPoolKey(token0, token1 common.Address, config PoolConfig) *PoolKey {
	return &PoolKey{
		Token0: token0,
		Token1: token1,
		Config: config,
	}
}

func (k *PoolKey) CloneState() *PoolKey {
	cloned := *k
	cloned.Config.compressed = slices.Clone(k.Config.compressed)
	return &cloned
}

func (k *PoolKey) StringId() string {
	if k.stringId == "" {
		k.stringId = k.Token0.Hex() + "/" + k.Token1.Hex() +
			"_" + strconv.FormatUint(k.Config.Fee, 10) +
			"_" + strconv.FormatUint(uint64(k.Config.TickSpacing), 10) +
			"_" + k.Config.Extension.Hex()
	}

	return k.stringId
}

func (k *PoolKey) NumId() ([]byte, error) {
	if k.numId == nil {
		enc, err := poolKeyABIArguments.Pack(
			k.Token0,
			k.Token1,
			[32]byte(k.Config.Compressed()),
		)
		if err != nil {
			return nil, fmt.Errorf("computing numerical id: %w", err)
		}

		hash := sha3.NewLegacyKeccak256()

		if _, err := hash.Write(enc); err != nil {
			return nil, fmt.Errorf("computing digest: %w", err)
		}

		k.numId = hash.Sum(nil)
	}

	return k.numId, nil
}

func (k *PoolKey) ToPoolAddress() (string, error) {
	numId, err := k.NumId()
	if err != nil {
		return "", err
	}

	return hexutil.Encode(numId), nil
}

func (k *PoolKey) ToAbi() AbiPoolKey {
	return AbiPoolKey{
		Token0: k.Token0,
		Token1: k.Token1,
		Config: common.Hash(k.Config.Compressed()),
	}
}

type PoolConfig struct {
	Fee         uint64         `json:"fee"`
	TickSpacing uint32         `json:"tickSpacing"`
	Extension   common.Address `json:"extension"`

	compressed []byte
}

func (c *PoolConfig) Compressed() []byte {
	if c.compressed == nil {
		c.compressed = append(c.compressed, c.Extension.Bytes()...)
		c.compressed = binary.BigEndian.AppendUint64(c.compressed, c.Fee)
		c.compressed = binary.BigEndian.AppendUint32(c.compressed, c.TickSpacing)
	}

	return c.compressed
}
