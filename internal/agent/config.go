package agent

type ConfigArgs struct {
	ServerAddr     string
	ReportInterval int
	PollInterval   int
	KeyEnc         string
	RateLimit      int
}
