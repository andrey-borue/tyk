package prom_monitoring

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

var cfg = Config{}

func init() {
	if err := env.Parse(&cfg); err != nil {
		fmt.Println("Error loading monitoring config")
		fmt.Print(err)
		return
	}
	fmt.Println("Monitoring config loaded")
}

type Config struct {
	Endpoint                       string    `env:"MONITORING_ENDPOINT" envDefault:"/metrics"`
	HostPort                       string    `env:"MONITORING_HOST_PORT" envDefault:":8005"`
	BucketRedisAppendToSetDuration []float64 `env:"MONITORING_BUCKET_REDIS_APPEND_TO_SET_DURATION" envDefault:"0.00005,0.0001,0.0005,0.001,0.005,0.01,0.05,0.1,0.2,0.4,0.6,0.8,1,2"`
	BucketRecordHitLockDuration    []float64 `env:"MONITORING_BUCKET_RECORD_HIT_LOCK_DURATION" envDefault:"0.00005,0.0001,0.0005,0.001,0.005,0.01,0.05,0.1,0.2,0.4,0.6,0.8,1,2"`
}
