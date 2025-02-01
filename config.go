package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig  `yaml:"server"`
	API    APIConfig     `yaml:"api"`
	Models []ModelConfig `yaml:"models"`
}

type ServerConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Passcode string `yaml:"passcode"`
}

type APIConfig struct {
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
	Timeout int    `yaml:"timeout"`
}

type ModelConfig struct {
	Name        string         `yaml:"name"`
	String      string         `yaml:"string"`
	Width       int            `yaml:"width"`
	Height      int            `yaml:"height"`
	Steps       int            `yaml:"steps"`
	Cfgscale    float32        `yaml:"cfgscale"`
	Loras       string         `yaml:"loras,omitempty"`
	LoraWeights float32        `yaml:"loraweights,omitempty"`
	Options     map[string]any `yaml:",inline"`
}

func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}
