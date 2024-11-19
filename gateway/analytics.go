package gateway

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	mathrand "math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TykTechnologies/tyk-pump/analytics"
	"github.com/TykTechnologies/tyk-pump/serializer"

	maxminddb "github.com/oschwald/maxminddb-golang"

	"github.com/TykTechnologies/tyk/config"
	"github.com/TykTechnologies/tyk/regexp"
	"github.com/TykTechnologies/tyk/storage"
)

const analyticsKeyName = "tyk-system-analytics"

const (
	recordsBufferFlushInterval       = 200 * time.Millisecond
	recordsBufferForcedFlushInterval = 1 * time.Second
)

func (gw *Gateway) initNormalisationPatterns() (pats config.NormaliseURLPatterns) {
	pats.UUIDs = regexp.MustCompile(`[0-9a-fA-F]{8}(-)?[0-9a-fA-F]{4}(-)?[0-9a-fA-F]{4}(-)?[0-9a-fA-F]{4}(-)?[0-9a-fA-F]{12}`)
	pats.ULIDs = regexp.MustCompile(`(?i)[0-7][0-9A-HJKMNP-TV-Z]{25}`)
	pats.IDs = regexp.MustCompile(`\/(\d+)`)

	for _, pattern := range gw.GetConfig().AnalyticsConfig.NormaliseUrls.Custom {
		if patRe, err := regexp.Compile(pattern); err != nil {
			log.Error("failed to compile custom pattern: ", err)
		} else {
			pats.Custom = append(pats.Custom, patRe)
		}
	}
	return
}

// RedisAnalyticsHandler will record analytics data to a redis back end
// as defined in the Config object
type RedisAnalyticsHandler struct {
	Store                       storage.AnalyticsHandler
	GeoIPDB                     *maxminddb.Reader
	globalConf                  config.Config
	recordsChan                 chan *analytics.AnalyticsRecord
	workerBufferSize            uint64
	shouldStop                  uint32
	poolWg                      sync.WaitGroup
	enableMultipleAnalyticsKeys bool
	Clean                       Purger
	Gw                          *Gateway `json:"-"`
	mu                          sync.Mutex
	analyticsSerializer         serializer.AnalyticsSerializer

	// testing purposes
	mockEnabled   bool
	mockRecordHit func(record *analytics.AnalyticsRecord)
}

func (r *RedisAnalyticsHandler) Init() {
	r.globalConf = r.Gw.GetConfig()
	if r.globalConf.AnalyticsConfig.EnableGeoIP {
		if db, err := maxminddb.Open(r.globalConf.AnalyticsConfig.GeoIPDBLocation); err != nil {
			log.Error("Failed to init GeoIP Database: ", err)
		} else {
			r.GeoIPDB = db
		}
	}

	r.Store.Connect()
	ps := r.Gw.GetConfig().AnalyticsConfig.PoolSize
	recordsBufferSize := r.globalConf.AnalyticsConfig.RecordsBufferSize

	r.workerBufferSize = recordsBufferSize / uint64(ps)
	log.WithField("workerBufferSize", r.workerBufferSize).Debug("Analytics pool worker buffer size")
	r.enableMultipleAnalyticsKeys = r.globalConf.AnalyticsConfig.EnableMultipleAnalyticsKeys
	r.analyticsSerializer = serializer.NewAnalyticsSerializer(r.globalConf.AnalyticsConfig.SerializerType)

	r.Start()
}

var poolSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "analytics_pool_size",
		Help: "Number of workers in the analytics pool",
	},
	[]string{},
)

var recordsBufferSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "analytics_record_buffer_size",
		Help: "Number of RecordsBufferSize in the analytics pool",
	},
	[]string{},
)

func init() {
	prometheus.MustRegister(poolSizeGauge)
	prometheus.MustRegister(recordsBufferSizeGauge)
}

// Start initialize the records channel and spawn the record workers
func (r *RedisAnalyticsHandler) Start() {
	r.recordsChan = make(chan *analytics.AnalyticsRecord, r.globalConf.AnalyticsConfig.RecordsBufferSize)
	atomic.SwapUint32(&r.shouldStop, 0)

	poolSizeGauge.WithLabelValues().Set(float64(r.Gw.GetConfig().AnalyticsConfig.PoolSize))
	recordsBufferSizeGauge.WithLabelValues().Set(float64(r.globalConf.AnalyticsConfig.RecordsBufferSize))

	for i := 0; i < r.Gw.GetConfig().AnalyticsConfig.PoolSize; i++ {
		r.poolWg.Add(1)
		go r.recordWorker()
	}
}

// Stop stops the analytics processing
func (r *RedisAnalyticsHandler) Stop() {
	// flag to stop sending records into channel
	atomic.SwapUint32(&r.shouldStop, 1)

	// close channel to stop workers
	r.mu.Lock()
	close(r.recordsChan)
	r.mu.Unlock()

	// wait for all workers to be done
	r.poolWg.Wait()
}

// Flush will stop the analytics processing and empty the analytics buffer and then re-init the workers again
func (r *RedisAnalyticsHandler) Flush() {
	r.Stop()

	r.Start()
}

var lockDurationHistogram = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "analytics_record_hit_lock_duration_seconds",
		Help:    "Duration of lock and send operations in RecordHit function",
		Buckets: []float64{0.000005, 0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
	},
	[]string{"status"},
)

var recordsChanSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "analytics_records_chan_size",
		Help: "Current size of the recordsChan in RedisAnalyticsHandler",
	},
	[]string{"status"},
)

