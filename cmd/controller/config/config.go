package config

import (
	"os"
	strutil "strconv"
)

type Config struct {
	IngressClass string
	Listen       string
	Kubeconfig   string
	ResyncPeriod string
	Namespace    string
	Verbosity   int
}

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func EnvDefault(key string, def any) string {
	var defStr string

	switch v := def.(type) {
	case string:
		defStr = v
	case int:
		defStr = strutil.Itoa(v)
	case bool:
		defStr = strutil.FormatBool(v)
	default:
		defStr = def.(string)
	}

	if value := os.Getenv(key); value != "" {
		return value
	}
	return defStr
}
