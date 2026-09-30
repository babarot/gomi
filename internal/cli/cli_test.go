package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/xdg"
	"github.com/babarot/gomi/internal/utils/log"
)

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo.txt", "foo.txt"},
		{"./foo.txt", "foo.txt"},
		{"foo/../bar.txt", "bar.txt"},
		{"foo/./bar.txt", filepath.Join("foo", "bar.txt")},
		{"/absolute/path", filepath.Clean("/absolute/path")},
		{"", "."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := expandPath(tt.input)
			if err != nil {
				t.Fatalf("expandPath(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandWindowsPaths(t *testing.T) {
	// Without actual glob matches, args should be returned as-is
	args := []string{"noexist*.xyz", "plain.txt"}
	got := expandWindowsPaths(args)
	if len(got) != 2 {
		t.Fatalf("expandWindowsPaths() returned %d items, want 2", len(got))
	}
	if got[1] != "plain.txt" {
		t.Errorf("expandWindowsPaths()[1] = %q, want %q", got[1], "plain.txt")
	}
}

func TestExpandWindowsPaths_WithMatches(t *testing.T) {
	dir := t.TempDir()

	// Create temp files to match glob
	for _, name := range []string{"a.tmp", "b.tmp"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	args := []string{dir + "/*.tmp"}
	got := expandWindowsPaths(args)
	if len(got) != 2 {
		t.Fatalf("expandWindowsPaths() returned %d items, want 2", len(got))
	}
}

func TestDetermineLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  log.Level
	}{
		{"debug", log.DebugLevel},
		{"info", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"unknown", log.DebugLevel},
		{"", log.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := determineLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("determineLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsForbiddenPath(t *testing.T) {
	cli := &CLI{
		config: &config.Config{
			Core: config.Core{
				Trash: config.TrashConfig{
					ForbiddenPaths: []string{
						"/",
						"/etc",
						"/usr",
						"$HOME/.gomi",
					},
				},
			},
		},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/etc", true},
		{"/etc/hosts", true},     // sub-path of /etc
		{"/usr", true},           // exact match
		{"/usr/local/bin", true}, // sub-path of /usr
		{"/tmp/foo", false},
		{"/home/user/file.txt", false},
		{"relative/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := cli.isForbiddenPath(tt.path)
			if got != tt.want {
				t.Errorf("isForbiddenPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsForbiddenPathRelative(t *testing.T) {
	root := t.TempDir()
	sys := filepath.Join(root, "sys")
	if err := os.Mkdir(sys, 0o755); err != nil {
		t.Fatal(err)
	}
	cli := &CLI{
		config: &config.Config{
			Core: config.Core{
				Trash: config.TrashConfig{ForbiddenPaths: []string{sys}},
			},
		},
	}

	// A relative argument given inside a forbidden directory is caught.
	// The check comes before the trash is touched, so none is set up.
	t.Chdir(sys)
	err := cli.processFile("file", &syncStringSlice{})
	if err == nil || !strings.Contains(err.Error(), "forbidden path") {
		t.Errorf("processFile(%q) = %v, want a forbidden path error", "file", err)
	}
}

func TestIsForbidden(t *testing.T) {
	// root/
	//   sys/           forbidden, like /var
	//     tmp/         tempDir, like /var/folders/.../T
	//       keep/      forbidden explicitly
	//     other/
	//   link -> sys    like /var -> /private/var
	//   home/
	//     result -> sys/other
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sys := filepath.Join(root, "sys")
	tmp := filepath.Join(sys, "tmp")
	keep := filepath.Join(tmp, "keep")
	other := filepath.Join(sys, "other")
	home := filepath.Join(root, "home")
	for _, d := range []string{keep, other, home} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(sys, link); err != nil {
		t.Fatal(err)
	}
	result := filepath.Join(home, "result")
	if err := os.Symlink(other, result); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		path      string
		forbidden []string
		tempDir   string
		want      bool
	}{
		{"exact", sys, []string{sys}, "", true},
		{"inside", filepath.Join(sys, "x"), []string{sys}, "", true},
		{"outside", filepath.Join(home, "x"), []string{sys}, "", false},
		{"sibling with same prefix", sys + "2", []string{sys}, "", false},
		{"root matches only itself", filepath.Join(home, "x"), []string{"/"}, "", false},
		{"through a symlinked parent", filepath.Join(link, "x"), []string{sys}, "", true},
		{"forbidden given as the symlink", filepath.Join(sys, "x"), []string{link}, "", true},
		{"symlink itself pointing into forbidden", result, []string{sys}, "", false},
		{"inside tempDir", filepath.Join(tmp, "x"), []string{sys}, tmp, false},
		{"inside tempDir through symlink", filepath.Join(link, "tmp", "x"), []string{sys}, tmp, false},
		{"inside tempDir given as symlink", filepath.Join(tmp, "x"), []string{sys}, filepath.Join(link, "tmp"), false},
		{"tempDir itself", tmp, []string{sys}, tmp, true},
		{"elsewhere in forbidden with tempDir", filepath.Join(other, "x"), []string{sys}, tmp, true},
		{"forbidden inside tempDir", filepath.Join(keep, "x"), []string{sys, keep}, tmp, true},
		{"tempDir forbidden itself", filepath.Join(tmp, "x"), []string{tmp}, tmp, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isForbidden(tt.path, tt.forbidden, tt.tempDir)
			if got != tt.want {
				t.Errorf("isForbidden(%q, %q, %q) = %v, want %v", tt.path, tt.forbidden, tt.tempDir, got, tt.want)
			}
		})
	}
}

func TestSyncStringSlice(t *testing.T) {
	s := &syncStringSlice{}

	// Empty initially
	if got := s.Get(); len(got) != 0 {
		t.Errorf("empty slice: Get() returned %d items", len(got))
	}

	// Append and verify
	s.Append("a")
	s.Append("b")
	s.Append("c")

	got := s.Get()
	if len(got) != 3 {
		t.Fatalf("Get() returned %d items, want 3", len(got))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("Get() = %v, want [a b c]", got)
	}

	// Verify Get() returns a copy (modifying returned slice doesn't affect original)
	got[0] = "modified"
	original := s.Get()
	if original[0] != "a" {
		t.Error("Get() should return a copy, not a reference")
	}
}

func TestSyncStringSlice_Concurrent(t *testing.T) {
	s := &syncStringSlice{}
	done := make(chan struct{})

	// Concurrent writes
	for range 100 {
		go func() {
			s.Append("item")
			done <- struct{}{}
		}()
	}
	for range 100 {
		<-done
	}

	if got := s.Get(); len(got) != 100 {
		t.Errorf("concurrent Append: got %d items, want 100", len(got))
	}
}

func TestVersion_String(t *testing.T) {
	v := Version{
		AppName:   "gomi",
		Version:   "1.0.0",
		Revision:  "abc123",
		BuildDate: "2024-01-01",
	}

	s := v.String()
	if !strings.Contains(s, "gomi") {
		t.Error("should contain app name")
	}
	if !strings.Contains(s, "1.0.0") {
		t.Error("should contain version")
	}
	if !strings.Contains(s, "abc123") {
		t.Error("should contain revision")
	}
	if !strings.Contains(s, "2024-01-01") {
		t.Error("should contain build date")
	}
	if !strings.Contains(s, appURL) {
		t.Error("should contain app URL")
	}
}

func TestCLI_Run_Version(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cli := CLI{
		version: Version{
			AppName:   "gomi",
			Version:   "1.0.0",
			Revision:  "abc",
			BuildDate: "2024-01-01",
		},
		option: Option{
			Meta: MetaOption{Version: true},
		},
		config: config.NewDefaultConfig(),
	}

	err := cli.Run(nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("version output should contain version, got %q", output)
	}
}

func TestCLI_Run_PutNoArgs(t *testing.T) {
	cli := CLI{
		option: Option{},
		config: config.NewDefaultConfig(),
	}

	err := cli.Run(nil)
	if err == nil {
		t.Error("Run() with no args should return error")
	}
	if !strings.Contains(err.Error(), "too few arguments") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrune_NoArgs(t *testing.T) {
	cli := CLI{config: config.NewDefaultConfig()}
	err := cli.Prune(nil)
	if err == nil {
		t.Fatal("Prune(nil) should return error")
	}
	if !strings.Contains(err.Error(), "requires an argument") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrune_OrphansWithOtherArgs(t *testing.T) {
	cli := CLI{config: config.NewDefaultConfig()}
	err := cli.Prune([]string{"orphans", "30d"})
	if err == nil {
		t.Fatal("Prune with orphans + duration should return error")
	}
	if !strings.Contains(err.Error(), "cannot be combined") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrune_InvalidDuration(t *testing.T) {
	cli := CLI{config: config.NewDefaultConfig()}
	err := cli.Prune([]string{"xyz"})
	if err == nil {
		t.Fatal("Prune with invalid duration should return error")
	}
}

func TestPrune_EmptyArg(t *testing.T) {
	cli := CLI{config: config.NewDefaultConfig()}
	err := cli.Prune([]string{""})
	if err == nil {
		t.Fatal("Prune with empty arg should return error")
	}
}

func TestPruneArgs_UnmarshalFlag(t *testing.T) {
	var p PruneArgs
	if err := p.UnmarshalFlag("30d,orphans"); err != nil {
		t.Fatalf("UnmarshalFlag() error = %v", err)
	}
	if len(p) != 2 {
		t.Fatalf("expected 2 args, got %d", len(p))
	}
	if p[0] != "30d" || p[1] != "orphans" {
		t.Errorf("args = %v, want [30d orphans]", p)
	}
}

func TestOrphanedFile_Getters(t *testing.T) {
	o := xdg.OrphanedFile{
		TrashInfoName: "test.trashinfo",
	}
	if o.GetName() != "test.trashinfo" {
		t.Errorf("GetName() = %q, want %q", o.GetName(), "test.trashinfo")
	}
	// GetDeletedAt() should not panic on zero value
	_ = o.GetDeletedAt()
}

func TestFilterFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a real file to represent an existing trash entry
	existingFile := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	cli := &CLI{config: config.NewDefaultConfig()}

	files := []*trash.File{
		{Name: "exists.txt", TrashPath: existingFile},
		{Name: "gone.txt", TrashPath: "/nonexistent/gone.txt"},
	}

	filtered := cli.filterFiles(files)
	if len(filtered) != 1 {
		t.Fatalf("filterFiles() returned %d files, want 1", len(filtered))
	}
	if filtered[0].Name != "exists.txt" {
		t.Errorf("Name = %q, want %q", filtered[0].Name, "exists.txt")
	}
}
