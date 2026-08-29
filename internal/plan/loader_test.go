package plan

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	hasTerraform bool
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
		return nil, []byte("plan stderr"), f.planErr
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

func TestLoaderDirSuccess(t *testing.T) {
	dir := t.TempDir()
	fr := &fakeRunner{hasTerraform: true, showOutput: []byte(`{"format_version":"1.2"}`)}
	l := &fileLoader{run: fr}
	b, info, err := l.Load(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode != "dir" || string(b) != `{"format_version":"1.2"}` {
		t.Fatalf("bad result: %q %+v", b, info)
	}
	if len(fr.calls) != 2 || fr.calls[0][0] != "plan" || fr.calls[1][0] != "show" {
		t.Fatalf("unexpected terraform calls: %v", fr.calls)
	}
}
