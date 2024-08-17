// Package agent engine (and API) for sending monitoring metrics.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sethvargo/go-retry"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/sourcecd/monitoring/internal/cryptandsign"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

// Number of workers pool for sending metrics.
const workers = 3

// Sensors list for fetching monitoring metrics.
var rtMonitorSensGauge = []string{
	"Alloc", "BuckHashSys", "Frees", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse",
	"HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse",
	"MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "OtherSys", "PauseTotalNs",
	"StackInuse", "StackSys", "Sys", "TotalAlloc", "GCCPUFraction", "NumForcedGC", "NumGC",
}

type (
	// Type of system metrics includes interval of polling and some rand value.
	sysMon struct {
		sync.RWMutex
		pollCount   metrictypes.Counter
		randomValue metrictypes.Gauge
	}

	// MemStats type of memory metrics, fetched from runtime (local) library.
	MemStats struct {
		sync.RWMutex
		runtime.MemStats
	}

	// Type of base system metrics like CPU and Memory usage.
	kernelMetrics struct {
		CPUutilization []metrictypes.Gauge
		TotalMemory    metrictypes.Gauge
		FreeMemory     metrictypes.Gauge
		sync.RWMutex
	}

	// Type of collection gauge and counter metrics.
	jsonModelsMetrics struct {
		jsonMetricsSlice []models.Metrics
		sync.RWMutex
	}
)

func shutdownCatcher(ctx context.Context, msg string) bool {
	select {
	case <-ctx.Done():
		if msg != "" {
			log.Println(msg)
		}
		return true
	default:
	}
	return false
}

// send function for sending monitoring requests.
func send(r *resty.Request, send, serverHost string) (*resty.Response, error) {
	resp, err := r.SetBody(send).Post(serverHost + "/updates/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("ans: %d, %s", resp.StatusCode(), resp.Body())
	}
	return resp, err
}

// updateMetrics function for fetch runtime system metrics.
func updateMetrics(memstat *MemStats, sysmetrics *sysMon) {
	memstat.Lock()
	defer memstat.Unlock()

	runtime.ReadMemStats(&memstat.MemStats)

	sysmetrics.Lock()
	defer sysmetrics.Unlock()

	sysmetrics.pollCount += 1
	sysmetrics.randomValue = metrictypes.Gauge(rand.New(rand.NewSource(time.Now().UnixNano())).Float64())
}

// encodeJSON function for json metric encode.
func encodeJSON(jsMetrics *jsonModelsMetrics) (string, error) {
	jsMetrics.RLock()
	defer jsMetrics.RUnlock()

	jRes, err := json.Marshal(jsMetrics.jsonMetricsSlice)
	return string(jRes), err
}

// updateSysKernMetrics function for fetch cpu and memory metrics.
func updateSysKernMetrics(m *kernelMetrics) {
	vmstat, _ := mem.VirtualMemory()
	cpuU, _ := cpu.Percent(time.Second, true)

	m.Lock()
	defer m.Unlock()

	m.TotalMemory = metrictypes.Gauge(vmstat.Total)
	m.FreeMemory = metrictypes.Gauge(vmstat.Free)
	for i, c := range cpuU {
		m.CPUutilization[i] = metrictypes.Gauge(c)
	}
}

// parseRtm function for parse runtime metrics and format it to pre-json struct.
func parseRtm(rtm *MemStats, targerRtm []string, jsonMetrics *jsonModelsMetrics, sysM *sysMon) {
	rtm.RLock()
	defer rtm.RUnlock()

	// use reflect for update struct field by name
	rtmVal := reflect.ValueOf(rtm).Elem()
	for i := 0; i < len(targerRtm); i++ {
		v := rtmVal.FieldByName(targerRtm[i])
		fl64, err := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		if err != nil {
			log.Println(err)
			continue
		}

		addJSONModel(jsonMetrics, targerRtm[i], metrictypes.GaugeType, nil, &fl64)
	}

	sysM.RLock()
	defer sysM.RUnlock()

	pollCount := sysM.pollCount
	randomValue := sysM.randomValue
	addJSONModel(jsonMetrics, "PollCount", metrictypes.CounterType, (*int64)(&pollCount), nil)
	addJSONModel(jsonMetrics, "RandomValue", metrictypes.GaugeType, nil, (*float64)(&randomValue))
}

// parseKernMetrics function for parse cpu and memory metrics and format it to pre-json struct.
func parseKernMetrics(km *kernelMetrics, j *jsonModelsMetrics) {
	km.RLock()
	defer km.RUnlock()

	TotalMemory := km.TotalMemory
	FreeMemory := km.FreeMemory
	addJSONModel(j, "TotalMemory", metrictypes.GaugeType, nil, (*float64)(&TotalMemory))
	addJSONModel(j, "FreeMemory", metrictypes.GaugeType, nil, (*float64)(&FreeMemory))

	for i, v := range km.CPUutilization {
		v := v
		addJSONModel(j, fmt.Sprintf("CPUutilization%d", i), metrictypes.GaugeType, nil, (*float64)(&v))
	}
}

