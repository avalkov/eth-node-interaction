package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func NewConfig(envPath string) (Config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return Config{}, err
	}

	return Config{
		ApiPort:         getEnvAsInt("PORT", 31337),
		EthNodeUrl:      getEnv("ETH_NODE_URL", ""),
		DbConnectionUrl: getEnv("DB_CONNECTION_URL", ""),
	}, nil
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

type Config struct {
	ApiPort         int
	EthNodeUrl      string
	DbConnectionUrl string
}
