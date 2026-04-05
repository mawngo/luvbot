package config

import (
	"context"
	"github.com/mawngo/go-try/v2"
	"log/slog"
	"time"
)

const (
	DefaultProfile            = "default"
	ProfilesDirectory         = "profiles"
	ErrorScreenshotsDirectory = "errors"
)

var ElementRetryOpt = try.NewOptions(try.WithExponentialBackoff(2*time.Second, 10*time.Second), try.WithAttempts(5))

func IsDebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}
