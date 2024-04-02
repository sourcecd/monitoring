package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type (
	gauge   float64
	counter int64
)

type SysMon struct {
	PollCount   counter
	RandomValue gauge
}

type RtMonitorSensGauge []string

func parsedSysMetricsURL(serverHost string, randVal gauge, pollCount counter) []string {
	return []string{
		fmt.Sprintf("%s/update/gauge/randomvalue/%f", serverHost, randVal),
		fmt.Sprintf("%s/update/counter/pollcount/%d", serverHost, pollCount),
	}
}

func parsedRtMetricURL(serverHost, metricName string, val reflect.Value) string {
	return fmt.Sprintf("%s/update/gauge/%s/%v", serverHost, strings.ToLower(metricName), val)
}

func updateMetrics(memstat *runtime.MemStats, sysmetrics *SysMon, pollInterval int) {
	for {
		runtime.ReadMemStats(memstat)
		sysmetrics.PollCount += 1
		sysmetrics.RandomValue = gauge(rand.New(rand.NewSource(time.Now().UnixNano())).Float64())

		if pollInterval > 0 {
			time.Sleep(time.Duration(pollInterval) * time.Second)
		} else {
			break
		}
	}
}

func main() {

	serverHost := "http://localhost:8080"

	m := RtMonitorSensGauge{
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
	rtm := &runtime.MemStats{}
	sysMetrics := &SysMon{}
	pollInterval := 2    //sec
	reportInterval := 10 //sec

	go updateMetrics(rtm, sysMetrics, pollInterval)

	for {
		rtmVal := reflect.ValueOf(rtm).Elem()
		for i := 0; i < len(m); i++ {
			v := rtmVal.FieldByName(m[i])
			resp, err := http.Post(parsedRtMetricURL(serverHost, m[i], v), "text/plain", nil)
			if err != nil {
				log.Println(err)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				log.Printf("status_code: %d", resp.StatusCode)
				continue
			}
			_ = resp.Body.Close()
		}

		sysM := parsedSysMetricsURL(serverHost, sysMetrics.RandomValue, sysMetrics.PollCount)
		for _, s := range sysM {
			resp, err := http.Post(s, "text/plain", nil)
			if err != nil {
				log.Println(err)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				log.Printf("status_code: %d", resp.StatusCode)
				continue
			}
			_ = resp.Body.Close()
		}
		sysMetrics.PollCount = 0
		time.Sleep(time.Duration(reportInterval) * time.Second)
	}
}
