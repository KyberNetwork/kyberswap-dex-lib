package pools

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
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

type (
	PoolKey[T PoolTypeConfig] struct {
		Token0 common.Address `json:"token0"`
		Token1 common.Address `json:"token1"`
		Config PoolConfig[T]  `json:"config"`

		numId []byte
	}

	IPoolKey interface {
		Token0Address() common.Address
		Token1Address() common.Address
		Fee() uint64
		Extension() common.Address
		ToAbi() AbiPoolKey
		NumId() ([]byte, error)
	}

	PoolConfig[T PoolTypeConfig] struct {
		Fee        uint64         `json:"fee"`
		TypeConfig T              `json:"typeConfig"`
		Extension  common.Address `json:"extension"`

		compressed []byte
	}

	PoolTypeConfig interface {
		Compressed() [4]byte
		String() string
	}

	AbiPoolKey struct {
		Token0 common.Address `json:"token0"`
		Token1 common.Address `json:"token1"`
		Config common.Hash    `json:"config"`
	}

	AnyPoolKey struct {
		*PoolKey[PoolTypeConfig]
	}

	ConcentratedPoolKey = PoolKey[ConcentratedPoolTypeConfig]
	FullRangePoolKey    = PoolKey[FullRangePoolTypeConfig]
	StableswapPoolKey   = PoolKey[StableswapPoolTypeConfig]

	ConcentratedPoolTypeConfig struct {
		TickSpacing uint32 `json:"tickSpacing"`
	}
	FullRangePoolTypeConfig  struct{}
	StableswapPoolTypeConfig struct {
		CenterTick          int32 `json:"centerTick"`
		AmplificationFactor uint8 `json:"amplificationFactor"`
	}
)

func (k *PoolKey[T]) Token0Address() common.Address {
	return k.Token0
}

func (k *PoolKey[T]) Token1Address() common.Address {
	return k.Token1
}

func (k *PoolKey[T]) Extension() common.Address {
	return k.Config.Extension
}

func (k *PoolKey[T]) Fee() uint64 {
	return k.Config.Fee
}

func (k *PoolKey[T]) CloneState() *PoolKey[T] {
	cloned := *k
	cloned.Config.compressed = slices.Clone(k.Config.compressed)
	return &cloned
}

func (k *PoolKey[T]) ToAbi() AbiPoolKey {
	return AbiPoolKey{
		Token0: k.Token0,
		Token1: k.Token1,
		Config: common.Hash(k.Config.Compressed()),
	}
}

func (k *PoolKey[T]) ToFullRange(config FullRangePoolTypeConfig) *FullRangePoolKey {
	return poolKeyWithConfig(k, config)
}

func (k *PoolKey[T]) ToStableswap(config StableswapPoolTypeConfig) *StableswapPoolKey {
	return poolKeyWithConfig(k, config)
}

func (k *PoolKey[T]) ToConcentrated(config ConcentratedPoolTypeConfig) *ConcentratedPoolKey {
	return poolKeyWithConfig(k, config)
}

func (k *PoolKey[T]) ToPoolAddress() (string, error) {
	numId, err := k.NumId()
	if err != nil {
		return "", err
	}

	return hexutil.Encode(numId), nil
}