// addJSONModel function for collect parsed metrics to spectial metrics structure.
func addJSONModel(g *jsonModelsMetrics, id, mtype string, delta *int64, value *float64) {
	g.Lock()
	defer g.Unlock()

	g.jsonMetricsSlice = append(g.jsonMetricsSlice, models.Metrics{
		ID:    id,
		MType: mtype,
		Delta: delta,
		Value: value,
	})
}

// function for create parallel workers which send metrics to server.
func worker(ctx context.Context, id int, jobs <-chan string, timeout time.Duration, serverHost, keyenc, pubkeypath string, r *resty.Request, errRes chan<- error) {
	for j := range jobs {
		ctx2, cancel := context.WithTimeout(ctx, timeout)
		//lint:ignore SA9001 https://github.com/sourcecd/monitoring/pull/24#discussion_r1720019349
		defer cancel()
		backoff := retry.WithMaxRetries(3, retry.NewFibonacci(1*time.Second))

		// using retry and request sign function
		err := retry.Do(ctx2, backoff, func(ctx context.Context) error {
			if _, err := cryptandsign.AsymEncryptData(cryptandsign.SignNew(send, keyenc), pubkeypath)(r, j, serverHost); err != nil {
				return retry.RetryableError(fmt.Errorf("retry failed: %s", err.Error()))
			}
			return nil
		})
		cancel()
		if shutdownCatcher(ctx, "worker shutdown") {
			return
		}
		if err != nil {
			log.Printf("worker%d: %s", id, err.Error())
			errRes <- err
			continue
		}
		errRes <- err
		log.Printf("worker%d: job done", id)
	}
}

// Run main function for running agent engine.
func Run(ctx context.Context, config ConfigArgs) {
	reportInterval := time.Duration(config.ReportInterval) * time.Second
	pollInterval := time.Duration(config.PollInterval) * time.Second
	startCoordChan1 := make(chan struct{})
	startCoordChan2 := make(chan struct{})
	ratelimit := config.RateLimit
	serverHost := fmt.Sprintf("http://%s", config.ServerAddr)
	// ctx timeout per send
	timeout := 30 * time.Second
	cpuCount, _ := cpu.Counts(true)

	// init metrics structs
	rtm := &MemStats{}
	sysMetrics := &sysMon{}
	kernelSysMetrics := &kernelMetrics{CPUutilization: make([]metrictypes.Gauge, cpuCount)}
	jsonMetricsModel := &jsonModelsMetrics{}

	// init channels for workers
	jobsQueue := make(chan string, ratelimit)
	jobsErr := make(chan error, ratelimit)

	// init resty client
	client := resty.New()
	r := client.R().SetHeader("Content-Type", "application/json")

	if config.ReportInterval <= 0 || config.PollInterval <= 0 {
		log.Fatal("wrong intervals")
	}

	// run workers
	for w := 1; w <= workers; w++ {
		go worker(ctx, w, jobsQueue, timeout, serverHost, config.KeyEnc, config.PubKeyFile, r, jobsErr)
	}

	// poll runtime metrics
	go func() {
		open := true
		for {
			updateMetrics(rtm, sysMetrics)
			if open {
				close(startCoordChan1)
				open = false
			}
			time.Sleep(pollInterval)
		}
	}()

	// poll kernMetrics
	go func() {
		open := true
		for {
			updateSysKernMetrics(kernelSysMetrics)
			if open {
				close(startCoordChan2)
				open = false
			}
			time.Sleep(pollInterval)
		}
	}()

	for {
		// parse runtime metrics
		<-startCoordChan1
		parseRtm(rtm, rtMonitorSensGauge, jsonMetricsModel, sysMetrics)
		// parse kern sys metrics
		<-startCoordChan2
		parseKernMetrics(kernelSysMetrics, jsonMetricsModel)

		// parse full json
		strToSend, err := encodeJSON(jsonMetricsModel)
		// clear metrics structure on each iteration
		jsonMetricsModel.Lock()
		jsonMetricsModel.jsonMetricsSlice = []models.Metrics{}
		jsonMetricsModel.Unlock()
		if err != nil {
			log.Println(err)
			continue
		}

		// add metrics payload to send queue
		select {
		case jobsQueue <- strToSend:
		case <-ctx.Done():
		}

		// get worker errors
		select {
		case err := <-jobsErr:
			if err != nil {
				continue
			}
		case <-ctx.Done():
		}

		// reset polling counter when metrics send procedure is success
		sysMetrics.Lock()
		sysMetrics.pollCount = 0
		sysMetrics.Unlock()

		// catch ctx done
		if shutdownCatcher(ctx, "GraceFull shutdown") {
			return
		}
		// metrics report interval
		time.Sleep(reportInterval)
	}
}
