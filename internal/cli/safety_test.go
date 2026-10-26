package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui"
)

// This file tests the paths of the CLI that decide whether user data is
// moved, restored or permanently deleted. The trash is a fake backed by a
// temporary directory, so every test can check what is left on disk.

// scriptedPrompter answers prompts from fixed answers and records them
type scriptedPrompter struct {
	confirm    []bool // answers to Confirm, in order; false once used up
	confirmYes bool
	newName    string
	inputErr   error

	confirmPrompts    []string
	confirmYesPrompts []string
	inputAsked        int
}

func (p *scriptedPrompter) Confirm(prompt string) bool {
	p.confirmPrompts = append(p.confirmPrompts, prompt)
	if len(p.confirm) == 0 {
		return false
	}
	answer := p.confirm[0]
	p.confirm = p.confirm[1:]
	return answer
}

func (p *scriptedPrompter) ConfirmYes(prompt string) bool {
	p.confirmYesPrompts = append(p.confirmYesPrompts, prompt)
	return p.confirmYes
}

func (p *scriptedPrompter) InputFilename(*trash.File) (string, error) {
	p.inputAsked++
	return p.newName, p.inputErr
}

func (p *scriptedPrompter) asked() int {
	return len(p.confirmPrompts) + len(p.confirmYesPrompts) + p.inputAsked
}

// diskTrash is a trash kept in a directory: Put moves a file into it,
// Restore moves it back out (overwriting, as os.Rename does, so a wrong
// destination shows up as lost data), and Remove deletes it for good
type diskTrash struct {
	dir   string
	files []*trash.File

	listErr    error
	removeErrs map[string]error // by file name
	restoreErr error

	put      []string
	removed  []string
	restored []string
}

func newDiskTrash(t *testing.T) *diskTrash {
	t.Helper()
	return &diskTrash{dir: t.TempDir()}
}

// add places a file with content in the trash, deleted age ago
func (d *diskTrash) add(t *testing.T, name, originalPath, content string, age time.Duration) *trash.File {
	t.Helper()
	path := filepath.Join(d.dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &trash.File{
		Name:         name,
		OriginalPath: originalPath,
		TrashPath:    path,
		DeletedAt:    time.Now().Add(-age),
	}
	d.files = append(d.files, f)
	return f
}

func (d *diskTrash) Put(src string) error {
	d.put = append(d.put, src)
	return os.Rename(src, filepath.Join(d.dir, filepath.Base(src)))
}

func (d *diskTrash) List() ([]*trash.File, error) {
	return d.files, d.listErr
}

func (d *diskTrash) Restore(file *trash.File, dst string) error {
	if d.restoreErr != nil {
		return d.restoreErr
	}
	d.restored = append(d.restored, dst)
	return os.Rename(file.TrashPath, dst)
}

func (d *diskTrash) Remove(file *trash.File) error {
	if err := d.removeErrs[file.Name]; err != nil {
		return err
	}
	d.removed = append(d.removed, file.Name)
	return os.RemoveAll(file.TrashPath)
}

// captureStdout runs fn with os.Stdout sent to a pipe and returns the output
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		done <- buf.String()
	}()
	defer func() {
		os.Stdout = old
	}()
	fn()
	w.Close()
	return <-done
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err != nil {
		t.Errorf("%s is gone: %v", path, err)
	}
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("%s still exists (err = %v)", path, err)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("read %s: %v", path, err)
		return
	}
	if string(got) != want {
		t.Errorf("%s contains %q, want %q", path, got, want)
	}
}

// pruneFixture is a trash with one file deleted 90 days ago and one deleted
// 10 days ago
func pruneFixture(t *testing.T) (tr *diskTrash, old, recent *trash.File) {
	t.Helper()
	tr = newDiskTrash(t)
	old = tr.add(t, "old.txt", "/home/user/old.txt", "old", 90*24*time.Hour)
	recent = tr.add(t, "recent.txt", "/home/user/recent.txt", "recent", 10*24*time.Hour)
	return tr, old, recent
}

