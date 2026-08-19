package config

import (
	"os"
	"strconv"
)

func ParseEnv() EnvConfig {
	var env EnvConfig

	if vusStr := os.Getenv("BLASTER_VUS"); vusStr != "" {
		if vus, err := strconv.Atoi(vusStr); err == nil {
			env.VUs = &vus
		}
	}

	if durStr := os.Getenv("BLASTER_DURATION"); durStr != "" {
		env.Duration = &durStr
	}

	if val := os.Getenv("BLASTER_MAX_DURATION"); val != "" {
		env.MaxDuration = &val
	}

	if val := os.Getenv("BLASTER_ITERATIONS"); val != "" {
		if iter, err := strconv.Atoi(val); err == nil && iter > 0 {
			env.Iterations = &iter
		}
	}

	if val := os.Getenv("BLASTER_START_AT"); val != "" {
		env.StartAt = &val
	}

	if val := os.Getenv("BLASTER_AUTO_PLUMB"); val != "" {
		if plumb, err := strconv.ParseBool(val); err == nil {
			env.AutoPlumb = &plumb
		}
	}

	if val := os.Getenv("BLASTER_INSECURE_SKIP_TLS_VERIFY"); val != "" {
		if skip, err := strconv.ParseBool(val); err == nil {
			env.InsecureSkipTLS = &skip
		}
	}

	return env
}
