package server

// ConfigArgs stores server config information.
type ConfigArgs struct {
	DatabaseDsn     string `json:"database_dsn"`   // database connection string
	PprofAddr       string `json:"pprof_address"`  // address for pprof buildin server
	KeyEnc          string `json:"key_enc_sign"`   // symmetric encryption key for signing requests
	ServerAddr      string `json:"address"`        // server address
	Loglevel        string `json:"log_level"`      // level of logging
	FileStoragePath string `json:"store_file"`     // path to file, where metrics will be store
	PrivKeyFile     string `json:"crypto_key"`     // path to private key file for asymmetric encryption
	StoreInterval   int    `json:"store_interval"` // periodic interval before save metrics data to file
	Restore         bool   `json:"restore"`        // a flag that indicates whether to restore saved metrics from a file when starting the server
	TrustedSubnet	string `json:"trusted_subnet"` // allow connections from specified subnet
}
