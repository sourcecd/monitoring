package agent

// ConfigArgs stores agent config information.
type ConfigArgs struct {
	ServerAddr     string `json:"address"`         // destination server address
	KeyEnc         string `json:"key_enc_sign"`    // symmetric encryption key for signing requests
	PprofAddr      string `json:"pprof_address"`   // address for pprof buildin server
	PubKeyFile     string `json:"crypto_key"`      // path to public key file for asymmetric encryption
	ReportInterval int    `json:"report_interval"` // periodic interval between metric reporting to server
	PollInterval   int    `json:"poll_interval"`   // periodic interval between collecting metrics
	RateLimit      int    `json:"rate_limit"`      // number of requests sending to server at the same time
}
