package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnvFiles() error {
	for _, file := range []string{".env.local", ".env"} {
		if err := godotenv.Load(file); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("failed to load %s: %w", file, err)
		}
	}

	return nil
}
