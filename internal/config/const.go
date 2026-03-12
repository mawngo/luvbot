package config

import (
	"github.com/mawngo/go-try/v2"
	"time"
)

const (
	DefaultProfile            = "default"
	ProfilesDirectory         = "profiles"
	ErrorScreenshotsDirectory = "errors"
)

var ElementRetryOpt = try.NewOptions(try.WithExponentialBackoff(2*time.Second, 10*time.Second), try.WithAttempts(5))
