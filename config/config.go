package config

import (
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

type Servers struct {
	Pool       []string `toml:"pool"`
	Ping       int      `toml:"ping"`
	ErrorLimit int      `toml:"errorLimit"`
}

type Certificate struct {
	Enabled  bool   `toml:"enabled"`
	CertFile string `toml:"certFile"`
	KeyFile  string `toml:"keyFile"`
}

type Loadbalancer struct {
	Algorithm string `toml:"algorithm"`
	Weights   []int  `toml:"weights"`
}

type Caching struct {
	Enabled bool `toml:"enabled"`
}

type Ratelimiting struct {
	Enabled  bool    `toml:"enabled"`
	Capacity float64 `toml:"capacity"`
	Rate     float64 `toml:"rate"`
}

type Config struct {
	Servers      Servers      `toml:"servers"`
	Certificate  Certificate  `toml:"certificate"`
	Loadbalancer Loadbalancer `toml:"loadbalancer"`
	Caching      Caching      `toml:"caching"`
	Ratelimiting Ratelimiting `toml:"ratelimiting"`
}

func LoadConfig() *Config {
	// Check if config.toml exists, else use default.toml
	fileName := "config.toml"
	if _, err := os.Stat(fileName); err != nil {
		log.Println("config.toml not found - using default config")
		fileName = "default.toml"
	}

	// Load config file into a new Config struct
	var conf Config
	_, err := toml.DecodeFile(fileName, &conf)
	if err != nil {
		log.Fatal("Error loading config file - ", err)
	}

	// If no server ip was provided
	if len(conf.Servers.Pool) == 0 {
		log.Fatal("Empty server pool, kindly check your config file")
	}
	// Safe defaults as fallback
	if conf.Servers.Ping < 1 {
		conf.Servers.Ping = 60
	}
	if conf.Ratelimiting.Capacity < 1 {
		conf.Ratelimiting.Capacity = 10
	}
	if conf.Ratelimiting.Rate < 1 {
		conf.Ratelimiting.Rate = 2
	}
	if conf.Loadbalancer.Algorithm == "weighted" || conf.Loadbalancer.Algorithm == "weightedLeast" {
		if len(conf.Servers.Pool) != len(conf.Loadbalancer.Weights) {
			log.Println("Invalid weights list, falling back to random load balancing")
			conf.Loadbalancer.Algorithm = "random"
		}
	}
	return &conf
}
