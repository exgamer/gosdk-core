package app

import (
	"bytes"
	"context"
	stdlog "log"
	"os"
	"strings"
	"testing"

	coreContext "github.com/exgamer/gosdk-core/pkg/context"
	"github.com/exgamer/gosdk-core/pkg/logger"
)

func TestApp_InitPutsAppInfoUnderOwnKey(t *testing.T) {
	out := captureLog(t)
	t.Setenv("APP_NAME", "svc-177")
	t.Setenv("LOG_LEVEL", "info")

	a := NewApp().SetEnvFiles()

	if err := a.RegisterModules(); err != nil {
		t.Fatalf("RegisterModules: %v", err)
	}

	info := coreContext.GetAppInfoFromContext(a.GetContext())
	if info == nil {
		t.Fatal("GetAppInfoFromContext = nil, want AppInfo")
	}

	if info.ServiceName != "svc-177" {
		t.Fatalf("ServiceName = %q, want %q", info.ServiceName, "svc-177")
	}

	if v := a.GetContext().Value("app_info"); v != nil {
		t.Fatalf(`Value("app_info") = %+v, want nil`, v)
	}

	logger.Info(a.GetContext(), "x")

	if !strings.Contains(out.String(), "[INFO,svc-177] x") {
		t.Fatalf("log = %q, want [INFO,svc-177] x", out.String())
	}
}

func TestLogger_NilContextWithoutServiceName(t *testing.T) {
	out := captureLog(t)
	logger.SetLevel(logger.LevelInfo)

	var ctx context.Context

	logger.Info(ctx, "x")

	if !strings.Contains(out.String(), "[INFO] x") {
		t.Fatalf("log = %q, want [INFO] x", out.String())
	}
}

// captureLog Перенаправляет логгер в буфер; логгер и уровень восстанавливаются после теста.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	prevLevel := logger.GetLevel()
	t.Cleanup(func() {
		logger.SetLogger(stdlog.New(os.Stdout, "", 0))
		logger.SetLevel(prevLevel)
	})

	var out bytes.Buffer
	logger.SetLogger(stdlog.New(&out, "", 0))

	return &out
}
