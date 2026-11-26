package config

import (
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func podNS(ns string) func() string { return func() string { return ns } }

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(env(map[string]string{
		"REPORT_URL":   "https://api.example.com/images",
		"REPORT_TOKEN": "t",
	}), podNS("pra"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Namespace != "pra" || cfg.HTTPTimeout != 30*time.Second || cfg.DryRun {
		t.Errorf("valeurs par défaut inattendues : %+v", cfg)
	}
}

func TestLoadTargetNamespaceOverridesPodNamespace(t *testing.T) {
	cfg, err := load(env(map[string]string{"TARGET_NAMESPACE": "pra-dev", "DRY_RUN": "true"}), podNS("informer"))
	if err != nil || cfg.Namespace != "pra-dev" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]map[string]string{
		"url manquante":         {"REPORT_TOKEN": "t"},
		"jeton manquant":        {"REPORT_URL": "https://api.example.com"},
		"http refusé":           {"REPORT_URL": "http://api.example.com", "REPORT_TOKEN": "t"},
		"url sans hôte":         {"REPORT_URL": "https://", "REPORT_TOKEN": "t"},
		"délai invalide":        {"REPORT_URL": "https://a.b", "REPORT_TOKEN": "t", "HTTP_TIMEOUT": "abc"},
		"booléen invalide":      {"DRY_RUN": "oui"},
		"namespace introuvable": {"DRY_RUN": "true"},
	}
	for name, m := range cases {
		ns := "pra"
		if name == "namespace introuvable" {
			ns = ""
		}
		if _, err := load(env(m), podNS(ns)); err == nil {
			t.Errorf("%s : erreur attendue", name)
		}
	}
}

func TestLoadAllowsHTTPWhenExplicit(t *testing.T) {
	_, err := load(env(map[string]string{
		"REPORT_URL": "http://localhost:8080", "REPORT_TOKEN": "t", "REPORT_ALLOW_INSECURE": "true",
	}), podNS("pra"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadDryRunNeedsNoURLNorToken(t *testing.T) {
	cfg, err := load(env(map[string]string{"DRY_RUN": "1"}), podNS("pra"))
	if err != nil || !cfg.DryRun {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}
