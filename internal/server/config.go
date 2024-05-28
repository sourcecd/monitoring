package server

type ConfigArgs struct {
	ServerAddr      string
	Loglevel        string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DatabaseDsn     string
	KeyEnc          string
}
