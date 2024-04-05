package agent

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sourcecd/monitoring/internal/metrictypes"
)

type SysMon struct {
	PollCount   metrictypes.Counter
	RandomValue metrictypes.Gauge
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

func parsedSysMetricsURL(serverHost string, randVal metrictypes.Gauge, pollCount metrictypes.Counter) []string {
	return []string{
		fmt.Sprintf("%s/update/gauge/randomvalue/%f", serverHost, randVal),
		fmt.Sprintf("%s/update/counter/pollcount/%d", serverHost, pollCount),
	}
}

func parsedRtMetricURL(serverHost, metricName string, val reflect.Value) string {
	return fmt.Sprintf("%s/update/gauge/%s/%v", serverHost, strings.ToLower(metricName), val)
}

func updateMetrics(memstat *runtime.MemStats, sysmetrics *SysMon) {
	runtime.ReadMemStats(memstat)
	sysmetrics.PollCount += 1
	sysmetrics.RandomValue = metrictypes.Gauge(rand.New(rand.NewSource(time.Now().UnixNano())).Float64())
}

func Run(serverAddr string, reportInterval, pollInterval int) {
	serverHost := fmt.Sprintf("http://%s", serverAddr)

	m := rtMonitorSensGauge()
	rtm := &runtime.MemStats{}
	sysMetrics := &SysMon{}

	client := resty.New()
	r := client.R().SetHeader("Content-Type", "text/plain")

	if reportInterval > 0 && pollInterval > 0 && reportInterval > pollInterval {
		for {
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
				resp, err := r.Post(parsedRtMetricURL(serverHost, m[i], v))
				if err != nil {
					log.Println(err)
					continue
				}
				if resp.StatusCode() != http.StatusOK {
					log.Printf("status_code: %d", resp.StatusCode())
					continue
				}
			}

			sysM := parsedSysMetricsURL(serverHost, sysMetrics.RandomValue, sysMetrics.PollCount)
			for _, s := range sysM {
				resp, err := r.Post(s)
				if err != nil {
					log.Println(err)
					continue
				}
				if resp.StatusCode() != http.StatusOK {
					log.Printf("status_code: %d", resp.StatusCode())
					continue
				}
			}
			sysMetrics.PollCount = 0
		}
	} else {
		log.Fatal("wrong intervals")
	}
}
