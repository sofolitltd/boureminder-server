// Package config loads server configuration from the environment.
package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port          int    `env:"PORT" envDefault:"8080"`
	Env           string `env:"ENV" envDefault:"dev"`
	MinAppVersion string `env:"MIN_APP_VERSION" envDefault:"1.0.0"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`

	DatabaseURL    string `env:"DATABASE_URL,required"`
	DatabaseDirect string `env:"DATABASE_DIRECT_URL,required"`

	S3Endpoint  string `env:"AWS_ENDPOINT_URL_S3,required"`
	S3Region    string `env:"AWS_REGION" envDefault:"ap-southeast-1"`
	S3AccessKey string `env:"AWS_ACCESS_KEY_ID,required"`
	S3SecretKey string `env:"AWS_SECRET_ACCESS_KEY,required"`
	S3Bucket    string `env:"S3_BUCKET,required"`

	FirebaseProject          string `env:"FIREBASE_PROJECT_ID,required"`
	FirebaseCreds            string `env:"FIREBASE_CREDENTIALS_JSON"`
	FirebaseAuthEmulatorHost string `env:"FIREBASE_AUTH_EMULATOR_HOST"`

	SentryDSN string `env:"SENTRY_DSN"`
}

func Load() (Config, error) {
	return env.ParseAs[Config]()
}
