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

const workers = 3

var rtMonitorSensGauge = []string{
	"Alloc", "BuckHashSys", "Frees", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse",
	"HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse",
	"MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "OtherSys", "PauseTotalNs",
	"StackInuse", "StackSys", "Sys", "TotalAlloc", "GCCPUFraction", "NumForcedGC", "NumGC",
}

type (
	sysMon struct {
		mx          sync.RWMutex
		pollCount   metrictypes.Counter
		randomValue metrictypes.Gauge
	}

	MemStats struct {
		mx sync.RWMutex
		runtime.MemStats
	}

	kernelMetrics struct {
		mx sync.RWMutex
		TotalMemory,
		FreeMemory metrictypes.Gauge
		CPUutilization []metrictypes.Gauge
	}

	jsonModelsMetrics struct {
		mx               sync.RWMutex
		jsonMetricsSlice []models.Metrics
	}
)

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

func updateMetrics(memstat *MemStats, sysmetrics *sysMon) {
	memstat.mx.Lock()
	defer memstat.mx.Unlock()

	runtime.ReadMemStats(&memstat.MemStats)

	sysmetrics.mx.Lock()
	defer sysmetrics.mx.Unlock()

	sysmetrics.pollCount += 1
	sysmetrics.randomValue = metrictypes.Gauge(rand.New(rand.NewSource(time.Now().UnixNano())).Float64())
}

func encodeJSON(jsMetrics *jsonModelsMetrics) (string, error) {
	jsMetrics.mx.RLock()
	defer jsMetrics.mx.RUnlock()

	jRes, err := json.Marshal(jsMetrics.jsonMetricsSlice)
	return string(jRes), err
}

func updateSysKernMetrics(m *kernelMetrics) {
	vmstat, _ := mem.VirtualMemory()
	cpuU, _ := cpu.Percent(time.Second, true)

	m.mx.Lock()
	defer m.mx.Unlock()

	m.TotalMemory = metrictypes.Gauge(vmstat.Total)
	m.FreeMemory = metrictypes.Gauge(vmstat.Free)
	for i, c := range cpuU {
		m.CPUutilization[i] = metrictypes.Gauge(c)
	}
}

func parseRtm(rtm *MemStats, targerRtm []string, jsonMetrics *jsonModelsMetrics, sysM *sysMon) {
	rtm.mx.RLock()
	defer rtm.mx.RUnlock()

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

	sysM.mx.RLock()
	defer sysM.mx.RUnlock()

	pollCount := sysM.pollCount
	randomValue := sysM.randomValue
	addJSONModel(jsonMetrics, "PollCount", metrictypes.CounterType, (*int64)(&pollCount), nil)
	addJSONModel(jsonMetrics, "RandomValue", metrictypes.GaugeType, nil, (*float64)(&randomValue))
}

func parseKernMetrics(km *kernelMetrics, j *jsonModelsMetrics) {
	km.mx.RLock()
	defer km.mx.RUnlock()

	TotalMemory := km.TotalMemory
	FreeMemory := km.FreeMemory
	addJSONModel(j, "TotalMemory", metrictypes.GaugeType, nil, (*float64)(&TotalMemory))
	addJSONModel(j, "FreeMemory", metrictypes.GaugeType, nil, (*float64)(&FreeMemory))

	for i, v := range km.CPUutilization {
		v := v
		addJSONModel(j, fmt.Sprintf("CPUutilization%d", i), metrictypes.GaugeType, nil, (*float64)(&v))
	}
}

func addJSONModel(g *jsonModelsMetrics, id, mtype string, delta *int64, value *float64) {
	g.mx.Lock()
	defer g.mx.Unlock()

	g.jsonMetricsSlice = append(g.jsonMetricsSlice, models.Metrics{
		ID:    id,
		MType: mtype,
		Delta: delta,
		Value: value,
	})
}

func worker(id int, jobs <-chan string, timeout time.Duration, serverHost string, keyenc string, r *resty.Request, errRes chan<- error) {
	for j := range jobs {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		backoff := retry.WithMaxRetries(3, retry.NewFibonacci(1*time.Second))

		err := retry.Do(ctx, backoff, func(ctx context.Context) error {
			if _, err := cryptandsign.SignNew(send, keyenc)(r, j, serverHost); err != nil {
				return retry.RetryableError(fmt.Errorf("retry failed: %s", err.Error()))
			}
			return nil
		})
		cancel()
		if err != nil {
			log.Printf("worker%d: %s", id, err.Error())
			errRes <- err
			continue
		}
		errRes <- err
		log.Printf("worker%d: job done", id)
	}
}

func Run(config ConfigArgs) {
	startCoordChan1 := make(chan struct{})
	startCoordChan2 := make(chan struct{})
	ratelimit := config.RateLimit
	serverHost := fmt.Sprintf("http://%s", config.ServerAddr)
	// ctx timeout per send
	timeout := 30 * time.Second
	cpuCount, _ := cpu.Counts(true)

	rtm := &MemStats{}
	sysMetrics := &sysMon{}
	kernelSysMetrics := &kernelMetrics{CPUutilization: make([]metrictypes.Gauge, cpuCount)}
	jsonMetricsModel := &jsonModelsMetrics{}

	jobsQueue := make(chan string, ratelimit)
	jobsErr := make(chan error, ratelimit)

	client := resty.New()
	r := client.R().SetHeader("Content-Type", "application/json")

	if config.ReportInterval <= 0 || config.PollInterval <= 0 {
		log.Fatal("wrong intervals")
	}

	// run workers
	for w := 1; w <= workers; w++ {
		go worker(w, jobsQueue, timeout, serverHost, config.KeyEnc, r, jobsErr)
	}

	// poll metrics
	go func() {
		open := true
		for {
			updateMetrics(rtm, sysMetrics)
			if open {
				close(startCoordChan1)
				open = false
			}
			time.Sleep(time.Duration(config.PollInterval) * time.Second)
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
			time.Sleep(time.Duration(config.PollInterval) * time.Second)
		}
	}()

	for {
		// parse runtime metrics
		<-startCoordChan1
		parseRtm(rtm, rtMonitorSensGauge, jsonMetricsModel, sysMetrics)
		// kern sys metrics
		<-startCoordChan2
		parseKernMetrics(kernelSysMetrics, jsonMetricsModel)

		// parse full json
		strToSend, err := encodeJSON(jsonMetricsModel)
		// clear on each iteration
		jsonMetricsModel.mx.Lock()
		jsonMetricsModel.jsonMetricsSlice = []models.Metrics{}
		jsonMetricsModel.mx.Unlock()
		if err != nil {
			log.Println(err)
			continue
		}

		jobsQueue <- strToSend

		if err := <-jobsErr; err != nil {
			continue
		}

		sysMetrics.mx.Lock()
		sysMetrics.pollCount = 0
		sysMetrics.mx.Unlock()

		// report interval
		time.Sleep(time.Duration(config.ReportInterval) * time.Second)
	}
}