func TestPrune_DecliningFirstConfirmationDeletesNothing(t *testing.T) {
	tr, old, recent := pruneFixture(t)
	p := &scriptedPrompter{confirm: []bool{false}, confirmYes: true}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}

	var err error
	captureStdout(t, func() { err = cli.Prune([]string{"30d"}) })
	if err != nil {
		t.Fatalf("Prune() = %v, want nil", err)
	}
	if len(tr.removed) != 0 {
		t.Errorf("removed %v after the first confirmation was declined", tr.removed)
	}
	if len(p.confirmYesPrompts) != 0 {
		t.Errorf("asked the second confirmation after the first was declined")
	}
	assertExists(t, old.TrashPath)
	assertExists(t, recent.TrashPath)
}

func TestPrune_SecondConfirmationIsRequired(t *testing.T) {
	tr, old, _ := pruneFixture(t)
	p := &scriptedPrompter{confirm: []bool{true}, confirmYes: false}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}

	var err error
	out := captureStdout(t, func() { err = cli.Prune([]string{"30d"}) })
	if err != nil {
		t.Fatalf("Prune() = %v, want nil", err)
	}
	if len(p.confirmPrompts) != 1 || len(p.confirmYesPrompts) != 1 {
		t.Errorf("asked Confirm %d and ConfirmYes %d times, want 1 and 1",
			len(p.confirmPrompts), len(p.confirmYesPrompts))
	}
	if len(tr.removed) != 0 {
		t.Errorf("removed %v with only the first confirmation", tr.removed)
	}
	assertExists(t, old.TrashPath)
	if !strings.Contains(out, "Operation canceled.") {
		t.Errorf("output %q does not say the operation was canceled", out)
	}
}

func TestPrune_BothConfirmationsRemoveOnlyOlderFiles(t *testing.T) {
	tr, old, recent := pruneFixture(t)
	p := &scriptedPrompter{confirm: []bool{true}, confirmYes: true}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}

	var err error
	captureStdout(t, func() { err = cli.Prune([]string{"30d"}) })
	if err != nil {
		t.Fatalf("Prune() = %v, want nil", err)
	}
	if !strings.Contains(p.confirmPrompts[0], "1 files") {
		t.Errorf("confirmation %q does not name the 1 file to remove", p.confirmPrompts[0])
	}
	assertGone(t, old.TrashPath)
	assertExists(t, recent.TrashPath)
}

func TestPrune_RangeRemovesOnlyFilesBetween(t *testing.T) {
	tr := newDiskTrash(t)
	oldest := tr.add(t, "oldest", "/x/oldest", "", 100*24*time.Hour)
	middle := tr.add(t, "middle", "/x/middle", "", 20*24*time.Hour)
	newest := tr.add(t, "newest", "/x/newest", "", 1*24*time.Hour)
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: &scriptedPrompter{}}
	cli.option.Rm.Force = true

	var err error
	// order of the arguments does not matter
	captureStdout(t, func() { err = cli.Prune([]string{"30d", "7d"}) })
	if err != nil {
		t.Fatalf("Prune() = %v, want nil", err)
	}
	assertExists(t, oldest.TrashPath)
	assertGone(t, middle.TrashPath)
	assertExists(t, newest.TrashPath)
}

func TestPrune_ForceSkipsPrompts(t *testing.T) {
	tr, old, recent := pruneFixture(t)
	p := &scriptedPrompter{}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}
	cli.option.Rm.Force = true

	var err error
	captureStdout(t, func() { err = cli.Prune([]string{"30d"}) })
	if err != nil {
		t.Fatalf("Prune() = %v, want nil", err)
	}
	if p.asked() != 0 {
		t.Errorf("prompted %d times with -f", p.asked())
	}
	assertGone(t, old.TrashPath)
	assertExists(t, recent.TrashPath)
}

