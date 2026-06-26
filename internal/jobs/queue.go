package jobs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Job struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Step       string `json:"step"`
	Pct        int    `json:"pct"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	HFRepo     string `json:"hf_repo"`
	OutputName string `json:"output_name"`
	CreatedAt  int64  `json:"created_at"`
}

// Config controls how conversion jobs are executed.
type Config struct {
	Python    string // interpreter, e.g. "python3"
	Script    string // path to gguf_to_mlx.py
	OutputDir string // base dir under which <output_name> MLX weights are written
	HFToken   string // optional HuggingFace token for gated/private models
}

func ConfigFromEnv() Config {
	return Config{
		Python:    envOr("COFISWARM_CONVERT_PYTHON", "python3"),
		Script:    resolveScript(),
		OutputDir: resolveOutputDir(),
		HFToken:   os.Getenv("HF_TOKEN"),
	}
}

type Queue struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	cfg  Config
	// run executes a job; defaults to runPython and is overridable in tests.
	run func(j *Job, qBits int)
}

func New() *Queue { return NewWithConfig(ConfigFromEnv()) }

func NewWithConfig(cfg Config) *Queue {
	q := &Queue{jobs: map[string]*Job{}, cfg: cfg}
	q.run = q.runPython
	return q
}

// Create registers a job and dispatches the conversion to a background worker.
func (q *Queue) Create(hfRepo, outputName string, qBits int) *Job {
	now := time.Now().UnixMilli()
	j := &Job{
		ID: fmt.Sprintf("cvt_%d", now), Status: "running", Step: "starting", Pct: 0,
		HFRepo: hfRepo, OutputName: outputName, CreatedAt: now,
	}
	q.mu.Lock()
	q.jobs[j.ID] = j
	q.mu.Unlock()
	go q.run(j, qBits)
	return j
}

func (q *Queue) Get(id string) (*Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	j, ok := q.jobs[id]
	return j, ok
}

func (q *Queue) List() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()
	out := make([]*Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		out = append(out, j)
	}
	return out
}

// progress mirrors a single NDJSON line emitted by gguf_to_mlx.py. Pct is a
// pointer so an omitted value does not reset the job to 0%.
type progress struct {
	Status string `json:"status"`
	Step   string `json:"step"`
	Pct    *int   `json:"pct"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

// runPython spawns the converter and streams its NDJSON progress into the job.
func (q *Queue) runPython(j *Job, qBits int) {
	outDir := filepath.Join(q.cfg.OutputDir, j.OutputName)
	args := []string{
		q.cfg.Script,
		"--hf-repo", j.HFRepo,
		"--output", outDir,
		"--q-bits", fmt.Sprintf("%d", qBits),
	}
	if q.cfg.HFToken != "" {
		args = append(args, "--hf-token", q.cfg.HFToken)
	}
	cmd := exec.Command(q.cfg.Python, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		q.fail(j, fmt.Sprintf("stdout pipe: %v", err))
		return
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		q.fail(j, fmt.Sprintf("start %q: %v", q.cfg.Python, err))
		return
	}

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] != '{' {
			continue // non-JSON noise on stdout (e.g. download bars)
		}
		var p progress
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			log.Printf("[convert] job %s: unparseable progress %q: %v", j.ID, line, err)
			continue
		}
		q.applyProgress(j, p)
	}

	werr := cmd.Wait()
	q.mu.RLock()
	status := j.Status
	q.mu.RUnlock()
	if werr != nil && status != "error" {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = werr.Error()
		}
		q.fail(j, msg)
		return
	}
	if status != "done" && status != "error" {
		// Process exited cleanly without a terminal line — treat as success.
		q.update(j, func(jj *Job) {
			jj.Status, jj.Step, jj.Pct = "done", "done", 100
			if jj.Output == "" {
				jj.Output = outDir
			}
		})
	}
}

func (q *Queue) applyProgress(j *Job, p progress) {
	q.update(j, func(jj *Job) {
		if p.Status != "" {
			jj.Status = p.Status
		}
		if p.Step != "" {
			jj.Step = p.Step
		}
		if p.Pct != nil {
			jj.Pct = *p.Pct
		}
		if p.Output != "" {
			jj.Output = p.Output
		}
		if p.Error != "" {
			jj.Error = p.Error
		}
	})
	if p.Status == "error" {
		log.Printf("[convert] job %s failed: %s", j.ID, p.Error)
	}
}

func (q *Queue) fail(j *Job, msg string) {
	log.Printf("[convert] job %s failed: %s", j.ID, msg)
	q.update(j, func(jj *Job) {
		jj.Status, jj.Error = "error", msg
	})
}

func (q *Queue) update(j *Job, fn func(*Job)) {
	q.mu.Lock()
	defer q.mu.Unlock()
	fn(j)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// resolveScript locates gguf_to_mlx.py: explicit env override, then next to the
// binary, then the repo-relative src/ path.
func resolveScript() string {
	if v := os.Getenv("COFISWARM_CONVERT_SCRIPT"); v != "" {
		return v
	}
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "src", "gguf_to_mlx.py"),
			filepath.Join(dir, "..", "src", "gguf_to_mlx.py"),
		)
	}
	candidates = append(candidates, filepath.Join("src", "gguf_to_mlx.py"))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return filepath.Join("src", "gguf_to_mlx.py")
}

// resolveOutputDir picks the MLX weight output base from env, defaulting to the
// FHS var/lib models tree.
func resolveOutputDir() string {
	if v := os.Getenv("COFISWARM_CONVERT_OUTPUT_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("COFISWARM_VAR_LIB"); v != "" {
		return filepath.Join(v, "models", "MLX")
	}
	return filepath.Join("models", "MLX")
}
