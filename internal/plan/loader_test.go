package plan

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	hasTerraform bool
	planOutput   []byte
	planStderr   []byte
	showOutput   []byte
	planErr      error
	showErr      error
	calls        [][]string
}

func (f *fakeRunner) lookPath(bin string) (string, error) {
	if f.hasTerraform {
		return "/usr/bin/" + bin, nil
	}
	return "", os.ErrNotExist
}

func (f *fakeRunner) run(ctx context.Context, dir string, args ...string) ([]byte, []byte, error) {
	f.calls = append(f.calls, args)
	switch args[0] {
	case "plan":
		return f.planOutput, f.planStderr, f.planErr
	case "show":
		return f.showOutput, []byte("show stderr"), f.showErr
	}
	return nil, nil, nil
}

func TestLoaderStdin(t *testing.T) {
	l := &fileLoader{run: &fakeRunner{}}
	b, info, err := l.Load(context.Background(), "-", strings.NewReader(`{"format_version":"1.2"}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode != "stdin" || string(b) == "" {
		t.Fatalf("bad result: %q %+v", b, info)
	}
}

func TestLoaderStdinEmpty(t *testing.T) {
	l := &fileLoader{run: &fakeRunner{}}
	if _, _, err := l.Load(context.Background(), "", strings.NewReader("   \n")); err == nil {
		t.Fatal("expected error for empty stdin")
	}
}

func TestLoaderFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(path, []byte(`{"format_version":"1.2"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	l := &fileLoader{run: &fakeRunner{}}
	b, info, err := l.Load(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode != "file" || len(b) == 0 {
		t.Fatalf("bad result: %+v", info)
	}
}

func TestLoaderMissingPath(t *testing.T) {
	l := &fileLoader{run: &fakeRunner{}}
	if _, _, err := l.Load(context.Background(), "/no/such/plan.json", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoaderDirNoTerraform(t *testing.T) {
	dir := t.TempDir()
	l := &fileLoader{run: &fakeRunner{hasTerraform: false}}
	_, _, err := l.Load(context.Background(), dir, nil)
	if err == nil || !strings.Contains(err.Error(), "terraform") {
		t.Fatalf("want terraform-not-found error, got %v", err)
	}
}

func TestLoaderDirPlanFailureShowsTerraformDiagnostic(t *testing.T) {
	dir := t.TempDir()
	planOutput := []byte(`{"type":"diagnostic","diagnostic":{"severity":"error","summary":"Backend initialization required","detail":"Please run terraform init with -reconfigure."}}`)
	fr := &fakeRunner{hasTerraform: true, planOutput: planOutput, planErr: errors.New("exit status 1")}
	l := &fileLoader{run: fr}
	_, _, err := l.Load(context.Background(), dir, nil)
	if err == nil {
		t.Fatal("expected plan error")
	}
	if !strings.Contains(err.Error(), "Backend initialization required") || !strings.Contains(err.Error(), "Please run terraform init with -reconfigure") {
		t.Fatalf("Terraform diagnostic was hidden: %v", err)
	}
	if strings.Contains(err.Error(), `"type":"diagnostic"`) {
		t.Fatalf("raw JSON diagnostic was not formatted: %v", err)
	}
}

func TestLoaderDirSuccess(t *testing.T) {
	dir := t.TempDir()
	planOutput := []byte("{\"type\":\"diagnostic\",\"diagnostic\":{\"severity\":\"warning\",\"summary\":\"Deprecated resource\",\"detail\":\"aws_old_resource is deprecated; use aws_new_resource\",\"range\":{\"filename\":\"main.tf\",\"start\":{\"line\":7}}}}\n")
	fr := &fakeRunner{hasTerraform: true, planOutput: planOutput, showOutput: []byte(`{"format_version":"1.2"}`)}
	l := &fileLoader{run: fr}
	b, info, err := l.Load(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode != "dir" || string(b) != `{"format_version":"1.2"}` {
		t.Fatalf("bad result: %q %+v", b, info)
	}
	if len(info.Deprecations) != 1 || info.Deprecations[0].Filename != "main.tf" || info.Deprecations[0].Line != 7 {
		t.Fatalf("deprecation diagnostic not captured: %+v", info.Deprecations)
	}
	if len(fr.calls) != 2 || fr.calls[0][0] != "plan" || fr.calls[1][0] != "show" {
		t.Fatalf("unexpected terraform calls: %v", fr.calls)
	}
	if len(fr.calls[0]) < 3 || fr.calls[0][1] != "-json" {
		t.Fatalf("plan did not enable machine-readable output: %v", fr.calls[0])
	}
}
