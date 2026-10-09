package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	Port      string
	DBURL     string
	JWTSecret string
	SeedDemo  bool
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	seed := false
	if v := os.Getenv("SEED_DEMO"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			slog.Warn("invalid SEED_DEMO, using false", "value", v)
		} else {
			seed = b
		}
	}
	return Config{
		Port:      port,
		DBURL:     os.Getenv("DATABASE_URL"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		SeedDemo:  seed,
	}
}
