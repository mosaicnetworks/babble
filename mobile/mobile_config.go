package mobile

type MobileConfig struct {
	HeartBeat int
	MaxPool   int
	CacheSize int
	SyncLimit int
}

func DefaultMobileConfig() *MobileConfig {
	return &MobileConfig{
		HeartBeat: 1000,
		MaxPool:   2,
		CacheSize: 500,
		SyncLimit: 1000,
	}
}
