package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

type sysMon struct {
	pollCount   metrictypes.Counter
	randomValue metrictypes.Gauge
}

func rtMonitorSensGauge() []string {
	return []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
		"GCCPUFraction",
		"NumForcedGC",
		"NumGC",
	}
}

func send(r *resty.Request, send, serverHost string) (*resty.Response, error) {
	resp, err := r.SetBody(send).Post(serverHost + "/updates/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}
	return resp, err
}

func updateMetrics(memstat *runtime.MemStats, sysmetrics *sysMon) {
	runtime.ReadMemStats(memstat)
	sysmetrics.pollCount += 1
	sysmetrics.randomValue = metrictypes.Gauge(rand.New(rand.NewSource(time.Now().UnixNano())).Float64())
}

func encodeJSON(metrics []models.Metrics) (string, error) {
	jRes, err := json.Marshal(metrics)
	return string(jRes), err
}

func Run(serverAddr string, reportInterval, pollInterval int) {
	serverHost := fmt.Sprintf("http://%s", serverAddr)

	m := rtMonitorSensGauge()
	rtm := &runtime.MemStats{}
	sysMetrics := &sysMon{}

	client := resty.New()
	r := client.R().SetHeader("Content-Type", "application/json")

	if reportInterval <= 0 || pollInterval <= 0 {
		log.Fatal("wrong intervals")
	}
	for {
		var batchMetrics []models.Metrics
		timeStart := time.Now().Unix()
		for {
			updateMetrics(rtm, sysMetrics)

			time.Sleep(time.Duration(pollInterval) * time.Second)
			if time.Now().Unix()-timeStart >= int64(reportInterval) {
				break
			}
		}

		rtmVal := reflect.ValueOf(rtm).Elem()
		for i := 0; i < len(m); i++ {
			v := rtmVal.FieldByName(m[i])
			fl64, err := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
			if err != nil {
				log.Println(err)
				continue
			}
			// resty framework automaticaly close Body
			// resty automatical send Accept-Encoding: gzip (can see it in server log)
			batchMetrics = append(batchMetrics, models.Metrics{
				ID:    m[i],
				MType: metrictypes.GaugeType,
				Value: &fl64,
			})
		}

		batchMetrics = append(batchMetrics,
			models.Metrics{
				ID:    "PollCount",
				MType: metrictypes.CounterType,
				Delta: (*int64)(&sysMetrics.pollCount),
			},
			models.Metrics{
				ID:    "RandomValue",
				MType: metrictypes.GaugeType,
				Value: (*float64)(&sysMetrics.randomValue),
			})

		strToSend, err := encodeJSON(batchMetrics)
		if err != nil {
			log.Println(err)
			continue
		}
		// resty framework automaticaly close Body
		// resty automatical send Accept-Encoding: gzip (can see it in server log)
		_, err = send(r, strToSend, serverHost)
		if err != nil {
			log.Println(err)
			continue
		}
		sysMetrics.pollCount = 0
	}
}
