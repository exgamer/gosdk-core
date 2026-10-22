package context

import (
	"context"
	"testing"

	"github.com/exgamer/gosdk-core/pkg/config"
)

func TestWithAppInfo_ReturnsSamePointer(t *testing.T) {
	info := &config.AppInfo{ServiceName: "svc"}

	got := GetAppInfoFromContext(WithAppInfo(context.Background(), info))

	if got != info {
		t.Fatalf("GetAppInfoFromContext = %p, want %p", got, info)
	}
}

func TestGetAppInfoFromContext_IgnoresStringKey(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{name: "AppInfo", value: &config.AppInfo{ServiceName: "spoof"}},
		{name: "string", value: "str"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "app_info", c.value) //nolint:staticcheck // SA1029: проверка, что строковый ключ "app_info" больше не читается

			if got := GetAppInfoFromContext(ctx); got != nil {
				t.Fatalf("GetAppInfoFromContext = %+v, want nil", got)
			}
		})
	}
}

func TestGetAppInfoFromContext_Empty(t *testing.T) {
	var nilCtx context.Context

	cases := []struct {
		name string
		ctx  context.Context
	}{
		{name: "background", ctx: context.Background()},
		{name: "nil", ctx: nilCtx},
		{name: "nil AppInfo", ctx: WithAppInfo(context.Background(), nil)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetAppInfoFromContext(c.ctx); got != nil {
				t.Fatalf("GetAppInfoFromContext = %+v, want nil", got)
			}
		})
	}
}

func TestGetAppInfoFromContext_StringKeyDoesNotOverride(t *testing.T) {
	actual := &config.AppInfo{ServiceName: "real"}
	spoof := &config.AppInfo{ServiceName: "spoof"}

	ctx := WithAppInfo(context.Background(), actual)
	ctx = context.WithValue(ctx, "app_info", spoof) //nolint:staticcheck // SA1029: проверка, что строковый ключ "app_info" не перекрывает AppInfo

	if got := GetAppInfoFromContext(ctx); got != actual {
		t.Fatalf("GetAppInfoFromContext = %+v, want %+v", got, actual)
	}
}

func TestWithAppInfo_OuterOverridesInner(t *testing.T) {
	inner := &config.AppInfo{ServiceName: "inner"}
	outer := &config.AppInfo{ServiceName: "outer"}

	ctx := WithAppInfo(WithAppInfo(context.Background(), inner), outer)

	if got := GetAppInfoFromContext(ctx); got != outer {
		t.Fatalf("GetAppInfoFromContext = %+v, want %+v", got, outer)
	}
}
