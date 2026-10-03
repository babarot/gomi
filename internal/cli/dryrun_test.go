package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/xdg"
)

func TestDryRunFormat(t *testing.T) {
	tests := []struct {
		format     string
		isTerminal bool
		want       string
	}{
		{"auto", true, "table"},
		{"auto", false, "text"},
		{"table", false, "table"},
		{"text", true, "text"},
		{"json", true, "json"},
	}
	for _, tt := range tests {
		if got := dryRunFormat(tt.format, tt.isTerminal); got != tt.want {
			t.Errorf("dryRunFormat(%q, %v) = %q, want %q", tt.format, tt.isTerminal, got, tt.want)
		}
	}
}

func dryRunTestFiles() []*trash.File {
	now := time.Now()
	return []*trash.File{
		{
			Name:         "old.txt",
			OriginalPath: "/home/user/old.txt",
			TrashPath:    "/home/user/.local/share/Trash/files/old.txt",
			DeletedAt:    now.Add(-90 * 24 * time.Hour),
		},
		{
			Name:         "dir",
			OriginalPath: "/home/user/src/dir",
			TrashPath:    "/home/user/.local/share/Trash/files/dir",
			DeletedAt:    now.Add(-40 * 24 * time.Hour),
			IsDir:        true,
		},
	}
}

func TestPrintDryRunFiles_Text(t *testing.T) {
	var buf bytes.Buffer
	if err := printDryRunFiles(&buf, dryRunTestFiles(), "text"); err != nil {
		t.Fatalf("printDryRunFiles() error = %v", err)
	}
	// newest first, as the table shows them
	want := "/home/user/.local/share/Trash/files/dir\n" +
		"/home/user/.local/share/Trash/files/old.txt\n"
	if got := buf.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestPrintDryRunFiles_JSON(t *testing.T) {
	var buf bytes.Buffer
	if err := printDryRunFiles(&buf, dryRunTestFiles(), "json"); err != nil {
		t.Fatalf("printDryRunFiles() error = %v", err)
	}
	var got []dryRunFile
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != 2 {
		t.Fatalf("got %d files, want 2", len(got))
	}
	if got[0].Name != "dir" || got[0].OriginalPath != "/home/user/src/dir" || !got[0].IsDir {
		t.Errorf("got[0] = %+v, want the dir entry first", got[0])
	}
	if got[1].TrashPath != "/home/user/.local/share/Trash/files/old.txt" {
		t.Errorf("got[1].TrashPath = %q", got[1].TrashPath)
	}
}

func TestPrintDryRunFiles_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := printDryRunFiles(&buf, nil, "text"); err != nil {
		t.Fatalf("printDryRunFiles() error = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("text output = %q, want nothing", buf.String())
	}

	buf.Reset()
	if err := printDryRunFiles(&buf, nil, "json"); err != nil {
		t.Fatalf("printDryRunFiles() error = %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "[]" {
		t.Errorf("json output = %q, want []", got)
	}
}

func TestPrintDryRunOrphans(t *testing.T) {
	files := []xdg.OrphanedFile{
		{TrashInfoPath: "/trash/info/a.trashinfo", OriginalPath: "/home/user/a", DeletedAt: time.Now().Add(-2 * time.Hour)},
		{TrashInfoPath: "/trash/info/b.trashinfo", OriginalPath: "/home/user/b", DeletedAt: time.Now().Add(-1 * time.Hour)},
	}

	var buf bytes.Buffer
	if err := printDryRunOrphans(&buf, files, "text"); err != nil {
		t.Fatalf("printDryRunOrphans() error = %v", err)
	}
	want := "/trash/info/b.trashinfo\n/trash/info/a.trashinfo\n"
	if got := buf.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}

	buf.Reset()
	if err := printDryRunOrphans(&buf, files, "json"); err != nil {
		t.Fatalf("printDryRunOrphans() error = %v", err)
	}
	var got []dryRunOrphan
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != 2 || got[0].TrashInfoPath != "/trash/info/b.trashinfo" || got[0].OriginalPath != "/home/user/b" {
		t.Errorf("got = %+v", got)
	}
}

// listOnlyTrash lists the given files and records any removal
type listOnlyTrash struct {
	trash.Trash
	files   []*trash.File
	removed []*trash.File
}

func (l *listOnlyTrash) List() ([]*trash.File, error) { return l.files, nil }

func (l *listOnlyTrash) Remove(file *trash.File) error {
	l.removed = append(l.removed, file)
	return nil
}

func TestPrune_DryRunRemovesNothing(t *testing.T) {
	tr := &listOnlyTrash{files: dryRunTestFiles()}
	cli := &CLI{
		config: config.NewDefaultConfig(),
		trash:  tr,
	}
	// "auto" prints paths since stdout is a pipe here, and
	// -f must not make the dry run remove anything
	cli.option.Meta.DryRun = "auto"
	cli.option.Rm.Force = true

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cli.Prune([]string{"60d"})
	w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}
	if len(tr.removed) != 0 {
		t.Errorf("dry run removed %d files, want 0", len(tr.removed))
	}
	// only the file older than 60 days
	want := "/home/user/.local/share/Trash/files/old.txt\n"
	if got := buf.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestCLI_Run_DryRunWithoutPrune(t *testing.T) {
	cli := CLI{config: config.NewDefaultConfig()}
	cli.option.Meta.DryRun = "auto"

	err := cli.Run([]string{"file"})
	if err == nil || !strings.Contains(err.Error(), "--dry-run requires --prune") {
		t.Errorf("Run() = %v, want an error that --dry-run requires --prune", err)
	}
}
