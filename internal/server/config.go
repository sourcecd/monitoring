package server

// ConfigArgs stores server config information.
type ConfigArgs struct {
	ServerAddr      string // server address
	Loglevel        string // level of logging
	StoreInterval   int    // periodic interval before save metrics data to file
	FileStoragePath string // path to file, where metrics will be store
	Restore         bool   // a flag that indicates whether to restore saved metrics from a file when starting the server
	DatabaseDsn     string // database connection string
	PprofAddr       string // address for pprof buildin server
	KeyEnc          string // symmetric encryption key for signing requests
}
