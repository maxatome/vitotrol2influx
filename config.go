package main

import (
	"io/ioutil"

	"github.com/maxatome/go-vitotrol"
	"gopkg.in/yaml.v2"
)

type ConfigInflux struct {
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
	Address  string `yaml:"address"`
}

type ConfigVitotrol struct {
	Login        string `yaml:"login"`
	Password     string `yaml:"password"`
	RetryTimeout uint   `yaml:"retry_timeout"`
	Frequency    uint   `yaml:"frequency"`
}

type ConfigDevice struct {
	Name             string            `yaml:"name"`
	Location         string            `yaml:"location"`
	Database         string            `yaml:"database"`
	Precision        string            `yaml:"precision"`
	RetentionPolicy  string            `yaml:"retention_policy"`
	WriteConsistency string            `yaml:"write_consistency"`
	Measurement      string            `yaml:"measurement"`
	Tags             map[string]string `yaml:"tags"`
	Fields           []string          `yaml:"fields"`
	attrs            []vitotrol.AttrID
	computedFields   []string
}

type Config struct {
	Influx   ConfigInflux    `yaml:"influx"`
	Vitotrol ConfigVitotrol  `yaml:"vitotrol"`
	Devices  []*ConfigDevice `yaml:"devices"`
}

// ReadConfig load a YAML config file and returns its contents.
func ReadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}

// GetConfigDevice returns the ConfigDevice corresponding to the
// device name in location location.
func (c *Config) GetConfigDevice(name, location string) *ConfigDevice {
	for _, device := range c.Devices {
		if (device.Name == "*" || device.Name == name) &&
			(device.Location == "*" || device.Location == location) {
			return device
		}
	}
	return nil
}
