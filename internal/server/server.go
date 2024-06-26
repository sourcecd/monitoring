package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
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
	"golang.org/x/sync/errgroup"
)

// seconds
const serverShutdownTime = 5

type urlToMetric struct {
	metricType  string
	metricName  string
	metricValue string
}

type metricHandlers struct {
	ctx     context.Context
	storage storage.StoreMetrics
	rtr     *retr.Retr
}

func (mh *metricHandlers) updateMetrics() http.HandlerFunc {
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
			if err := mh.rtr.UseRetrWM(mh.storage.WriteMetric)(mh.ctx, metric.metricType, metric.metricName, metrictypes.Gauge(fl64)); err != nil {
				http.Error(resp, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
		case metrictypes.CounterType:
			i64, err := strconv.ParseInt(metric.metricValue, 10, 64)
			if err != nil {
				http.Error(resp, "can't parse counter metric", http.StatusBadRequest)
				return
			}
			if err := mh.rtr.UseRetrWM(mh.storage.WriteMetric)(mh.ctx, metric.metricType, metric.metricName, metrictypes.Counter(i64)); err != nil {
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

func (mh *metricHandlers) getMetrics() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "text/plain")
		mType := chi.URLParam(req, "type")
		mVal := chi.URLParam(req, "val")
		switch mType {
		case metrictypes.GaugeType:
			val, err := mh.rtr.UseRetrGetMetric(mh.storage.GetMetric)(mh.ctx, metrictypes.GaugeType, mVal)
			if err != nil {
				http.Error(resp, "gauge not found", http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		case metrictypes.CounterType:
			val, err := mh.rtr.UseRetrGetMetric(mh.storage.GetMetric)(mh.ctx, metrictypes.CounterType, mVal)
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

func (mh *metricHandlers) getAll() http.HandlerFunc {
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
		res, err := mh.rtr.UseRetrGetAllM(mh.storage.GetAllMetricsTxt)(mh.ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = tmpl.Execute(w, res)
	}
}

func (mh *metricHandlers) updateMetricsJSON() http.HandlerFunc {
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
			if err := mh.rtr.UseRetrWM(mh.storage.WriteMetric)(mh.ctx, resultParsedJSON.MType, resultParsedJSON.ID, metrictypes.Gauge(*resultParsedJSON.Value)); err != nil {
				log.Println(err)
				http.Error(w, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
		} else if resultParsedJSON.MType == metrictypes.CounterType && resultParsedJSON.Delta != nil && resultParsedJSON.ID != "" {
			if err := mh.rtr.UseRetrWM(mh.storage.WriteMetric)(mh.ctx, resultParsedJSON.MType, resultParsedJSON.ID, metrictypes.Counter(*resultParsedJSON.Delta)); err != nil {
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

func (mh *metricHandlers) getMetricsJSON() http.HandlerFunc {
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
		res, err := mh.rtr.UseRetrGetMetric(mh.storage.GetMetric)(mh.ctx, resultParsedJSON.MType, resultParsedJSON.ID)
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

func (mh *metricHandlers) updateBatchMetricsJSON() http.HandlerFunc {
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

		if err := mh.rtr.UseRetrWMB(mh.storage.WriteBatchMetrics)(mh.ctx, batchMettricsJSON); err != nil {
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

func (mh *metricHandlers) dbPing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(mh.ctx, mh.rtr.GetTimeoutCtx())
		defer cancel()
		if err := mh.storage.Ping(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	}
}

func chiRouter(mh *metricHandlers, keyenc string) chi.Router {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", logging.WriteLogging(compression.GzipCompDecomp(cryptandsign.SignCheck(mh.updateMetrics(), keyenc))))
	r.Get("/value/{type}/{val}", logging.WriteLogging(compression.GzipCompDecomp(mh.getMetrics())))
	r.Get("/", logging.WriteLogging(compression.GzipCompDecomp(mh.getAll())))

	//json
	r.Post("/update/", logging.WriteLogging(compression.GzipCompDecomp(cryptandsign.SignCheck(mh.updateMetricsJSON(), keyenc))))
	r.Post("/value/", logging.WriteLogging(compression.GzipCompDecomp(mh.getMetricsJSON())))
	r.Post("/updates/", logging.WriteLogging(compression.GzipCompDecomp(cryptandsign.SignCheck(mh.updateBatchMetricsJSON(), keyenc))))

	//ping
	r.Get("/ping", logging.WriteLogging(compression.GzipCompDecomp(mh.dbPing())))

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

func Run(ctx context.Context, config ConfigArgs) {
	if err := logging.Setup(config.Loglevel); err != nil {
		log.Fatal(err)
	}

	g, ctx := errgroup.WithContext(ctx)

	var store storage.StoreMetrics

	//retr
	rtr := retr.NewRetr()

	//main context timeout (default 60 sec)
	rtr.SetParams(1*time.Second, 30*time.Second, 3)

	if config.DatabaseDsn != "" {
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
		g.Go(func() error {
			<-ctx.Done()
			fmt.Println("Saving file")
			saveToFile(m, config.FileStoragePath, 0)
			fmt.Println("Exiting...")
			return nil
		})

		go saveToFile(m, config.FileStoragePath, config.StoreInterval)

		store = m
	}

	mh := &metricHandlers{
		ctx:     ctx,
		storage: store,
		rtr:     rtr,
	}

	srv := http.Server{
		Addr:    config.ServerAddr,
		Handler: chiRouter(mh, config.KeyEnc),
	}

	g.Go(func() error {
		logging.Log.Info("Starting server on", zap.String("address", config.ServerAddr))
		return srv.ListenAndServe()
	})

	g.Go(func() error {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTime*time.Second)
		defer cancel()

		return srv.Shutdown(ctx)
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			logging.Log.Info("Server successful shutdown")
			return
		}
		logging.Log.Fatal("Failed: ", zap.Error(err))
	}
}
