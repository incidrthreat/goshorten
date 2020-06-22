package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
)

// Configuration - the server config
type Configuration struct {
	GRPCProxyAddr   string    `json:"grpc_proxy_addr"`
	ListenInterface string    `json:"listen_interface"`
	Redis           RedisConf `json:"redis_conf"`
	GRPCHost        string    `json:"grpc_host"`
}

// RedisConf - config for the Redis server
type RedisConf struct {
	Host      string `json:"host"`
	Pass      string `json:"pass"`
	DB        int    `json:"logical_database"`
	CharFloor int    `json:"char_floor"`
}

// ConfigFromFile parses the given file and returns the config
func ConfigFromFile(fileName string) (Configuration, error) {
	var conf Configuration

	confjson, err := ioutil.ReadFile(fileName)
	if err != nil {
		return conf, errors.Wrapf(err, "Failed to open the config file at: %s", fileName)
	}

	if err := json.Unmarshal(confjson, &conf); err != nil {
		return conf, errors.Wrapf(err, "Unable to parse the config file at: %s", fileName)
	}

	return conf, nil
}