func TestPrune_DryRunNeverPromptsOrRemoves(t *testing.T) {
	for _, force := range []bool{false, true} {
		for _, args := range [][]string{{"30d"}, {"30d", "100d"}} {
			t.Run(fmt.Sprintf("force=%v/%s", force, strings.Join(args, ",")), func(t *testing.T) {
				tr, old, recent := pruneFixture(t)
				// a prompter that would say yes to everything
				p := &scriptedPrompter{confirm: []bool{true, true, true}, confirmYes: true}
				cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}
				cli.option.Meta.DryRun = "json"
				cli.option.Rm.Force = force

				var err error
				out := captureStdout(t, func() { err = cli.Prune(args) })
				if err != nil {
					t.Fatalf("Prune() = %v, want nil", err)
				}
				if p.asked() != 0 {
					t.Errorf("dry run prompted %d times", p.asked())
				}
				if len(tr.removed) != 0 {
					t.Errorf("dry run removed %v", tr.removed)
				}
				assertExists(t, old.TrashPath)
				assertExists(t, recent.TrashPath)
				if !strings.Contains(out, "old.txt") {
					t.Errorf("dry run output %q does not list old.txt", out)
				}
			})
		}
	}
}

func TestPrune_NoMatchingFilesDoesNotPrompt(t *testing.T) {
	tr, old, recent := pruneFixture(t)
	p := &scriptedPrompter{confirm: []bool{true}, confirmYes: true}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}

	var err error
	out := captureStdout(t, func() { err = cli.Prune([]string{"365d"}) })
	if err != nil {
		t.Fatalf("Prune() = %v, want nil", err)
	}
	if p.asked() != 0 {
		t.Errorf("prompted %d times with nothing to remove", p.asked())
	}
	assertExists(t, old.TrashPath)
	assertExists(t, recent.TrashPath)
	if !strings.Contains(out, "No matching files found.") {
		t.Errorf("output = %q", out)
	}
}

func TestPrune_ListErrorPropagates(t *testing.T) {
	tr := newDiskTrash(t)
	tr.listErr = errors.New("cannot read trash")
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: &scriptedPrompter{}}
	cli.option.Rm.Force = true

	err := cli.Prune([]string{"30d"})
	if err == nil || !strings.Contains(err.Error(), "cannot read trash") {
		t.Errorf("Prune() = %v, want the List error", err)
	}
}

func TestPrune_RemoveErrorPropagatesAndOthersStillRemoved(t *testing.T) {
	tr := newDiskTrash(t)
	stuck := tr.add(t, "stuck", "/x/stuck", "", 90*24*time.Hour)
	other := tr.add(t, "other", "/x/other", "", 90*24*time.Hour)
	tr.removeErrs = map[string]error{
		"stuck": &fs.PathError{Op: "unlinkat", Path: stuck.TrashPath + "/inner", Err: fs.ErrPermission},
	}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: &scriptedPrompter{}}
	cli.option.Rm.Force = true

	var err error
	out := captureStdout(t, func() { err = cli.Prune([]string{"30d"}) })
	if err == nil {
		t.Fatal("Prune() = nil, want an error when a file could not be removed")
	}
	assertExists(t, stuck.TrashPath)
	assertGone(t, other.TrashPath)
	if !strings.Contains(out, "Failed to remove 1 files") {
		t.Errorf("output does not report the failure:\n%s", out)
	}
	// the exact path that failed, and the hint for permission errors
	if !strings.Contains(out, stuck.TrashPath+"/inner") || !strings.Contains(out, "sudo rm -rf") {
		t.Errorf("output does not explain the permission failure:\n%s", out)
	}
	if strings.Contains(out, "Successfully removed") {
		t.Errorf("output claims success:\n%s", out)
	}
}

