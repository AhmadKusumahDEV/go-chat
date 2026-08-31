package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration
// Compatible with viper/mapstructure
type RedisConfig struct {
	Addr            string        `mapstructure:"addr"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db"`
	PoolSize        int           `mapstructure:"pool_size"`
	MinIdleConns    int           `mapstructure:"min_idle_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	PoolTimeout     time.Duration `mapstructure:"pool_timeout"`
	MaxRetries      int           `mapstructure:"max_retries"`
	MinRetryBackoff time.Duration `mapstructure:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `mapstructure:"max_retry_backoff"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// NewClient is a compatibility wrapper that creates a simple redis.Client
// For production use, prefer NewRedis() which provides connection management
func NewClient(ctx context.Context, addr string, password string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: 0,
	})

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}
