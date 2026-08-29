package plan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// LoadInfo describes how a plan was obtained.
type LoadInfo struct {
	Mode string // "file", "stdin", or "dir"
}

// PlanLoader resolves a --path value (or stdin) to raw plan JSON bytes.
type PlanLoader interface {
	Load(ctx context.Context, source string, stdin io.Reader) ([]byte, LoadInfo, error)
}

// NewLoader returns the default PlanLoader.
func NewLoader() PlanLoader { return &fileLoader{run: execRunner{}} }

// runner abstracts running the terraform binary so tests do not shell out.
type runner interface {
	run(ctx context.Context, dir string, args ...string) (stdout, stderr []byte, err error)
	lookPath(bin string) (string, error)
}

type fileLoader struct {
	run runner
}

func (l *fileLoader) Load(ctx context.Context, source string, stdin io.Reader) ([]byte, LoadInfo, error) {
	if source == "" || source == "-" {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return nil, LoadInfo{}, fmt.Errorf("read plan from stdin: %w", err)
		}
		if len(bytes.TrimSpace(b)) == 0 {
			return nil, LoadInfo{}, fmt.Errorf("empty plan input on stdin")
		}
		return b, LoadInfo{Mode: "stdin"}, nil
	}

	info, err := os.Stat(source)
	if err != nil {
		return nil, LoadInfo{}, fmt.Errorf("open plan path: %w", err)
	}
	if !info.IsDir() {
		b, err := os.ReadFile(source)
		if err != nil {
			return nil, LoadInfo{}, fmt.Errorf("read plan file: %w", err)
		}
		if len(bytes.TrimSpace(b)) == 0 {
			return nil, LoadInfo{}, fmt.Errorf("plan file %s is empty", source)
		}
		return b, LoadInfo{Mode: "file"}, nil
	}

	b, err := l.loadFromDir(ctx, source)
	if err != nil {
		return nil, LoadInfo{}, err
	}
	return b, LoadInfo{Mode: "dir"}, nil
}

func (l *fileLoader) loadFromDir(ctx context.Context, dir string) ([]byte, error) {
	if _, err := l.run.lookPath("terraform"); err != nil {
		return nil, fmt.Errorf(
			"-path %q is a directory but 'terraform' was not found on PATH; "+
				"pass a plan JSON file or pipe 'terraform show -json' output instead", dir)
	}

	tmp, err := os.CreateTemp("", "price-checker-plan-*.tfplan")
	if err != nil {
		return nil, fmt.Errorf("create temp plan file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if _, stderr, err := l.run.run(ctx, dir, "plan", "-input=false", "-out", tmpPath); err != nil {
		return nil, fmt.Errorf("terraform plan failed: %s: %w", trimStderr(stderr), err)
	}
	stdout, stderr, err := l.run.run(ctx, dir, "show", "-json", tmpPath)
	if err != nil {
		return nil, fmt.Errorf("terraform show -json failed: %s: %w", trimStderr(stderr), err)
	}
	if len(bytes.TrimSpace(stdout)) == 0 {
		return nil, fmt.Errorf("terraform show -json produced no output for %s", dir)
	}
	return stdout, nil
}

func trimStderr(b []byte) string {
	s := string(bytes.TrimSpace(b))
	const max = 2000
	if len(s) > max {
		s = s[:max] + "…"
	}
	if s == "" {
		return "(no stderr)"
	}
	return s
}

type execRunner struct{}

func (execRunner) lookPath(bin string) (string, error) { return exec.LookPath(bin) }

func (execRunner) run(ctx context.Context, dir string, args ...string) ([]byte, []byte, error) {
	full := append([]string{"-chdir=" + filepath.Clean(dir)}, args...)
	cmd := exec.CommandContext(ctx, "terraform", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
