package repository

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	redisv8 "github.com/go-redis/redis/v8"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestScannerStateRedisRepository_GetDexOffset(t *testing.T) {
	t.Run("it should return 0 when dex offset is empty", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		offset, err := repo.GetDexOffset(context.Background(), "kyberswap:offset")

		assert.Equal(t, 0, offset)
		assert.Nil(t, err)
	})

	t.Run("it should return correct dex offset when it exists in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		// Prepare data
		redisServer.HSet(fmt.Sprintf(":%s", KeyScannerState), "kyberswap:offset", "11111")

		offset, err := repo.GetDexOffset(context.Background(), "kyberswap:offset")

		assert.Equal(t, 11111, offset)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		offset, err := repo.GetDexOffset(context.Background(), "kyberswap:offset")

		assert.Equal(t, 0, offset)
		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_SetDexOffset(t *testing.T) {
	t.Run("it should return error when dex offset is empty", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		err = repo.SetDexOffset(context.Background(), "kyberswap:offset", nil)

		assert.Error(t, err)
	})

	t.Run("it should set dex offset correctly", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		err = repo.SetDexOffset(context.Background(), "kyberswap:offset", "11111")

		assert.Nil(t, err)

		offset := redisServer.HGet(fmt.Sprintf(":%s", KeyScannerState), "kyberswap:offset")

		assert.Equal(t, "11111", offset)

	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		err = repo.SetDexOffset(context.Background(), "kyberswap:offset", "11111")

		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_GetScanBlock(t *testing.T) {
	t.Run("it should return 0 when scan block does not exist in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		scanBlock, err := repo.GetScanBlock(context.Background())

		assert.Equal(t, uint64(0), scanBlock)
		assert.Nil(t, err)
	})

	t.Run("it should return correct scan block when it exists in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		// Prepare data
		redisServer.HSet(fmt.Sprintf(":%s", KeyScannerState), FieldScanBlock, "11111")

		offset, err := repo.GetScanBlock(context.Background())

		assert.Equal(t, uint64(11111), offset)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		offset, err := repo.GetDexOffset(context.Background(), "kyberswap:offset")

		assert.Equal(t, 0, offset)
		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_SetScanBlock(t *testing.T) {
	t.Run("it should set scan block correctly", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		err = repo.SetScanBlock(context.Background(), 11111)

		assert.Nil(t, err)

		scanBlock := redisServer.HGet(fmt.Sprintf(":%s", KeyScannerState), FieldScanBlock)

		assert.Equal(t, "11111", scanBlock)

	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		err = repo.SetScanBlock(context.Background(), 11111)

		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_GetGasPrice(t *testing.T) {
	t.Run("it should return nil when gas price does not exist in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		scanBlock, err := repo.GetGasPrice(context.Background())

		assert.Nil(t, scanBlock)
		assert.Errorf(t, err, "redis: nil")
	})

	t.Run("it should return correct gas price when it exists in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		// Prepare data
		redisServer.HSet(fmt.Sprintf(":%s", KeyScannerState), FieldGasPrice, "11111")

		gasPrice, err := repo.GetGasPrice(context.Background())

		assert.Equal(t, true, big.NewFloat(11111).Cmp(gasPrice) == 0)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		gasPrice, err := repo.GetGasPrice(context.Background())

		assert.Nil(t, gasPrice)
		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_SetGasPrice(t *testing.T) {
	t.Run("it should set gas price correctly", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		err = repo.SetGasPrice(context.Background(), "11111")

		assert.Nil(t, err)

		scanBlock := redisServer.HGet(fmt.Sprintf(":%s", KeyScannerState), FieldGasPrice)

		assert.Equal(t, "11111", scanBlock)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		err = repo.SetGasPrice(context.Background(), "11111")

		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_GetL2Fee(t *testing.T) {
	t.Run("it should return nil when l2 fee data does not exist in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		l2Fee, err := repo.GetL2Fee(context.Background())

		assert.Nil(t, l2Fee)
		assert.ErrorIs(t, redisv8.Nil, err)
	})

	t.Run("it should return correct l2 fee when it exists in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		// Prepare data
		redisServer.HSet(fmt.Sprintf(":%s", KeyScannerState), FieldL2Fee, "{\"decimals\":6,\"l1BaseFee\":20000000000,\"overhead\":1000,\"scalar\":10}")

		l2Fee, err := repo.GetL2Fee(context.Background())

		assert.Equal(t, &entity.L2Fee{
			Decimals:  big.NewInt(6),
			L1BaseFee: big.NewInt(20000000000),
			Overhead:  big.NewInt(1000),
			Scalar:    big.NewInt(10),
		}, l2Fee)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		l2Fee, err := repo.GetL2Fee(context.Background())

		assert.Nil(t, l2Fee)
		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_SetL2Fee(t *testing.T) {
	t.Run("it should set l2 fee correctly", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		err = repo.SetL2Fee(context.Background(), &entity.L2Fee{
			Decimals:  big.NewInt(6),
			L1BaseFee: big.NewInt(20000000000),
			Overhead:  big.NewInt(1000),
			Scalar:    big.NewInt(10),
		})

		assert.Nil(t, err)

		l2Fee := redisServer.HGet(fmt.Sprintf(":%s", KeyScannerState), FieldL2Fee)

		assert.Equal(t, "{\"decimals\":6,\"l1BaseFee\":20000000000,\"overhead\":1000,\"scalar\":10}", l2Fee)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewScannerStateRedisRepository(db)

		redisServer.Close()

		err = repo.SetL2Fee(context.Background(), &entity.L2Fee{
			Decimals:  big.NewInt(6),
			L1BaseFee: big.NewInt(20000000000),
			Overhead:  big.NewInt(1000),
			Scalar:    big.NewInt(10),
		})

		assert.Error(t, err)
	})
}

func TestScannerStateRedisRepository_encodeL2Fee(t *testing.T) {
	type args struct {
		f *entity.L2Fee
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			name: "it should encode l2 fee correctly",
			args: args{
				f: &entity.L2Fee{
					Decimals:  big.NewInt(6),
					L1BaseFee: big.NewInt(20000000000),
					Overhead:  big.NewInt(1000),
					Scalar:    big.NewInt(10),
				},
			},
			want:    "{\"decimals\":6,\"l1BaseFee\":20000000000,\"overhead\":1000,\"scalar\":10}",
			wantErr: nil,
		},
		{
			name: "it should encode l2 fee correctly (2)",
			args: args{
				f: &entity.L2Fee{
					Decimals:  big.NewInt(6),
					L1BaseFee: big.NewInt(20000000000),
					Overhead:  big.NewInt(1000),
				},
			},
			want:    "{\"decimals\":6,\"l1BaseFee\":20000000000,\"overhead\":1000,\"scalar\":null}",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup redis server
			redisServer, err := miniredis.Run()
			if err != nil {
				t.Fatalf("failed to setup redis for testing: %v", err.Error())
			}

			// Setup redis client
			port, err := strconv.Atoi(redisServer.Port())
			if err != nil {
				t.Fatalf("failed to convert redis port: %v", err.Error())
			}

			redisConfig := &redis.Config{
				Host:   redisServer.Host(),
				Port:   port,
				Prefix: "",
			}

			db, err := redis.New(redisConfig)
			if err != nil {
				t.Fatalf("failed to init redis client: %v", err.Error())
			}

			repo := NewScannerStateRedisRepository(db)

			redisServer.Close()

			got, err := repo.encodeL2Fee(tt.args.f)

			assert.ErrorIs(t, tt.wantErr, err)
			assert.Equalf(t, tt.want, got, "encodeL2Fee(%v)", tt.args.f)
		})
	}
}

func TestScannerStateRedisRepository_decodeL2Fee(t *testing.T) {
	type args struct {
		l2FeeString string
	}
	tests := []struct {
		name    string
		args    args
		want    *entity.L2Fee
		wantErr error
	}{
		{
			name: "it should decode l2 fee correctly",
			args: args{
				l2FeeString: "{\"decimals\":6,\"l1BaseFee\":20000000000,\"overhead\":1000,\"scalar\":10}",
			},
			want: &entity.L2Fee{
				Decimals:  big.NewInt(6),
				L1BaseFee: big.NewInt(20000000000),
				Overhead:  big.NewInt(1000),
				Scalar:    big.NewInt(10),
			},
			wantErr: nil,
		},
		{
			name: "it should decode l2 fee correctly (2)",
			args: args{
				l2FeeString: "{\"decimals\":6,\"l1BaseFee\":20000000000,\"overhead\":1000}",
			},
			want: &entity.L2Fee{
				Decimals:  big.NewInt(6),
				L1BaseFee: big.NewInt(20000000000),
				Overhead:  big.NewInt(1000),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup redis server
			redisServer, err := miniredis.Run()
			if err != nil {
				t.Fatalf("failed to setup redis for testing: %v", err.Error())
			}

			// Setup redis client
			port, err := strconv.Atoi(redisServer.Port())
			if err != nil {
				t.Fatalf("failed to convert redis port: %v", err.Error())
			}

			redisConfig := &redis.Config{
				Host:   redisServer.Host(),
				Port:   port,
				Prefix: "",
			}

			db, err := redis.New(redisConfig)
			if err != nil {
				t.Fatalf("failed to init redis client: %v", err.Error())
			}

			repo := NewScannerStateRedisRepository(db)

			redisServer.Close()

			got, err := repo.decodeL2Fee(tt.args.l2FeeString)

			assert.ErrorIs(t, tt.wantErr, err)
			assert.Equalf(t, tt.want, got, "encodeL2Fee(%v)", tt.args.l2FeeString)
		})
	}
}
