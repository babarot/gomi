package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/go-units"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gobwas/glob"
	"github.com/jessevdk/go-flags"
	"github.com/k0kubun/pp"
	"github.com/k1LoW/duration"
	"github.com/lmittmann/tint"
	"github.com/rs/xid"
	"github.com/samber/lo"
	slogmulti "github.com/samber/slog-multi"
	"golang.org/x/sync/errgroup"
)

const (
	appName    = "gomi"
	gomiDotDir = ".gomi"
	envGomiLog = "GOMI_LOG"

	inventoryVersion = 1
	inventoryFile    = "inventory.json"
)

const (
	datefmtRel = "relative"
	datefmtAbs = "absolute"
)

const listHeight = 20

// These variables are set in build step
var (
	Version  = "unset"
	Revision = "unset"
)

var (
	gomiPath      = filepath.Join(os.Getenv("HOME"), gomiDotDir)
	inventoryPath = filepath.Join(gomiPath, inventoryFile)
)

type Option struct {
	Restore  bool     `short:"b" long:"restore" description:"Restore deleted file"`
	Version  bool     `long:"version" description:"Show version"`
	Config   string   `long:"config" description:"Path to config file" default:""`
	RmOption RmOption `group:"Dummy Options (compatible with rm)"`
}

// use this configuration file
// (default lookup:
//   1. a .gomi.yaml file if inside a git repo
//   2. $GOMI_ENV_CONFIG env var
//   3. $XDG_CONFIG_HOME/gh-dash/config.yml
// )

// This should be not conflicts with app option
// https://man7.org/linux/man-pages/man1/rm.1.html
type RmOption struct {
	Interactive   bool `short:"i" description:"(dummy) prompt before every removal"`
	Recursive     bool `short:"r" long:"recursive" description:"(dummy) remove directories and their contents recursively"`
	Force         bool `short:"f" long:"force" description:"(dummy) ignore nonexistent files, never prompt"`
	Directory     bool `short:"d" long:"dir" description:"(dummy) remove empty directories"`
	Verbose       bool `short:"v" long:"verbose" description:"(dummy) explain what is being done"`
	OneFileSystem bool `long:"one-file-system" description:"(dummy) when removing a hierarchy recursively, skip any directory\n....... that is on a file system different from that of the\n....... corresponding command line argument"`
}

