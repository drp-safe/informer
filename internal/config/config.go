// Package config lit la configuration du programme depuis l'environnement.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// serviceAccountNamespaceFile contient le namespace du pod courant.
const serviceAccountNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// Config regroupe les réglages du programme.
type Config struct {
	Namespace   string
	ReportURL   string
	ReportToken string
	HTTPTimeout time.Duration
	DryRun      bool
}

// Load lit la configuration avec os.Getenv.
func Load() (Config, error) {
	return load(os.Getenv, readNamespaceFile)
}

func readNamespaceFile() string {
	b, err := os.ReadFile(serviceAccountNamespaceFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func load(getenv func(string) string, podNamespace func() string) (Config, error) {
	cfg := Config{
		Namespace:   strings.TrimSpace(getenv("TARGET_NAMESPACE")),
		ReportURL:   strings.TrimSpace(getenv("REPORT_URL")),
		ReportToken: strings.TrimSpace(getenv("REPORT_TOKEN")),
		HTTPTimeout: 30 * time.Second,
	}

	var err error
	if cfg.DryRun, err = boolEnv(getenv, "DRY_RUN"); err != nil {
		return cfg, err
	}
	allowInsecure, err := boolEnv(getenv, "REPORT_ALLOW_INSECURE")
	if err != nil {
		return cfg, err
	}
	if v := getenv("HTTP_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return cfg, fmt.Errorf("HTTP_TIMEOUT invalide %q : durée attendue, ex. 30s", v)
		}
		cfg.HTTPTimeout = d
	}

	if cfg.Namespace == "" {
		cfg.Namespace = podNamespace()
	}
	if cfg.Namespace == "" {
		return cfg, errors.New("TARGET_NAMESPACE absent et namespace du pod illisible")
	}

	if cfg.DryRun {
		return cfg, nil
	}

	if cfg.ReportURL == "" {
		return cfg, errors.New("REPORT_URL obligatoire (ou DRY_RUN=true)")
	}
	u, err := url.Parse(cfg.ReportURL)
	if err != nil || u.Host == "" {
		return cfg, fmt.Errorf("REPORT_URL invalide : %q", cfg.ReportURL)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && allowInsecure) {
		return cfg, fmt.Errorf("REPORT_URL doit être en https (REPORT_ALLOW_INSECURE=true pour un test local)")
	}
	if cfg.ReportToken == "" {
		return cfg, errors.New("REPORT_TOKEN obligatoire")
	}
	return cfg, nil
}

func boolEnv(getenv func(string) string, name string) (bool, error) {
	v := getenv(name)
	if v == "" {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s invalide %q : true ou false attendu", name, v)
	}
	return b, nil
}
