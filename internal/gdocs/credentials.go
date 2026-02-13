package gdocs

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	envGoogleType                    = "GOOGLE_TYPE"
	envGoogleProjectID               = "GOOGLE_PROJECT_ID"
	envGooglePrivateKeyID            = "GOOGLE_PRIVATE_KEY_ID"
	envGooglePrivateKey              = "GOOGLE_PRIVATE_KEY"
	envGoogleClientEmail             = "GOOGLE_CLIENT_EMAIL"
	envGoogleClientID                = "GOOGLE_CLIENT_ID"
	envGoogleAuthURI                 = "GOOGLE_AUTH_URI"
	envGoogleTokenURI                = "GOOGLE_TOKEN_URI"
	envGoogleAuthProviderX509CertURL = "GOOGLE_AUTH_PROVIDER_X509_CERT_URL"
	envGoogleClientX509CertURL       = "GOOGLE_CLIENT_X509_CERT_URL"
	envGoogleUniverseDomain          = "GOOGLE_UNIVERSE_DOMAIN"
)

// ServiceAccountCredentials represents Google service account credentials.
type ServiceAccountCredentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url,omitempty"`
	ClientX509CertURL       string `json:"client_x509_cert_url,omitempty"`
	UniverseDomain          string `json:"universe_domain,omitempty"`
}

func LoadCredentialsFromEnv() (*ServiceAccountCredentials, error) {
	creds := &ServiceAccountCredentials{}
	requiredFields := []struct {
		envKey string
		target *string
	}{
		{envKey: envGoogleType, target: &creds.Type},
		{envKey: envGoogleProjectID, target: &creds.ProjectID},
		{envKey: envGooglePrivateKeyID, target: &creds.PrivateKeyID},
		{envKey: envGooglePrivateKey, target: &creds.PrivateKey},
		{envKey: envGoogleClientEmail, target: &creds.ClientEmail},
		{envKey: envGoogleClientID, target: &creds.ClientID},
		{envKey: envGoogleAuthURI, target: &creds.AuthURI},
		{envKey: envGoogleTokenURI, target: &creds.TokenURI},
	}

	for _, field := range requiredFields {
		value, err := requiredEnv(field.envKey)
		if err != nil {
			return nil, err
		}
		*field.target = value
	}

	creds.PrivateKey = strings.ReplaceAll(creds.PrivateKey, `\n`, "\n")
	creds.AuthProviderX509CertURL = os.Getenv(envGoogleAuthProviderX509CertURL)
	creds.ClientX509CertURL = os.Getenv(envGoogleClientX509CertURL)
	creds.UniverseDomain = os.Getenv(envGoogleUniverseDomain)

	return creds, nil
}

func (c *ServiceAccountCredentials) JSON() ([]byte, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service account credentials: %w", err)
	}
	return payload, nil
}

func ValidateCredentialsEnv() error {
	_, err := LoadCredentialsFromEnv()
	return err
}

func requiredEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return value, nil
}
