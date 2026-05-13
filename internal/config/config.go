package config

import (
	"strings"

	"github.com/spf13/viper"
)

type JwtConfig struct {
	SecretKeyrefresh       string `mapstructure:"RefreshSecretKey"`
	SecretKeyAccess        string `mapstructure:"AccessSecretKey"`
	AccessTokenExpiration  int    `mapstructure:"access_token_expiration"`
	RefreshTokenExpiration int    `mapstructure:"refresh_token_expiration"`
}

type Cfg struct {
	AppEnv      string         `mapstructure:"APP_ENV"`
	DatabaseURL string         `mapstructure:"DATABASE_URL"`
	Secret      string         `mapstructure:"secret_key"`
	RabbitMQ    RabbitMQConfig `mapstructure:"rabbitmq"`
	Redis       RedisConfig    `mapstructure:"redis"`
	Server      ServerConfig   `mapstructure:"server"`
	Jwt         JwtConfig      `mapstructure:"jwt"`
	OAuth       OAuthConfig    `mapstructure:"oauth"`
	Firebase    FirebaseConfig `mapstructure:"firebase"`
}

func LoadConfig() (config Cfg, err error) {
	// Menentukan path dan nama file konfigurasi
	viper.AddConfigPath("../../configs")
	viper.AddConfigPath("./configs")

	viper.SetConfigName("local")
	viper.SetConfigType("yaml")

	// Mengaktifkan pembacaan environment variables secara otomatis
	viper.AutomaticEnv()
	// Mengganti tanda '.' dengan '_' saat membaca env var, misal: server.port menjadi SERVER_PORT
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind nested structs explicitly
	viper.BindEnv("rabbitmq.url", "RABBITMQ_URL")
	viper.BindEnv("redis.addr", "REDIS_ADDR")
	viper.BindEnv("server.port", "SERVER_PORT")

	// Menetapkan nilai default (prioritas terendah)
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	viper.SetDefault("server_port", 8081)

	// Membaca file konfigurasi
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
	}

	// Unmarshal (memasukkan) nilai konfigurasi ke dalam struct Config
	err = viper.Unmarshal(&config)
	return
}
