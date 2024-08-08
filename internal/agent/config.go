package agent

// ConfigArgs stores agent config information.
type ConfigArgs struct {
	ServerAddr     string // destination server address
	KeyEnc         string // symmetric encryption key for signing requests
	PprofAddr      string // address for pprof buildin server
	PubKeyFile     string // path to public key file for asymmetric encryption
	ReportInterval int    // periodic interval between metric reporting to server
	PollInterval   int    // periodic interval between collecting metrics
	RateLimit      int    // number of requests sending to server at the same time
}