func TestShellQuote(t *testing.T) {
	tests := map[string]string{
		"/plain/path-1.txt": "/plain/path-1.txt",
		"":                  "''",
		"with space":        "'with space'",
		"it's":              `'it'\''s'`,
		"$(rm -rf ~)":       "'$(rm -rf ~)'",
	}
	for in, want := range tests {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

// writeOrphan creates trashDir/info/name.trashinfo with no data file
func writeOrphan(t *testing.T, trashDir, name string) string {
	t.Helper()
	info := filepath.Join(trashDir, "info")
	if err := os.MkdirAll(filepath.Join(trashDir, "files"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(info, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(info, name+".trashinfo")
	content := "[Trash Info]\nPath=/home/user/" + name + "\nDeletionDate=2024-01-02T03:04:05\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The orphan tests only take paths that remove nothing: the orphan search
// also scans the trash of every real mount point, which a test must not
// clean up.
func TestPruneOrphans_DeclineOrDryRunRemovesNothing(t *testing.T) {
	tests := []struct {
		name   string
		dryRun string
		force  bool
	}{
		{"declined", "", false},
		{"dry run", "text", false},
		{"dry run with force", "text", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataHome := t.TempDir()
			t.Setenv("XDG_DATA_HOME", dataHome)
			orphan := writeOrphan(t, filepath.Join(dataHome, "Trash"), "lost")

			p := &scriptedPrompter{confirm: []bool{false}}
			cli := &CLI{config: config.NewDefaultConfig(), prompter: p}
			cli.option.Meta.DryRun = tt.dryRun
			cli.option.Rm.Force = tt.force

			var err error
			out := captureStdout(t, func() { err = cli.Prune([]string{"orphans"}) })
			if err != nil {
				t.Fatalf("Prune(orphans) = %v, want nil", err)
			}
			assertExists(t, orphan)
			if tt.dryRun != "" {
				if p.asked() != 0 {
					t.Errorf("dry run prompted %d times", p.asked())
				}
				if !strings.Contains(out, orphan) {
					t.Errorf("dry run output does not list %s:\n%s", orphan, out)
				}
			} else if len(p.confirmPrompts) != 1 {
				t.Errorf("asked Confirm %d times, want 1", len(p.confirmPrompts))
			}
		})
	}
}

func TestRestoreFile_NoConflictRestoresToOriginalPath(t *testing.T) {
	tr := newDiskTrash(t)
	dst := filepath.Join(t.TempDir(), "doc.txt")
	f := tr.add(t, "doc.txt", dst, "content", time.Hour)
	p := &scriptedPrompter{}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}
	cli.config.Core.Restore.Confirm = false

	if err := cli.restoreFile(f); err != nil {
		t.Fatalf("restoreFile() = %v", err)
	}
	if p.asked() != 0 {
		t.Errorf("prompted %d times with nothing in the way", p.asked())
	}
	assertContent(t, dst, "content")
	assertGone(t, f.TrashPath)
}

func TestRestoreFile_DeclinedConfirmationRestoresNothing(t *testing.T) {
	tr := newDiskTrash(t)
	dst := filepath.Join(t.TempDir(), "doc.txt")
	f := tr.add(t, "doc.txt", dst, "content", time.Hour)
	p := &scriptedPrompter{confirm: []bool{false}}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}
	cli.config.Core.Restore.Confirm = true

	if err := cli.restoreFile(f); err != nil {
		t.Fatalf("restoreFile() = %v", err)
	}
	if len(p.confirmPrompts) != 1 {
		t.Errorf("asked Confirm %d times, want 1", len(p.confirmPrompts))
	}
	if len(tr.restored) != 0 {
		t.Errorf("restored to %v after the confirmation was declined", tr.restored)
	}
	assertGone(t, dst)
	assertExists(t, f.TrashPath)
}

// conflictFixture is a trashed doc.txt whose original path is taken by
// another file
func conflictFixture(t *testing.T) (tr *diskTrash, f *trash.File, dir, existing string) {
	t.Helper()
	tr = newDiskTrash(t)
	dir = t.TempDir()
	existing = filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(existing, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	f = tr.add(t, "doc.txt", existing, "from trash", time.Hour)
	return tr, f, dir, existing
}

func TestRestoreFile_ConflictRenames(t *testing.T) {
	tr, f, dir, existing := conflictFixture(t)
	p := &scriptedPrompter{newName: "doc-2.txt"}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}
	cli.config.Core.Restore.Confirm = false

	if err := cli.restoreFile(f); err != nil {
		t.Fatalf("restoreFile() = %v", err)
	}
	if p.inputAsked != 1 {
		t.Errorf("asked for a new name %d times, want 1", p.inputAsked)
	}
	assertContent(t, existing, "keep me")
	assertContent(t, filepath.Join(dir, "doc-2.txt"), "from trash")
}

func TestRestoreFile_ConflictCanceledRestoresNothing(t *testing.T) {
	tr, f, _, existing := conflictFixture(t)
	p := &scriptedPrompter{newName: "partial", inputErr: ui.ErrInputCanceled}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}

	if err := cli.restoreFile(f); err != nil {
		t.Fatalf("restoreFile() = %v, want nil on cancel", err)
	}
	if len(tr.restored) != 0 {
		t.Errorf("restored to %v after the rename was canceled", tr.restored)
	}
	assertContent(t, existing, "keep me")
	assertExists(t, f.TrashPath)
}

func TestRestoreFile_ConflictInputErrorPropagates(t *testing.T) {
	tr, f, _, existing := conflictFixture(t)
	p := &scriptedPrompter{inputErr: errors.New("no tty")}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}

	err := cli.restoreFile(f)
	if err == nil || !strings.Contains(err.Error(), "no tty") {
		t.Errorf("restoreFile() = %v, want the input error", err)
	}
	if len(tr.restored) != 0 {
		t.Errorf("restored to %v after the input failed", tr.restored)
	}
	assertContent(t, existing, "keep me")
}

// A new name that is taken too asks once more before restoring over it
func TestRestoreFile_NewNameAlsoTakenAsksAgain(t *testing.T) {
	tr, f, dir, existing := conflictFixture(t)
	taken := filepath.Join(dir, "doc-2.txt")
	if err := os.WriteFile(taken, []byte("keep me too"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &scriptedPrompter{newName: "doc-2.txt", confirm: []bool{false}}
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: p}
	cli.config.Core.Restore.Confirm = false

	if err := cli.restoreFile(f); err != nil {
		t.Fatalf("restoreFile() = %v", err)
	}
	if len(p.confirmPrompts) != 1 || !strings.Contains(p.confirmPrompts[0], "already exists") {
		t.Errorf("prompts = %q, want one warning that the name exists", p.confirmPrompts)
	}
	if len(tr.restored) != 0 {
		t.Errorf("restored to %v after the overwrite was declined", tr.restored)
	}
	assertContent(t, existing, "keep me")
	assertContent(t, taken, "keep me too")
	assertExists(t, f.TrashPath)
}

func TestRestoreFile_RestoreErrorPropagates(t *testing.T) {
	tr := newDiskTrash(t)
	dst := filepath.Join(t.TempDir(), "doc.txt")
	f := tr.add(t, "doc.txt", dst, "content", time.Hour)
	tr.restoreErr = errors.New("disk full")
	cli := &CLI{config: config.NewDefaultConfig(), trash: tr, prompter: &scriptedPrompter{}}
	cli.config.Core.Restore.Confirm = false

	err := cli.restoreFile(f)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Errorf("restoreFile() = %v, want the Restore error", err)
	}
	assertExists(t, f.TrashPath)
}

// Restore says why there is nothing to show once the file browser, which
// reads the trash, ends without a file
func TestRestore_BeforeTheBrowser(t *testing.T) {
	// the file browser ends with what reading the trash returned
	orig := render
	render = func(_ trash.Trash, load ui.Loader, _ ui.RenderOptions) ([]*trash.File, error) {
		_, err := load(nil)
		return nil, err
	}
	t.Cleanup(func() { render = orig })

	t.Run("list error", func(t *testing.T) {
		tr := newDiskTrash(t)
		tr.listErr = errors.New("cannot read trash")
		cli := &CLI{config: config.NewDefaultConfig(), trash: tr}
		if err := cli.Restore(); err == nil || !strings.Contains(err.Error(), "cannot read trash") {
			t.Errorf("Restore() = %v, want the List error", err)
		}
	})
	t.Run("empty", func(t *testing.T) {
		cli := &CLI{config: config.NewDefaultConfig(), trash: newDiskTrash(t)}
		var err error
		out := captureStdout(t, func() { err = cli.Restore() })
		if err != nil || !strings.Contains(out, "The trash is empty") {
			t.Errorf("Restore() = %v, output %q", err, out)
		}
	})
	t.Run("all data files gone", func(t *testing.T) {
		tr := newDiskTrash(t)
		tr.files = []*trash.File{{Name: "gone", TrashPath: filepath.Join(tr.dir, "gone")}}
		cli := &CLI{config: config.NewDefaultConfig(), trash: tr}
		var err error
		out := captureStdout(t, func() { err = cli.Restore() })
		if err != nil || !strings.Contains(out, "Could not find any files") {
			t.Errorf("Restore() = %v, output %q", err, out)
		}
	})
}

func newPutCLI(t *testing.T, forbidden ...string) (*CLI, *diskTrash) {
	t.Helper()
	tr := newDiskTrash(t)
	cfg := config.NewDefaultConfig()
	cfg.Core.Trash.ForbiddenPaths = forbidden
	return &CLI{config: cfg, trash: tr}, tr
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPut_MovesFilesAndReportsMissing(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	writeFile(t, a, "a")
	writeFile(t, b, "b")
	missing := filepath.Join(dir, "missing")
	cli, tr := newPutCLI(t)

	err := cli.Put([]string{a, missing, b})
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Put() = %v, want an error for the missing file", err)
	}
	// the files that exist are still moved
	assertGone(t, a)
	assertGone(t, b)
	assertContent(t, filepath.Join(tr.dir, "a"), "a")
	assertContent(t, filepath.Join(tr.dir, "b"), "b")
}

func TestPut_TrashErrorPropagatesAndKeepsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	writeFile(t, file, "data")
	cli := &CLI{config: config.NewDefaultConfig(), trash: failingTrash{}}

	if err := cli.Put([]string{file}); err == nil || !strings.Contains(err.Error(), "put failed") {
		t.Errorf("Put() = %v, want the Put error", err)
	}
	assertContent(t, file, "data")
}

