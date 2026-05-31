package config

import (
	"os"
	"strconv"
)

type ServiceConfig struct {
	ServiceName       string
	HTTPAddr          string
	GRPCAddr          string
	DatabaseURL       string
	RedisAddr         string
	KafkaBrokers      string
	KafkaMaxRetries   int
	KafkaBackoffMs    int
	KafkaDLQSuffix    string
	OTLPEndpoint      string
	InventoryGRPCAddr string
	PaymentGRPCAddr   string
}

func LoadService(defaultName, defaultHTTPAddr, defaultGRPCAddr string) ServiceConfig {
	return ServiceConfig{
		ServiceName:       getEnv("SERVICE_NAME", defaultName),
		HTTPAddr:          getEnv("HTTP_ADDR", defaultHTTPAddr),
		GRPCAddr:          getEnv("GRPC_ADDR", defaultGRPCAddr),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers:      getEnv("KAFKA_BROKERS", "localhost:29092"),
		KafkaMaxRetries:   getEnvInt("KAFKA_MAX_RETRIES", 3),
		KafkaBackoffMs:    getEnvInt("KAFKA_BACKOFF_MS", 250),
		KafkaDLQSuffix:    getEnv("KAFKA_DLQ_SUFFIX", ".dlq"),
		OTLPEndpoint:      os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		InventoryGRPCAddr: getEnv("INVENTORY_GRPC_ADDR", "localhost:9001"),
		PaymentGRPCAddr:   getEnv("PAYMENT_GRPC_ADDR", "localhost:9002"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
