package server

// ConfigArgs stores server config information.
type ConfigArgs struct {
	DatabaseDsn     string // database connection string
	PprofAddr       string // address for pprof buildin server
	KeyEnc          string // symmetric encryption key for signing requests
	ServerAddr      string // server address
	Loglevel        string // level of logging
	FileStoragePath string // path to file, where metrics will be store
	PrivKeyFile     string // path to private key file for asymmetric encryption
	StoreInterval   int    // periodic interval before save metrics data to file
	Restore         bool   // a flag that indicates whether to restore saved metrics from a file when starting the server
}
