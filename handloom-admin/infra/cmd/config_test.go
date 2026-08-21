package main

import (
	"strings"
	"testing"
)

func TestEnvConfigs_AllValidate(t *testing.T) {
	for env, cfg := range envConfigs {
		if err := cfg.validate(env); err != nil {
			t.Errorf("env %q does not validate: %v", env, err)
		}
	}
}

func TestProdConfig_HasNoNonProductionHosts(t *testing.T) {
	prod, ok := envConfigs["prod"]
	if !ok {
		t.Fatal("no prod config")
	}

	for _, field := range []struct {
		name string
		val  string
	}{
		{"PhonePeBaseURL", prod.PhonePeBaseURL},
		{"PhonePeAuthBaseURL", prod.PhonePeAuthBaseURL},
		{"MSG91BaseURL", prod.MSG91BaseURL},
		{"PhonePeCallbackURL", prod.PhonePeCallbackURL},
		{"PhonePeRedirectURL", prod.PhonePeRedirectURL},
	} {
		lower := strings.ToLower(field.val)
		for _, marker := range []string{"preprod", "sandbox", "dev-", "-dev."} {
			if strings.Contains(lower, marker) {
				t.Errorf("prod %s = %q contains %q", field.name, field.val, marker)
			}
		}
	}
}
