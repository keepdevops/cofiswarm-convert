package jobs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeQueue builds a queue whose "interpreter" is /bin/sh running the given
// shell body, so the spawn + NDJSON-parse path is exercised without Python.
func fakeQueue(t *testing.T, body string) *Queue {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}
	return NewWithConfig(Config{Python: "/bin/sh", Script: script, OutputDir: dir})
}

func waitFor(t *testing.T, q *Queue, id, want string, d time.Duration) *Job {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if j, ok := q.Get(id); ok && j.Status == want {
			return j
		}
		time.Sleep(5 * time.Millisecond)
	}
	j, _ := q.Get(id)
	t.Fatalf("job %s: status %q never reached (last=%+v)", id, want, j)
	return nil
}

func TestRunParsesProgressAndCompletes(t *testing.T) {
	q := fakeQueue(t, `
echo '{"status":"running","step":"downloading","pct":42}'
echo '{"status":"done","step":"done","pct":100,"output":"/tmp/demo"}'
`)
	j := q.Create("org/model", "demo", 4)
	got := waitFor(t, q, j.ID, "done", 5*time.Second)
	if got.Pct != 100 || got.Step != "done" {
		t.Errorf("final job = %+v, want pct=100 step=done", got)
	}
	if got.Output != "/tmp/demo" {
		t.Errorf("output = %q, want /tmp/demo", got.Output)
	}
}

func TestRunMarksErrorOnNonZeroExit(t *testing.T) {
	q := fakeQueue(t, "echo 'boom failure' 1>&2\nexit 3\n")
	j := q.Create("org/model", "demo", 4)
	got := waitFor(t, q, j.ID, "error", 5*time.Second)
	if !strings.Contains(got.Error, "boom failure") {
		t.Errorf("error = %q, want it to contain stderr 'boom failure'", got.Error)
	}
}

func TestRunHonorsScriptEmittedError(t *testing.T) {
	q := fakeQueue(t, `
echo '{"status":"error","error":"mlx_lm not available"}'
exit 1
`)
	j := q.Create("org/model", "demo", 4)
	got := waitFor(t, q, j.ID, "error", 5*time.Second)
	if got.Error != "mlx_lm not available" {
		t.Errorf("error = %q, want the script-emitted message", got.Error)
	}
}

func TestRunMarksErrorWhenInterpreterMissing(t *testing.T) {
	q := NewWithConfig(Config{Python: "/no/such/interpreter", Script: "x.py", OutputDir: t.TempDir()})
	j := q.Create("org/model", "demo", 4)
	got := waitFor(t, q, j.ID, "error", 5*time.Second)
	if got.Error == "" {
		t.Errorf("missing interpreter should set a loud error, got %+v", got)
	}
}
