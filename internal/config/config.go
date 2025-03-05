package config

import (
	"os"

	"github.com/libops/isle-event-bus/internal/stomp"
	yaml "gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Queues []stomp.Queue `yaml:"queues,omitempty"`
}

func ReadConfig(yp string) (*ServerConfig, error) {
	var (
		y   []byte
		err error
	)
	yml := os.Getenv("isle-event-bus_YML")
	if yml != "" {
		y = []byte(yml)
	} else {
		y, err = os.ReadFile(yp)
		if err != nil {
			return nil, err
		}
	}

	var c ServerConfig
	err = yaml.Unmarshal(y, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