func init() {
	prometheus.MustRegister(lockDurationHistogram)
	prometheus.MustRegister(recordsChanSizeGauge)
}

// RecordHit will store an analytics.Record in Redis
func (r *RedisAnalyticsHandler) RecordHit(record *analytics.AnalyticsRecord) error {
	if r.mockEnabled {
		r.mockRecordHit(record)
		return nil
	}

	// check if we should stop sending records 1st
	if atomic.LoadUint32(&r.shouldStop) > 0 {
		return nil
	}

	start := time.Now()
	// just send record to channel consumed by pool of workers
	// leave all data crunching and Redis I/O work for pool workers
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recordsChan <- record

	chanSize := len(r.recordsChan)
	recordsChanSizeGauge.WithLabelValues("success").Set(float64(chanSize))
	lockDurationHistogram.WithLabelValues("success").Observe(time.Since(start).Seconds())

	return nil
}

func (r *RedisAnalyticsHandler) recordWorker() {
	defer r.poolWg.Done()

	// this is buffer to send one pipelined command to redis
	// use r.recordsBufferSize as cap to reduce slice re-allocations
	recordsBuffer := make([][]byte, 0, r.workerBufferSize)
	mathrand.Seed(time.Now().Unix())

	// read records from channel and process
	lastSentTs := time.Now()
	for {
		analyticKey := analyticsKeyName
		if r.enableMultipleAnalyticsKeys {
			suffix := mathrand.Intn(10)
			analyticKey = fmt.Sprintf("%v_%v", analyticKey, suffix)
		}
		serliazerSuffix := r.analyticsSerializer.GetSuffix()
		analyticKey += serliazerSuffix

		readyToSend := false

		flushTimer := time.NewTimer(recordsBufferFlushInterval)

		select {
		case record, ok := <-r.recordsChan:
			if !flushTimer.Stop() {
				// if the timer has been stopped then read from the channel to avoid leak
				<-flushTimer.C
			}

			// check if channel was closed and it is time to exit from worker
			if !ok {
				// send what is left in buffer
				r.Store.AppendToSetPipelined(analyticKey, recordsBuffer)
				return
			}

			// we have new record - prepare it and add to buffer

			// If we are obfuscating API Keys, store the hashed representation (config check handled in hashing function)
			record.APIKey = storage.HashKey(record.APIKey, r.globalConf.HashKeys)

			if r.globalConf.SlaveOptions.UseRPC {
				// Extend tag list to include this data so wecan segment by node if necessary
				record.Tags = append(record.Tags, "tyk-hybrid-rpc")
			}

			if r.globalConf.DBAppConfOptions.NodeIsSegmented {
				// Extend tag list to include this data so we can segment by node if necessary
				record.Tags = append(record.Tags, r.globalConf.DBAppConfOptions.Tags...)
			}

			// Lets add some metadata
			if record.APIKey != "" {
				record.Tags = append(record.Tags, "key-"+record.APIKey)
			}

			if record.OrgID != "" {
				record.Tags = append(record.Tags, "org-"+record.OrgID)
			}

			record.Tags = append(record.Tags, "api-"+record.APIID)

			// fix paths in record as they might have omitted leading "/"
			if !strings.HasPrefix(record.Path, "/") {
				record.Path = "/" + record.Path
			}
			if !strings.HasPrefix(record.RawPath, "/") {
				record.RawPath = "/" + record.RawPath
			}

			if encoded, err := r.analyticsSerializer.Encode(record); err != nil {
				log.WithError(err).Error("Error encoding analytics data")
			} else {
				recordsBuffer = append(recordsBuffer, encoded)
			}

			// identify that buffer is ready to be sent
			readyToSend = uint64(len(recordsBuffer)) == r.workerBufferSize
		case <-flushTimer.C:
			// nothing was received for that period of time
			// anyways send whatever we have, don't hold data too long in buffer
			readyToSend = true
		}

		// send data to Redis and reset buffer
		if len(recordsBuffer) > 0 && (readyToSend || time.Since(lastSentTs) >= recordsBufferForcedFlushInterval) {
			r.Store.AppendToSetPipelined(analyticKey, recordsBuffer)
			recordsBuffer = recordsBuffer[:0]
			lastSentTs = time.Now()
		}
	}
}

func DurationToMillisecond(d time.Duration) float64 {
	return float64(d) / 1e6
}

func NormalisePath(a *analytics.AnalyticsRecord, globalConfig *config.Config) {

	if globalConfig.AnalyticsConfig.NormaliseUrls.NormaliseUUIDs {
		a.Path = globalConfig.AnalyticsConfig.NormaliseUrls.CompiledPatternSet.UUIDs.ReplaceAllString(a.Path, "{uuid}")
	}

	if globalConfig.AnalyticsConfig.NormaliseUrls.NormaliseULIDs {
		a.Path = globalConfig.AnalyticsConfig.NormaliseUrls.CompiledPatternSet.ULIDs.ReplaceAllString(a.Path, "{ulid}")
	}

	if globalConfig.AnalyticsConfig.NormaliseUrls.NormaliseNumbers {
		a.Path = globalConfig.AnalyticsConfig.NormaliseUrls.CompiledPatternSet.IDs.ReplaceAllString(a.Path, "/{id}")
	}

	for _, r := range globalConfig.AnalyticsConfig.NormaliseUrls.CompiledPatternSet.Custom {
		a.Path = r.ReplaceAllString(a.Path, "{var}")
	}
}