// Arguments are taken literally: a file named "$FOO" is the file meant,
// not whatever $FOO names. The comment in processFile says environment
// variables are expanded, but expandPath intentionally does not.
func TestPut_DoesNotExpandEnvironmentVariables(t *testing.T) {
	dir := t.TempDir()
	literal := filepath.Join(dir, "$FOO")
	writeFile(t, literal, "literal")
	target := filepath.Join(t.TempDir(), "target")
	writeFile(t, target, "target")
	t.Setenv("FOO", target)
	t.Chdir(dir)
	cli, tr := newPutCLI(t)

	if err := cli.Put([]string{"$FOO"}); err != nil {
		t.Fatalf("Put($FOO) = %v", err)
	}
	assertGone(t, literal)
	assertContent(t, target, "target")
	assertContent(t, filepath.Join(tr.dir, "$FOO"), "literal")
}

// "~" is the shell's to expand; gomi looks for a file named "~"
func TestPut_DoesNotExpandTilde(t *testing.T) {
	home := t.TempDir()
	inHome := filepath.Join(home, "notes")
	writeFile(t, inHome, "notes")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir())
	cli, tr := newPutCLI(t)

	err := cli.Put([]string{"~/notes"})
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Put(~/notes) = %v, want no such file", err)
	}
	assertContent(t, inHome, "notes")
	if len(tr.put) != 0 {
		t.Errorf("moved %v to trash", tr.put)
	}
}

