package installer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// opkgTimeout bounds the opkg calls below: they may run with a background
// context during boot migration, and a wedged opkg (stale lock, hung wget
// child holding our pipe) must not hang the migration forever.
const opkgTimeout = 2 * time.Minute

// Lifecycle is what the installer needs from the running sing-box
// daemon to perform a migration. *singbox.Operator satisfies this via
// a thin adapter in main.go (avoids a circular import here).
type Lifecycle interface {
	Stop(ctx context.Context) error
	Start(ctx context.Context) error
}

// defaultOpkgListInstalled invokes `opkg list-installed`. Overridable
// via Installer.opkgListInstalled for tests.
var defaultOpkgListInstalled = func(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, opkgTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "opkg", "list-installed")
	cmd.WaitDelay = 5 * time.Second
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// IsLegacyOpkgInstalled returns true when `sing-box-naive` is recorded
// in the opkg database. False on systems without opkg, on errors, or
// when the package is absent.
func (i *Installer) IsLegacyOpkgInstalled(ctx context.Context) bool {
	list := i.opkgListInstalled
	if list == nil {
		list = defaultOpkgListInstalled
	}
	out, err := list(ctx)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "sing-box-naive ") {
			return true
		}
	}
	return false
}

// defaultOpkgRemove invokes `opkg remove sing-box-naive`. Overridable
// via Installer.opkgRemove for tests.
var defaultOpkgRemove = func(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, opkgTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "opkg", "remove", "sing-box-naive")
	cmd.WaitDelay = 5 * time.Second
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("opkg remove sing-box-naive: %s: %w", string(out), err)
	}
	return nil
}

// Migrate orchestrates the legacy → managed transition. Sequence:
// download+verify → stop → opkg-remove → activate → start.
// (Download() does SHA256 verification before returning — there is no
// separate verify step that could be reordered.)
//
// Aborts BEFORE opkg-remove on download or verification failure so a
// flaky network does not strand the user without a working sing-box.
// The opkg-remove error is logged but is NOT fatal — the managed
// binary is activated regardless, so the user ends up with our binary
// even if opkg lock contention prevented removing the legacy package
// (a follow-up boot can clean it up).
//
// Idempotent: when the managed binary is already in place Migrate returns
// nil without side effects.
func (i *Installer) Migrate(ctx context.Context, lc Lifecycle) error {
	if i.HasManagedBinary() {
		i.appLog.Info("migrate", "", "managed binary already present, skipping")
		return nil
	}

	i.appLog.Info("migrate", "", "starting auto-migration from sing-box-naive opkg package")

	tmp, err := i.Download(ctx, nil)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	if err := lc.Stop(ctx); err != nil {
		i.appLog.Warn("migrate", "", fmt.Sprintf("Lifecycle.Stop error (continuing): %v", err))
		// Best-effort — even if we can't stop the legacy proc, continue.
	}

	remove := i.opkgRemove
	if remove == nil {
		remove = defaultOpkgRemove
	}
	if err := remove(ctx); err != nil {
		// opkg may be locked; we still want to land the managed binary.
		i.appLog.Warn("migrate", "opkg", err.Error())
	}

	if err := i.Activate(tmp); err != nil {
		// Critical state: opkg-remove ran (or was attempted) and the
		// managed binary failed to land. The user has no working
		// sing-box until they retry install. Loud-log so the failure
		// is unmistakable in /logs.
		i.appLog.Error("migrate", "", fmt.Sprintf("CRITICAL: legacy package removed but managed binary activation failed — no sing-box available until reinstall: %v", err))
		return fmt.Errorf("activate: %w", err)
	}

	if err := lc.Start(ctx); err != nil {
		i.appLog.Warn("migrate", "", fmt.Sprintf("Lifecycle.Start failed after activation: %v", err))
		return fmt.Errorf("start: %w", err)
	}

	i.appLog.Info("migrate", "", "migration complete")
	return nil
}
