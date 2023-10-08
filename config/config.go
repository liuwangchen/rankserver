package config

import (
	"sync"

	pc "github.com/liuwangchen/toy/pkg/config"
)

type CommonConfig struct {
	NatsAddr  string `yaml:"nats"`
	Namespace string `yaml:"namespace"`
	RedisAddr string `yaml:"redis"`
	PprofAddr string `yaml:"pprof"`
	EtcdAddr  string `yaml:"etcd"`
}

type DynamicConfig struct {
	ServerId  string   `yaml:"server_id"`
	TypeRange [2]int32 `yaml:"type_range"`
}

type StaticConfig struct {
	Web string `yaml:"web"`
}

type Config struct {
	Common *CommonConfig `yaml:"common"`
	Rank   *RankConfig   `yaml:"rank"`
}

type RankConfig struct {
	Dynamic *DynamicConfig `yaml:"dynamic"`
	Static  *StaticConfig  `yaml:"static"`
}

var (
	cfg     *Config
	cfgOnce sync.Once
)

// ----------

func GetInstance() *Config {
	cfgOnce.Do(func() {
		cfg = &Config{}
	})

	return cfg
}

func (c *Config) Load(path string) error {
	return pc.LoadConfigFromFile(path, c, func(bytes []byte) []byte {
		return bytes
	})
}
