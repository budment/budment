package config

import (
	"os"
	"strconv"
)

func ParseEnv() EnvConfig {
	var env EnvConfig

	if v := os.Getenv("BLASTER_VUS"); v != "" {
		if vus, err := strconv.Atoi(v); err == nil {
			env.VUs = &vus
		}
	}
	if v := os.Getenv("BLASTER_DURATION"); v != "" {
		env.Duration = &v
	}
	if v := os.Getenv("BLASTER_MAX_DURATION"); v != "" {
		env.MaxDuration = &v
	}
	if v := os.Getenv("BLASTER_ITERATIONS"); v != "" {
		if iter, err := strconv.Atoi(v); err == nil && iter > 0 {
			env.Iterations = &iter
		}
	}
	if v := os.Getenv("BLASTER_START_AT"); v != "" {
		env.StartAt = &v
	}
	if v := os.Getenv("BLASTER_AUTO_PLUMB"); v != "" {
		if plumb, err := strconv.ParseBool(v); err == nil {
			env.AutoPlumb = &plumb
		}
	}
	if v := os.Getenv("BLASTER_INSECURE_SKIP_TLS_VERIFY"); v != "" {
		if skip, err := strconv.ParseBool(v); err == nil {
			env.InsecureSkipTLS = &skip
		}
	}
	if v := os.Getenv("BLASTER_HTTP_TIMEOUT"); v != "" {
		env.HTTPTimeout = &v
	}
	if v := os.Getenv("BLASTER_EXPORT_JSON"); v != "" {
		env.ExportJSON = &v
	}
	if v := os.Getenv("BLASTER_EXPORT_HTML"); v != "" {
		env.ExportHTML = &v
	}
	if v := os.Getenv("BLASTER_PROMETHEUS_OUT"); v != "" {
		env.PrometheusOut = &v
	}

	return env
}
