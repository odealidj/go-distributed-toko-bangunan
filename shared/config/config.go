package config

import "os"

type ServiceConfig struct {
	ServiceName  string
	HTTPAddr     string
	GRPCAddr     string
	DatabaseURL  string
	RedisAddr    string
	KafkaBrokers string
	OTLPEndpoint string
}

func LoadService(defaultName, defaultHTTPAddr, defaultGRPCAddr string) ServiceConfig {
	return ServiceConfig{
		ServiceName:  getEnv("SERVICE_NAME", defaultName),
		HTTPAddr:     getEnv("HTTP_ADDR", defaultHTTPAddr),
		GRPCAddr:     getEnv("GRPC_ADDR", defaultGRPCAddr),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:29092"),
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
