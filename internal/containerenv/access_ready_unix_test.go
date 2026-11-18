//go:build !windows

package containerenv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContainerSSHReadinessWaitsForService(t *testing.T) {
	dir := t.TempDir()
	engine := filepath.Join(dir, "engine")
	count := filepath.Join(dir, "count")
	script := "#!/bin/sh\nset -eu\n[ \"$1 $2 $3 $4 $5 $6 $7\" = 'exec --user 0 c1 /bin/sh -c systemctl is-active --quiet ssh.service' ]\nn=0\n[ ! -f '" + count + "' ] || n=$(cat '" + count + "')\nn=$((n+1))\nprintf '%s' \"$n\" > '" + count + "'\n[ \"$n\" -ge 3 ]\n"
	if err := os.WriteFile(engine, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	r := &Runtime{config: Config{Engine: "podman", Executable: engine}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.waitForSSHService(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	value, err := os.ReadFile(count)
	if err != nil || string(value) != "3" {
		t.Fatalf("readiness did not wait for the actual service transition: %q, %v", value, err)
	}
}

func TestContainerSSHReadinessHonorsCancellation(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	if err := os.WriteFile(engine, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	r := &Runtime{config: Config{Engine: "podman", Executable: engine}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.waitForSSHService(ctx, "c1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled readiness wait returned %v", err)
	}
}
