package repository

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	redisv8 "github.com/go-redis/redis/v8"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"
)

const KeyScannerState = "configs"
const FieldScanBlock = "scanBlock"
const FieldGasPrice = "GAS_PRICE"
const FieldCurveProviders = "curveProviders"

const FieldL2Fee = "l2Fee"

var ErrInvalidGasPrice = errors.New("invalid gas price")

type ScannerStateRepository struct {
	db *redis.Redis
}

func NewScannerStateRedisRepository(db *redis.Redis) *ScannerStateRepository {
	return &ScannerStateRepository{
		db: db,
	}
}

func (r *ScannerStateRepository) GetDexOffset(ctx context.Context, offsetKey string) (int, error) {
	offset, err := r.db.Client.HGet(ctx, r.db.FormatKey(KeyScannerState), offsetKey).Int()

	if err != nil {
		switch {
		case errors.Is(err, redisv8.Nil):
			return 0, nil
		default:
			return 0, err
		}
	}

	return offset, nil
}

func (r *ScannerStateRepository) SetDexOffset(ctx context.Context, offsetKey string, offset interface{}) error {
	if offset == nil {
		return errors.New("offset is nil")
	}

	return r.db.Client.HSet(ctx, r.db.FormatKey(KeyScannerState), offsetKey, offset).Err()
}

func (r *ScannerStateRepository) GetScanBlock(ctx context.Context) (uint64, error) {
	block, err := r.db.Client.HGet(ctx, r.db.FormatKey(KeyScannerState), FieldScanBlock).Uint64()
	if err != nil {
		switch {
		case errors.Is(err, redisv8.Nil):
			return 0, nil
		default:
			return 0, err
		}
	}

	return block, nil
}

func (r *ScannerStateRepository) SetScanBlock(ctx context.Context, block uint64) error {
	return r.db.Client.HSet(ctx, r.db.FormatKey(KeyScannerState), FieldScanBlock, block).Err()
}

func (r *ScannerStateRepository) GetGasPrice(ctx context.Context) (*big.Float, error) {
	gasPriceStr, err := r.db.Client.HGet(ctx, r.db.FormatKey(KeyScannerState), FieldGasPrice).Result()

	if err != nil {
		return nil, err
	}

	gasPrice, ok := new(big.Float).SetString(gasPriceStr)

	if !ok {
		return nil, ErrInvalidGasPrice
	}

	return gasPrice, nil
}

func (r *ScannerStateRepository) SetGasPrice(ctx context.Context, gasPrice string) error {
	return r.db.Client.HSet(ctx, r.db.FormatKey(KeyScannerState), FieldGasPrice, gasPrice).Err()
}

func (r *ScannerStateRepository) GetCurveAddressProviders(ctx context.Context) (string, error) {
	addressesFromProviderStr, err := r.db.Client.HGet(ctx, r.db.FormatKey(KeyScannerState), FieldCurveProviders).Result()
	if err != nil {
		switch {
		case errors.Is(err, redisv8.Nil):
			return "", nil
		default:
			return "", err
		}
	}

	return addressesFromProviderStr, nil
}

func (r *ScannerStateRepository) SetCurveAddressProviders(ctx context.Context, providers string) error {
	return r.db.Client.HSet(ctx, r.db.FormatKey(KeyScannerState), FieldCurveProviders, providers).Err()
}

func (r *ScannerStateRepository) GetL2Fee(ctx context.Context) (*entity.L2Fee, error) {
	l2FeeString, err := r.db.Client.HGet(ctx, r.db.FormatKey(KeyScannerState), FieldL2Fee).Result()
	if err != nil {
		return nil, err
	}

	return r.decodeL2Fee(l2FeeString)
}

func (r *ScannerStateRepository) SetL2Fee(ctx context.Context, l2fee *entity.L2Fee) error {
	encodedL2Fee, err := r.encodeL2Fee(l2fee)
	if err != nil {
		return err
	}

	return r.db.Client.HSet(ctx, r.db.FormatKey(KeyScannerState), FieldL2Fee, encodedL2Fee).Err()
}

func (r *ScannerStateRepository) encodeL2Fee(f *entity.L2Fee) (string, error) {
	bytes, err := json.Marshal(f)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (r *ScannerStateRepository) decodeL2Fee(l2FeeString string) (*entity.L2Fee, error) {
	var f entity.L2Fee
	err := json.Unmarshal([]byte(l2FeeString), &f)
	if err != nil {
		return nil, err
	}

	return &f, nil
}
