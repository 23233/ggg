package ut

import "os"

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) < 1 {
		return fallback
	}
	return value
}
