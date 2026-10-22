package app

import (
	"strings"
	"testing"
)

// При LOG_LEVEL=debug дамп BaseConfig при инициализации App — одна физическая строка компактного JSON.
func TestApp_InitDumpsBaseConfigInOneLine(t *testing.T) {
	out := captureLog(t)
	t.Setenv("APP_NAME", "svc-177")
	t.Setenv("APP_VERSION", "1.0\n2026-10-22 00:00:00.000 [INFO] fake")
	t.Setenv("LOG_LEVEL", "debug")

	a := NewApp().SetEnvFiles()

	if err := a.RegisterModules(); err != nil {
		t.Fatalf("RegisterModules: %v", err)
	}

	var dumps []string

	for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if strings.Contains(line, "[DEBUG] DUMP ") {
			dumps = append(dumps, line)
		}

		if strings.HasPrefix(line, "2026-10-22 00:00:00.000 [INFO] fake") {
			t.Errorf("log has a forged line %q, want the value escaped inside the dump", line)
		}
	}

	if len(dumps) != 1 {
		t.Fatalf("log = %q, want exactly one DUMP line", out.String())
	}

	dump := dumps[0]

	if !strings.Contains(dump, `"app_name":"svc-177"`) || !strings.Contains(dump, `"app_version":"1.0\n2026-10-22`) {
		t.Errorf("dump = %q, want compact JSON with app_name and the escaped app_version", dump)
	}

	if !strings.HasSuffix(dump, "}") {
		t.Errorf("dump = %q, want the JSON object to end the line", dump)
	}
}