func TestPut_RefusesForbiddenPaths(t *testing.T) {
	root := t.TempDir()
	sys := filepath.Join(root, "sys")
	sub := filepath.Join(sys, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	inSys := filepath.Join(sys, "conf")
	writeFile(t, inSys, "conf")
	other := filepath.Join(root, "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		cwd  string
		arg  string
	}{
		{"absolute", root, inSys},
		{"the forbidden dir itself", root, sys},
		{"relative with ..", sub, filepath.Join("..", "conf")},
		{"relative from outside", other, filepath.Join("..", "sys", "conf")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(tt.cwd)
			cli, tr := newPutCLI(t, sys)
			err := cli.Put([]string{tt.arg})
			if err == nil || !strings.Contains(err.Error(), "forbidden path") {
				t.Errorf("Put(%q) = %v, want a forbidden path error", tt.arg, err)
			}
			if len(tr.put) != 0 {
				t.Errorf("moved %v to trash", tr.put)
			}
			assertContent(t, inSys, "conf")
		})
	}
}

// Forbidden paths in the config may use environment variables
func TestPut_ForbiddenPathFromEnv(t *testing.T) {
	sys := t.TempDir()
	file := filepath.Join(sys, "conf")
	writeFile(t, file, "conf")
	t.Setenv("GOMI_TEST_SYS", sys)
	cli, tr := newPutCLI(t, "$GOMI_TEST_SYS")

	if err := cli.Put([]string{file}); err == nil || !strings.Contains(err.Error(), "forbidden path") {
		t.Errorf("Put() = %v, want a forbidden path error", err)
	}
	if len(tr.put) != 0 {
		t.Errorf("moved %v to trash", tr.put)
	}
	assertContent(t, file, "conf")
}

