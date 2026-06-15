package runtime

import (
	"time"

	"github.com/ntpoppe/fuse/internal/config"
)

const (
	demoMaxBodyBytes  = 64 << 10 // 64 KiB
	prodMaxBodyBytes  = 1 << 20  // 1 MiB
	demoQueryTimeout   = 10 * time.Second
	demoRatePerSecond  = 5
	demoRateBurst      = 10
)

type RateLimit struct {
	RequestsPerSecond float64
	Burst             int
}

type HTTPProfile struct {
	MaxBodyBytes           int64
	QueryTimeout           time.Duration
	CORSOrigins            []string
	RateLimit              RateLimit
	AllowConnectionChanges bool
}

type Profile struct {
	MaxQueryRows int
	HTTP         HTTPProfile
}

func FromConfig(cfg *config.Config) Profile {
	if cfg.DemoMode {
		return demoProfile(cfg)
	}
	return productionProfile(cfg)
}

func demoProfile(cfg *config.Config) Profile {
	return Profile{
		MaxQueryRows: cfg.MaxQueryRows,
		HTTP: HTTPProfile{
			MaxBodyBytes:           demoMaxBodyBytes,
			QueryTimeout:           demoQueryTimeout,
			CORSOrigins:            cfg.CORSOrigins,
			RateLimit:              RateLimit{RequestsPerSecond: demoRatePerSecond, Burst: demoRateBurst},
			AllowConnectionChanges: false,
		},
	}
}

func productionProfile(cfg *config.Config) Profile {
	return Profile{
		MaxQueryRows: cfg.MaxQueryRows,
		HTTP: HTTPProfile{
			MaxBodyBytes:           prodMaxBodyBytes,
			QueryTimeout:           0,
			CORSOrigins:            cfg.CORSOrigins,
			RateLimit:              RateLimit{},
			AllowConnectionChanges: true,
		},
	}
}
