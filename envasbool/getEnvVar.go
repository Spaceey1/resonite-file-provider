package envasbool

import (
	"os"
	"strconv"
)

func GetEnvAsBool(name string, defaultVal bool) bool {
	valStr := os.Getenv(name)
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}

	return defaultVal
}