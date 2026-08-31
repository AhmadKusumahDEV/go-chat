package rabbitmq

import "time"

type Exchange struct {
	Name       string          `mapstructure:"name"`
	Kind       string          `mapstructure:"kind"`
	Durable    bool            `mapstructure:"durable"`
	AutoDelete bool            `mapstructure:"auto_delete"`
	Internal   bool            `mapstructure:"internal"`
	NoWait     bool            `mapstructure:"no_wait"`
	Arguments  *map[string]any `mapstructure:"arguments"`
}

type Binding struct {
	Name      string          `mapstructure:"name"`
	Key       string          `mapstructure:"key"`
	Exchange  string          `mapstructure:"exchange"`
	NoWait    bool            `mapstructure:"no_wait"`
	Arguments *map[string]any `mapstructure:"arguments"`
}

type Queue struct {
	Name       string          `mapstructure:"name"`
	Durable    bool            `mapstructure:"durable"`
	AutoDelete bool            `mapstructure:"auto_delete"`
	Exclusive  bool            `mapstructure:"exclusive"`
	NoWait     bool            `mapstructure:"no_wait"`
	Bindings   []Binding       `mapstructure:"bindings"`
	Arguments  *map[string]any `mapstructure:"arguments"`
}

type RabbitMQConfig struct {
	URL            string        `mapstructure:"url"`
	MaxChannels    int           `mapstructure:"max_channels"`
	ReconnectDelay time.Duration `mapstructure:"reconnect_delay"`
	PrefetchCount  int           `mapstructure:"prefetch_count"`
	PrefetchSize   int           `mapstructure:"prefetch_size"`
	Heartbeat      time.Duration `mapstructure:"heartbeat"`
	ConnectionName string        `mapstructure:"connection_name"`
	Exchange       []Exchange    `mapstructure:"exchanges"`
	Queue          []Queue       `mapstructure:"queues"`
}
