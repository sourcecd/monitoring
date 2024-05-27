package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sourcecd/monitoring/internal/compression"
	"github.com/sourcecd/monitoring/internal/cryptandsign"
	"github.com/sourcecd/monitoring/internal/logging"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
	"github.com/sourcecd/monitoring/internal/retr"
	"github.com/sourcecd/monitoring/internal/storage"
	"go.uber.org/zap"
)

type urlToMetric struct {
	metricType  string
	metricName  string
	metricValue string
}

func updateMetrics(ctx context.Context, storage storage.StoreMetrics, rtr *retr.Retr) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "" && req.Header.Get("Content-Type") != "text/plain" {
			http.Error(resp, fmt.Sprintf("wrong content type: %s", req.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}

		metric := urlToMetric{
			metricType:  chi.URLParam(req, "type"),
			metricName:  chi.URLParam(req, "name"),
			metricValue: chi.URLParam(req, "value"),
		}

		switch metric.metricType {
		case metrictypes.GaugeType:
			fl64, err := strconv.ParseFloat(metric.metricValue, 64)
			if err != nil {
				http.Error(resp, "can't parse gauge metric", http.StatusBadRequest)
				return
			}
			if err := rtr.UseRetrWM(storage.WriteMetric)(ctx, metric.metricType, metric.metricName, metrictypes.Gauge(fl64)); err != nil {
				http.Error(resp, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
		case metrictypes.CounterType:
			i64, err := strconv.ParseInt(metric.metricValue, 10, 64)
			if err != nil {
				http.Error(resp, "can't parse counter metric", http.StatusBadRequest)
				return
			}
			if err := rtr.UseRetrWM(storage.WriteMetric)(ctx, metric.metricType, metric.metricName, metrictypes.Counter(i64)); err != nil {
				http.Error(resp, "can't store counter metric", http.StatusInternalServerError)
				return
			}
		default:
			http.Error(resp, "metric_type not found", http.StatusBadRequest)
			return
		}

		resp.Header().Set("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte("OK"))
	}
}

func getMetrics(ctx context.Context, storage storage.StoreMetrics, rtr *retr.Retr) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "text/plain")
		mType := chi.URLParam(req, "type")
		mVal := chi.URLParam(req, "val")
		switch mType {
		case metrictypes.GaugeType:
			val, err := rtr.UseRetrGetMetric(storage.GetMetric)(ctx, metrictypes.GaugeType, mVal)
			if err != nil {
				http.Error(resp, "gauge not found", http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		case metrictypes.CounterType:
			val, err := rtr.UseRetrGetMetric(storage.GetMetric)(ctx, metrictypes.CounterType, mVal)
			if err != nil {
				http.Error(resp, "counter not found", http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		default:
			http.Error(resp, "metric_type not found", http.StatusBadRequest)
			return
		}
	}
}

func getAll(ctx context.Context, storage storage.StoreMetrics, rtr *retr.Retr) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		tmpl, _ := template.New("data").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="counters" content="width=device-width, initial-scale=1.0" />
	<title>Counters</title>
</head>
<body>
<pre>
{{ .}}
</pre>
</body>
</html>`)
		res, err := rtr.UseRetrGetAllM(storage.GetAllMetricsTxt)(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = tmpl.Execute(w, res)
	}
}

func updateMetricsJSON(ctx context.Context, storage storage.StoreMetrics, rtr *retr.Retr) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resultParsedJSON models.Metrics

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&resultParsedJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		if resultParsedJSON.MType == metrictypes.GaugeType && resultParsedJSON.Value != nil && resultParsedJSON.ID != "" {
			if err := rtr.UseRetrWM(storage.WriteMetric)(ctx, resultParsedJSON.MType, resultParsedJSON.ID, metrictypes.Gauge(*resultParsedJSON.Value)); err != nil {
				log.Println(err)
				http.Error(w, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
		} else if resultParsedJSON.MType == metrictypes.CounterType && resultParsedJSON.Delta != nil && resultParsedJSON.ID != "" {
			if err := rtr.UseRetrWM(storage.WriteMetric)(ctx, resultParsedJSON.MType, resultParsedJSON.ID, metrictypes.Counter(*resultParsedJSON.Delta)); err != nil {
				log.Println(err)
				http.Error(w, "can't store counter metric", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "bad metric type or no metric value or id is empty", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(&resultParsedJSON); err != nil {
			http.Error(w, "can't prepare json answer", http.StatusInternalServerError)
			return
		}
	}
}

func getMetricsJSON(ctx context.Context, storage storage.StoreMetrics, rtr *retr.Retr) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resultParsedJSON models.Metrics

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&resultParsedJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		// use test method (from mentor issue iter9)
		res, err := rtr.UseRetrGetMetric(storage.GetMetric)(ctx, resultParsedJSON.MType, resultParsedJSON.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if g, ok := res.(metrictypes.Gauge); ok {
			resultParsedJSON.Value = (*float64)(&g)
		} else if c, ok := res.(metrictypes.Counter); ok {
			resultParsedJSON.Delta = (*int64)(&c)
		} else {
			http.Error(w, "bad metric type", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(&resultParsedJSON); err != nil {
			http.Error(w, "can't encode json", http.StatusInternalServerError)
			return
		}
	}
}

func updateBatchMetricsJSON(ctx context.Context, storage storage.StoreMetrics, rtr *retr.Retr) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var batchMettricsJSON []models.Metrics

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&batchMettricsJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		if err := rtr.UseRetrWMB(storage.WriteBatchMetrics)(ctx, batchMettricsJSON); err != nil {
			log.Println(err)
			http.Error(w, "error to store batch metrics", http.StatusInternalServerError)
			return
		}
		// check ref
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(batchMettricsJSON); err != nil {
			http.Error(w, "can't encode json", http.StatusInternalServerError)
			return
		}
	}
}

func dbPing(ctx context.Context, storage storage.StoreMetrics, rtr *retr.Retr) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(ctx, rtr.GetTimeoutCtx())
		defer cancel()
		if err := storage.Ping(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	}
}

func chiRouter(ctx context.Context, storage storage.StoreMetrics, keyenc string, rtr *retr.Retr) chi.Router {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", logging.WriteLogging(compression.GzipCompDecomp(cryptandsign.SignCheck(updateMetrics(ctx, storage, rtr), keyenc))))
	r.Get("/value/{type}/{val}", logging.WriteLogging(compression.GzipCompDecomp(getMetrics(ctx, storage, rtr))))
	r.Get("/", logging.WriteLogging(compression.GzipCompDecomp(getAll(ctx, storage, rtr))))

	//json
	r.Post("/update/", logging.WriteLogging(compression.GzipCompDecomp(cryptandsign.SignCheck(updateMetricsJSON(ctx, storage, rtr), keyenc))))
	r.Post("/value/", logging.WriteLogging(compression.GzipCompDecomp(getMetricsJSON(ctx, storage, rtr))))
	r.Post("/updates/", logging.WriteLogging(compression.GzipCompDecomp(cryptandsign.SignCheck(updateBatchMetricsJSON(ctx, storage, rtr), keyenc))))

	//ping
	r.Get("/ping", logging.WriteLogging(compression.GzipCompDecomp(dbPing(ctx, storage, rtr))))

	return r
}

func saveToFile(m *storage.MemStorage, fname string, duration int) {
	for {
		time.Sleep(time.Second * time.Duration(duration))
		if err := m.SaveToFile(fname); err != nil {
			log.Println(err)
		}
		if duration == 0 {
			break
		}
	}
}

func Run(config ConfigArgs, sigs chan os.Signal) {
	if err := logging.Setup(config.Loglevel); err != nil {
		log.Fatal(err)
	}

	var store storage.StoreMetrics

	//retr context
	ctx := context.Background()
	rtr := retr.NewRetr()

	//main context timeout (default 60 sec)
	rtr.SetParams(1*time.Second, 30*time.Second, 3)

	if config.DatabaseDsn != "" {
		signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		//PGDB new
		pgdb, err := storage.NewPgDB(config.DatabaseDsn)
		if err != nil {
			log.Fatal(err)
		}
		defer pgdb.CloseDB()

		if err := rtr.UseRetrPopDB(pgdb.PopulateDB)(ctx); err != nil {
			log.Fatal(err)
		}

		store = pgdb
	} else {
		m := storage.NewMemStorage()

		if config.Restore {
			if err := m.ReadFromFile(config.FileStoragePath); err != nil {
				log.Println(err)
			}
		}

		//save result on shutdown and throw signal
		go func() {
			sig := <-sigs
			fmt.Println("saving")
			saveToFile(m, config.FileStoragePath, 0)
			fmt.Println(sig)
			signal.Reset(syscall.SIGINT, syscall.SIGTERM)
			pid := os.Getpid()
			p, err := os.FindProcess(pid)
			if err != nil {
				log.Fatal(err)
			}
			if err := p.Signal(sig); err != nil {
				log.Fatal(err)
			}
		}()

		go saveToFile(m, config.FileStoragePath, config.StoreInterval)

		store = m
	}

	logging.Log.Info("Starting server on", zap.String("address", config.ServerAddr))
	logging.Log.Fatal("Failed to start server", zap.Error(http.ListenAndServe(config.ServerAddr, chiRouter(ctx, store, config.KeyEnc, rtr))))
}
