package config

import (
	"os"
	"time"
)

const (
	DefaultConfigMapName     = "pod-refresh-controller"
	DefaultPodExpirationTime = 24 * time.Hour
	DefaultResyncPeriod      = 1 * time.Minute
)

type Config struct {
	PodExpirationTime time.Duration
}

func NewDefaultConfig() *Config {
	return &Config{
		PodExpirationTime: DefaultPodExpirationTime,
	}
}

func GetConfigMapName() string {
	configMapName := os.Getenv("CONFIG_MAP_NAME")

	if configMapName == "" {
		return DefaultConfigMapName
	}

	return configMapName
}
