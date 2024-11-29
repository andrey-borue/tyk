package prom_monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

var PoolSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "analytics_pool_size",
		Help: "Number of workers in the analytics pool",
	},
	[]string{},
)

var WorkerBufferSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "worker_buffer_Size",
		Help: "Analytics pool worker buffer size",
	},
	[]string{},
)

var RecordsBufferSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "analytics_record_buffer_size",
		Help: "Number of RecordsBufferSize in the analytics pool",
	},
	[]string{},
)

var LockDurationHistogram = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "analytics_record_hit_lock_duration_seconds",
		Help:    "Duration of lock and send operations in RecordHit function",
		Buckets: cfg.BucketRecordHitLockDuration,
	},
	[]string{"status"},
)

var RecordsChanSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "analytics_records_chan_size",
		Help: "Current size of the recordsChan in RedisAnalyticsHandler",
	},
	[]string{"status"},
)

var AppendToSetDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "redis_append_to_set_duration_seconds",
		Help:    "Duration of the AppendToSetPipelined function",
		Buckets: cfg.BucketRedisAppendToSetDuration,
	},
	[]string{"status"},
)

var ResponseCodeCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_response_code_total",
		Help: "Total number of HTTP responses, classified by response code",
	},
	[]string{"response_code", "sent_to_redis"},
)

func init() {
	prometheus.MustRegister(AppendToSetDuration)
	prometheus.MustRegister(WorkerBufferSizeGauge)
	prometheus.MustRegister(PoolSizeGauge)
	prometheus.MustRegister(RecordsBufferSizeGauge)
	prometheus.MustRegister(LockDurationHistogram)
	prometheus.MustRegister(RecordsChanSizeGauge)
	prometheus.MustRegister(ResponseCodeCounter)
}

func SetGauge(gauge *prometheus.GaugeVec, labels []string, value float64) {
	if !cfg.Enabled {
		return
	}
	gauge.WithLabelValues(labels...).Set(value)
}

func IncrementCounter(counter *prometheus.CounterVec, labels []string) {
	if !cfg.Enabled {
		return
	}
	counter.WithLabelValues(labels...).Inc()
}

func ObserveHistogram(histogram *prometheus.HistogramVec, labels []string, value float64) {
	if !cfg.Enabled {
		return
	}
	histogram.WithLabelValues(labels...).Observe(value)
}