func TestPut_RefusesDotAndDotDot(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "cwd")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cwd, "file"), "")

	for _, arg := range []string{".", "./", "..", "../", "sub/..", filepath.Join("..", "..")} {
		t.Run(arg, func(t *testing.T) {
			t.Chdir(cwd)
			cli, tr := newPutCLI(t)
			if err := cli.Put([]string{arg}); err == nil || !strings.Contains(err.Error(), "unsafe path") {
				t.Errorf("Put(%q) = %v, want an unsafe path error", arg, err)
			}
			if len(tr.put) != 0 {
				t.Errorf("moved %v to trash", tr.put)
			}
			assertExists(t, filepath.Join(cwd, "file"))
		})
	}
}

func TestPut_RefusesRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no single root on Windows")
	}
	// with no forbidden paths, the unsafe path check alone refuses "/"
	for _, arg := range []string{"/", "//", "/.."} {
		cli, tr := newPutCLI(t)
		if err := cli.Put([]string{arg}); err == nil {
			t.Errorf("Put(%q) = nil, want an error", arg)
		}
		if len(tr.put) != 0 {
			t.Errorf("Put(%q) moved %v to trash", arg, tr.put)
		}
	}
}

// A broken symlink is moved as itself, not refused as missing
func TestPut_BrokenSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join(dir, "gone"), link); err != nil {
		t.Fatal(err)
	}
	cli, tr := newPutCLI(t)

	if err := cli.Put([]string{link}); err != nil {
		t.Fatalf("Put(broken symlink) = %v", err)
	}
	assertGone(t, link)
	if _, err := os.Lstat(filepath.Join(tr.dir, "link")); err != nil {
		t.Errorf("link not in trash: %v", err)
	}
}

// "link/../victim" names victim next to link's target, as rm and the
// kernel resolve it. Cleaning the path lexically turns it into ./victim, a
// different file.
func TestPut_DotDotAfterSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	t.Skip("bug: expandPath and filepath.Abs clean \"link/../victim\" lexically, so gomi trashes ./victim instead of the victim next to link's target (internal/cli/put.go:116, put.go:62)")

	root := t.TempDir()
	far := filepath.Join(root, "far", "inner")
	if err := os.MkdirAll(far, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(far, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	named := filepath.Join(root, "far", "victim") // what link/../victim is
	writeFile(t, named, "named")
	bystander := filepath.Join(root, "victim") // what Clean makes of it
	writeFile(t, bystander, "bystander")
	t.Chdir(root)
	cli, _ := newPutCLI(t)

	if err := cli.Put([]string{filepath.Join("link", "..", "victim")}); err != nil {
		t.Fatalf("Put() = %v", err)
	}
	assertContent(t, bystander, "bystander")
	assertGone(t, named)
}
