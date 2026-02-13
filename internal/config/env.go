package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnvFiles() error {
	for _, file := range []string{".env.local", ".env"} {
		if _, err := os.Stat(file); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to stat %s: %w", file, err)
		}

		if err := godotenv.Load(file); err != nil {
			return fmt.Errorf("failed to load %s: %w", file, err)
		}
	}

	return nil
}