// inventory represents the log data of deleted objects
type inventory struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`

	config ConfigInventory
	path   string
}

type File struct {
	Name      string    `json:"name"`
	ID        string    `json:"id"`
	RunID     string    `json:"group_id"` // to keep backward compatible
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
}

func (f File) isSelected() bool {
	return selectionManager.Contains(f)
}

type CLI struct {
	config    Config
	Option    Option
	inventory inventory
	runID     string
}

func main() {
	if err := runMain(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		slog.Error("failed to run cli", "error", err)
		os.Exit(1)
	}
}

var runID = sync.OnceValue(func() string {
	id := xid.New().String()
	return id
})

func init() {
	var errs []error
	fp, ok := os.LookupEnv("LOGS_DIRECTORY")
	if !ok {
		var err error
		fp, err = xdg.StateFile("gomi/log")
		if err != nil {
			errs = append(errs, err)
			fp = "gomi.log"
		}
	}

	var fw, cw io.Writer
	if file, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fw = file
	} else {
		errs = append(errs, err)
		fw = io.Discard
	}

	if os.Getenv("DEBUG") == "" {
		cw = io.Discard
	} else {
		cw = os.Stderr
	}

	handler := NewWrapHandler(
		slog.NewJSONHandler(fw, &slog.HandlerOptions{Level: slog.LevelDebug}),
		func() []slog.Attr {
			return []slog.Attr{
				slog.String("run_id", runID()),
			}
		})

	logger := slog.New(
		slogmulti.Fanout(
			handler,
			tint.NewHandler(cw, &tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.Kitchen,
			}),
		),
	)
	slog.SetDefault(logger)

	if len(errs) > 0 {
		slog.Error("Log setup failed.", LogErrAttr(errors.Join(errs...)))
	}
}

func runMain() error {
	defer slog.Debug("finished main function")
	slog.Debug("runMain running successfully",
		slog.Group(
			"metadata",
			slog.String("version", Version),
			slog.String("revision", Revision),
		),
	)

	var opt Option
	parser := flags.NewParser(&opt, flags.Default)
	parser.Name = appName
	parser.Usage = "[OPTIONS] files..."
	args, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			return nil
		}
		return err
	}

	cfg, err := parseConfig(opt.Config)
	if err != nil {
		return err
	}

	cli := CLI{
		config:    cfg,
		Option:    opt,
		inventory: inventory{path: inventoryPath, config: cfg.Inventory},
		runID:     runID(),
	}
	return cli.Run(args)
}

func (c CLI) Run(args []string) error {
	slog.Debug("cli.Run starts")
	if err := c.inventory.open(); err != nil {
		return err
	}

	switch {
	case c.Option.Version:
		fmt.Fprintf(os.Stdout, "%s %s (%s)\n", appName, Version, Revision)
		return nil
	case c.Option.Restore:
		slog.Debug("open restore view")
		return c.Restore()
	default:
	}

	return c.Put(args)
}

func newViewportModel(file File, width, height int, cmd string, hl bool, cs string) (viewport.Model, error) {
	getFileContent := func(path string) string {
		content := "cannot preview"
		fi, err := os.Stat(path)
		if err != nil {
			return content
		}
		if fi.IsDir() {
			input := cmd
			out, _, err := runBash(input)
			if err != nil {
				slog.Error(fmt.Sprintf("command failed: %s", input), "error", err)
			}
			return out
		}
		mtype, err := mimetype.DetectFile(path)
		if err != nil {
			return content
		}
		if !strings.Contains(mtype.String(), "text/plain") {
			return content
		}
		f, err := os.Open(file.To)
		if err != nil {
			return content
		}
		defer f.Close()

		var fileContent strings.Builder
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fileContent.WriteString(scanner.Text() + "\n")
		}
		if err := scanner.Err(); err != nil {
			return content
		}
		return fileContent.String()
	}

	content := getFileContent(file.To)
	if hl {
		content, _ = highlight(content, file.Name, cs)
	}
	viewportModel := viewport.New(width, height)
	viewportModel.KeyMap = viewport.KeyMap{
		Up: key.NewBinding(
			key.WithKeys("ctrl+p", "ctrl+k"), // second keymap is undocumented
			key.WithHelp("ctrl+p", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("ctrl+n", "ctrl+j"), // second keymap is undocumented
			key.WithHelp("ctrl+n", "down"),
		),
	}
	viewportModel.SetContent(content)
	return viewportModel, nil
}

func (c CLI) initModel() model {
	slog.Debug("initModel starts")
	const defaultWidth = 20

	filteredFiles := c.inventory.filter()
	var files []list.Item
	for _, file := range filteredFiles {
		files = append(files, file)
	}

	// TODO: configable
	// l := list.New(files, ClassicDelegate{}, defaultWidth, listHeight)
	l := list.New(files, NewRestoreDelegate(c.config), defaultWidth, listHeight)

	// TODO: which one?
	// l.Paginator.Type = paginator.Arabic

	l.Title = ""
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{listAdditionalKeys.Enter, listAdditionalKeys.Space}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{listAdditionalKeys.Enter, listAdditionalKeys.Space, keys.Quit, keys.Select, keys.DeSelect}
	}
	l.DisableQuitKeybindings()
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	m := model{
		navState: INVENTORY_LIST,
		datefmt:  datefmtRel,
		files:    filteredFiles,
		cli:      &c,
		// models
		list:     l,
		viewport: viewport.Model{},
	}
	return m
}

func (c CLI) Restore() error {
	m := c.initModel()
	returnModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	files := returnModel.(model).choices
	if returnModel.(model).navState == QUITTING {
		fmt.Println("bye!")
		return nil
	}

	pp.Println(files)
	file := files[0]
	return nil

	defer c.inventory.remove(file)
	if _, err := os.Stat(file.From); err == nil {
		// already exists so to prevent to overwrite
		// add id to the end of filename
		// TODO: Ask to overwrite?
		// e.g. using github.com/AlecAivazis/survey
		file.From = file.From + "." + file.ID
	}
	return os.Rename(file.To, file.From)
}

func (c CLI) Put(args []string) error {
	if len(args) == 0 {
		return errors.New("too few arguments")
	}

	files := make([]File, len(args))

	var eg errgroup.Group

	for i, arg := range args {
		i, arg := i, arg // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			_, err := os.Stat(arg)
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: no such file or directory", arg)
			}
			file, err := getFileMetadata(c.runID, arg)
			if err != nil {
				return err
			}

			// For debugging
			var buf bytes.Buffer
			file.toJSON(&buf)
			slog.Debug(fmt.Sprintf("generating file metadata: %s", buf.String()))

			files[i] = file
			_ = os.MkdirAll(filepath.Dir(file.To), 0777)
			slog.Debug(fmt.Sprintf("moving %q -> %q", file.From, file.To))
			return os.Rename(file.From, file.To)
		})
	}
	defer c.inventory.save(files)

	defer eg.Wait()
	if c.Option.RmOption.Force {
		// ignore errors when given rm -f option
		return nil
	}

	return eg.Wait()
}

func (i *inventory) open() error {
	slog.Debug(fmt.Sprintf("open inventory file: %s", i.path))
	f, err := os.Open(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&i); err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("inventory version: %d", i.Version))
	return nil
}

func (i *inventory) update(files []File) error {
	slog.Debug("updating inventory")
	f, err := os.Create(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = files
	i.setVersion()
	return json.NewEncoder(f).Encode(&i)
}

func (i *inventory) save(files []File) error {
	slog.Debug("saving inventory")
	f, err := os.Create(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = append(i.Files, files...)
	i.setVersion()
	return json.NewEncoder(f).Encode(&i)
}

func (i inventory) filter() []File {
	// do not overwrite original slices
	// because remove them from inventory file actually
	// when updating inventory
	files := i.Files
	files = lo.Reject(files, func(file File, index int) bool {
		return slices.Contains(i.config.Exclude.Files, file.Name)
	})
	files = lo.Reject(files, func(file File, index int) bool {
		for _, pat := range i.config.Exclude.Patterns {
			if regexp.MustCompile(pat).MatchString(file.Name) {
				return true
			}
		}
		for _, g := range i.config.Exclude.Globs {
			if glob.MustCompile(g).Match(file.Name) {
				return true
			}
		}
		return false
	})
	files = lo.Reject(files, func(file File, index int) bool {
		size, err := DirSize(file.To)
		if err != nil {
			return false // false positive
		}
		for _, s := range i.config.Exclude.SizeBelow {
			below, err := units.FromHumanSize(s)
			if err != nil {
				continue
			}
			if size <= below {
				return true
			}
		}
		for _, s := range i.config.Exclude.SizeAbove {
			above, err := units.FromHumanSize(s)
			if err != nil {
				continue
			}
			if above <= size {
				return true
			}
		}
		return false
	})
	files = lo.Filter(files, func(file File, index int) bool {
		for _, input := range i.config.Include.Durations {
			d, err := duration.Parse(input)
			if err != nil {
				slog.Error("duration.Parse failed", "input", input, "error", err)
				return false
			}
			return time.Since(file.Timestamp) < d
		}
		return false
	})
	return files
}

func (i *inventory) remove(target File) error {
	slog.Debug("deleting from inventory")
	var files []File
	for _, file := range i.Files {
		if file.ID == target.ID {
			continue
		}
		files = append(files, file)
	}
	return i.update(files)
}

func (i *inventory) setVersion() {
	if i.Version == 0 {
		i.Version = inventoryVersion
	}
}

func getFileMetadata(runID string, arg string) (File, error) {
	name := filepath.Base(arg)
	from, err := filepath.Abs(arg)
	if err != nil {
		return File{}, err
	}
	id := xid.New().String()
	now := time.Now()
	return File{
		Name:  name,
		ID:    id,
		RunID: runID,
		From:  from,
		To: filepath.Join(
			gomiPath,
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
			runID,
			fmt.Sprintf("%s.%s", name, id),
		),
		Timestamp: now,
	}, nil
}

// toJSON writes json objects based on File
func (f File) toJSON(w io.Writer) {
	out, err := json.Marshal(&f)
	if err != nil {
		return
	}
	fmt.Fprint(w, string(out))
}