func (k *PoolKey[T]) NumId() ([]byte, error) {
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

func (k *AnyPoolKey) UnmarshalJSON(data []byte) error {
	type rawPoolConfig struct {
		Fee        uint64          `json:"fee"`
		Extension  common.Address  `json:"extension"`
		TypeConfig json.RawMessage `json:"typeConfig"`
	}
	type rawPoolKey struct {
		Token0 common.Address `json:"token0"`
		Token1 common.Address `json:"token1"`
		Config rawPoolConfig  `json:"config"`
	}

	var raw rawPoolKey
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal pool key: %w", err)
	}

	if len(raw.Config.TypeConfig) == 0 || string(raw.Config.TypeConfig) == "null" {
		return fmt.Errorf("missing pool type config")
	}

	var typeFields map[string]json.RawMessage
	if err := json.Unmarshal(raw.Config.TypeConfig, &typeFields); err != nil {
		return fmt.Errorf("unmarshal pool type config: %w", err)
	}

	var typeConfig PoolTypeConfig
	switch {
	case len(typeFields) == 0:
		typeConfig = NewFullRangePoolTypeConfig()
	case typeFields["tickSpacing"] != nil:
		var cfg ConcentratedPoolTypeConfig
		if err := json.Unmarshal(raw.Config.TypeConfig, &cfg); err != nil {
			return fmt.Errorf("unmarshal concentrated pool type config: %w", err)
		}
		typeConfig = cfg
	case typeFields["amplificationFactor"] != nil || typeFields["centerTick"] != nil:
		var cfg StableswapPoolTypeConfig
		if err := json.Unmarshal(raw.Config.TypeConfig, &cfg); err != nil {
			return fmt.Errorf("unmarshal stableswap pool type config: %w", err)
		}
		typeConfig = cfg
	default:
		return fmt.Errorf("unknown pool type config: %s", string(raw.Config.TypeConfig))
	}

	k.PoolKey = NewPoolKey(
		raw.Token0,
		raw.Token1,
		NewPoolConfig(raw.Config.Extension, raw.Config.Fee, typeConfig),
	)

	return nil
}

func (c *PoolConfig[T]) Compressed() common.Hash {
	if c.compressed == nil {
		c.compressed = append(c.compressed, c.Extension.Bytes()...)
		c.compressed = binary.BigEndian.AppendUint64(c.compressed, c.Fee)

		typeConfigCompressed := c.TypeConfig.Compressed()
		c.compressed = append(c.compressed, typeConfigCompressed[:]...)
	}

	return common.Hash(c.compressed)
}

func (c *PoolConfig[T]) String() string {
	return c.Extension.Hex() +
		"_" + strconv.FormatUint(c.Fee, 10) +
		"_" + c.TypeConfig.String()
}

func (c FullRangePoolTypeConfig) Compressed() [4]byte {
	return [4]byte{}
}

func (c FullRangePoolTypeConfig) String() string {
	return "full_range"
}

func (c ConcentratedPoolTypeConfig) Compressed() [4]byte {
	var compressed [4]byte
	binary.BigEndian.PutUint32(compressed[:], c.TickSpacing)
	compressed[0] |= 0x80
	return compressed
}

func (c ConcentratedPoolTypeConfig) String() string {
	return "concentrated_" + strconv.FormatUint(uint64(c.TickSpacing), 10)
}

func (c StableswapPoolTypeConfig) Compressed() [4]byte {
	var compressed [4]byte
	binary.BigEndian.PutUint32(compressed[:], uint32(c.CenterTick/16))
	compressed[0] = c.AmplificationFactor
	return compressed
}

func (c StableswapPoolTypeConfig) String() string {
	return "stableswap_" + strconv.FormatInt(int64(c.CenterTick), 10) + strconv.FormatUint(uint64(c.AmplificationFactor), 10)
}

func NewPoolKey[T PoolTypeConfig](token0, token1 common.Address, config PoolConfig[T]) *PoolKey[T] {
	return &PoolKey[T]{
		Token0: token0,
		Token1: token1,
		Config: config,
	}
}

func NewPoolConfig[T PoolTypeConfig](extension common.Address, fee uint64, typeConfig T) PoolConfig[T] {
	return PoolConfig[T]{
		Extension:  extension,
		Fee:        fee,
		TypeConfig: typeConfig,
	}
}

func NewFullRangePoolTypeConfig() FullRangePoolTypeConfig {
	return FullRangePoolTypeConfig{}
}

func NewConcentratedPoolTypeConfig(tickSpacing uint32) ConcentratedPoolTypeConfig {
	return ConcentratedPoolTypeConfig{TickSpacing: tickSpacing}
}

func NewStableswapPoolTypeConfig(centerTick int32, amplificationFactor uint8) StableswapPoolTypeConfig {
	return StableswapPoolTypeConfig{
		CenterTick:          centerTick,
		AmplificationFactor: amplificationFactor,
	}
}

func poolKeyWithConfig[C1, C2 PoolTypeConfig](key *PoolKey[C1], config C2) *PoolKey[C2] {
	return NewPoolKey(key.Token0, key.Token1, NewPoolConfig(key.Config.Extension, key.Config.Fee, config))
}
