package config

import "time"

const (
	DefaultConfigMapName     = "pod-refresh-controller"
	DefaultPodExpirationTime = 24 * time.Hour
)

type Config struct {
	PodExpirationTime time.Duration
}

func NewDefaultConfig() *Config {
	return &Config{
		PodExpirationTime: DefaultPodExpirationTime,
	}
}
