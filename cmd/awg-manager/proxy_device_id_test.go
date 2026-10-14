package main

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"
)

type sysGetter struct {
	hostname string
	err      error
	hang     bool // ждать отмены ctx, как зависший RCI
}

func (g sysGetter) Get(ctx context.Context, path string, dst any) error {
	if g.hang {
		<-ctx.Done()
		return ctx.Err()
	}
	if g.err != nil {
		return g.err
	}
	if path != "/show/system" {
		return errors.New("unexpected path " + path)
	}
	b, _ := json.Marshal(map[string]string{"hostname": g.hostname})
	return json.Unmarshal(b, dst)
}

// F138: device-id нового wdtt-клиента — hostname роутера; иначе случайный.
func TestProxyDefaultDeviceID(t *testing.T) {
	random := regexp.MustCompile(`^[0-9a-f]{16}$`)
	if got := proxyDefaultDeviceID(context.Background(), sysGetter{hostname: " Ultramarina "}); got != "Ultramarina" {
		t.Fatalf("hostname: %q", got)
	}
	for name, g := range map[string]sysGetter{
		"пустой":         {hostname: ""},
		"с пробелом":     {hostname: "my router"},
		"RCI не ответил": {err: errors.New("rci down")},
	} {
		if got := proxyDefaultDeviceID(context.Background(), g); !random.MatchString(got) {
			t.Fatalf("%s: ждали случайный id, got %q", name, got)
		}
	}
	// Зависший RCI: ответ за разумное время, случайный id.
	start := time.Now()
	if got := proxyDefaultDeviceID(context.Background(), sysGetter{hang: true}); !random.MatchString(got) {
		t.Fatalf("hang: ждали случайный id, got %q", got)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("зависший RCI держал %v", d)
	}
	a := proxyDefaultDeviceID(context.Background(), sysGetter{})
	b := proxyDefaultDeviceID(context.Background(), sysGetter{})
	if a == b {
		t.Fatal("случайные id совпали")
	}
}
