package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/go-units"
	"github.com/gobwas/glob"
	"github.com/google/uuid"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/k0kubun/pp"
	"github.com/rs/xid"
	"github.com/samber/lo"
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
//   1. a .gh-dash.yml file if inside a git repo
//   2. $GH_DASH_CONFIG env var
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

	excludes ConfigExclude
	path     string
}

type File struct {
	Name      string    `json:"name"`         // file.go
	ID        string    `json:"id"`           // asfasfafd
	RunID     string    `json:"operation_id"` // zoapompji
	From      string    `json:"from"`         // $PWD/file.go
	To        string    `json:"to"`           // ~/.gomi/2020/01/16/zoapompji/file.go.asfasfafd
	Timestamp time.Time `json:"timestamp"`
}

func (f File) isSelected() bool {
	return selectionManager.Contains(f)
}

type CLI struct {
	Config    Config
	Option    Option
	Stdout    io.Writer
	Stderr    io.Writer
	inventory inventory
	// logger    *slog.Logger
	runID string
}

func main() {
	if err := runMain(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", appName, err)
		slog.Error(err.Error())
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
			fp = "gitfetcher.log"
		}
	}

	var writer io.Writer
	if file, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		writer = file
	} else {
		errs = append(errs, err)
		writer = os.Stdout
	}

	// uuidV1, err := uuid.NewUUID()
	// if err != nil {
	// }

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(
		slog.New(handler).With("id", runID()),
	)
	if len(errs) > 0 {
		slog.Error("Log setup failed.", LogErrAttr(errors.Join(errs...)))
	}
}

func runMain() error {
	log.SetOutput(logOutput(envGomiLog))
	defer log.Printf("[INFO] finish main function")
	defer slog.Debug("finished")

	log.Printf("[INFO] version: %s (%s)", Version, Revision)
	log.Printf("[INFO] gomiPath: %s", gomiPath)
	log.Printf("[INFO] inventoryPath: %s", inventoryPath)

	slog.Debug("version", slog.String("version", Version), slog.String("revision", Revision))

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
		Config:    cfg,
		Option:    opt,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		inventory: inventory{path: inventoryPath, excludes: cfg.Inventory.Exclude},
		runID:     runID(),
	}

	log.Printf("[INFO] Args: %#v", args)
	slog.Debug("CLI", "args", args, "opt", opt)
	return cli.Run(args)
}

func (c CLI) Run(args []string) error {
	if err := c.inventory.open(); err != nil {
		return err
	}

	switch {
	case c.Option.Version:
		fmt.Fprintf(c.Stdout, "%s %s (%s)\n", appName, Version, Revision)
		return nil
	case c.Option.Restore:
		slog.Debug("open restore view")
		return c.Restore()
	default:
	}

	return c.Put(args)
}

func (c CLI) initModel() model {
	const defaultWidth = 20

	excludedFiles := c.inventory.exclude()
	var files []list.Item
	for _, file := range excludedFiles {
		files = append(files, file)
	}

	// TODO: configable
	// l := list.New(files, ClassicDelegate{}, defaultWidth, listHeight)
	l := list.New(files, NewRestoreDelegate(c.Config), defaultWidth, listHeight)

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
		files:    excludedFiles,
		cli:      &c,
		list:     l,
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
	log.Printf("[DEBUG] restoring %q -> %q", file.To, file.From)
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
			log.Printf("[DEBUG] generating file metadata: %s", buf.String())

			files[i] = file
			_ = os.MkdirAll(filepath.Dir(file.To), 0777)
			log.Printf("[DEBUG] moving %q -> %q", file.From, file.To)
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
	log.Printf("[DEBUG] opening inventory")
	f, err := os.Open(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&i); err != nil {
		return err
	}
	log.Printf("[DEBUG] get inventory version: %d", i.Version)
	return nil
}

func (i *inventory) update(files []File) error {
	log.Printf("[DEBUG] updating inventory")
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
	log.Printf("[DEBUG] saving inventory")
	f, err := os.Create(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = append(i.Files, files...)
	i.setVersion()
	return json.NewEncoder(f).Encode(&i)
}

func (i inventory) exclude() []File {
	// do not overwrite original slices
	// because remove them from inventory file actually
	// when updating inventory
	files := i.Files
	files = lo.Reject(files, func(file File, index int) bool {
		return slices.Contains(i.excludes.Files, file.Name)
	})
	files = lo.Reject(files, func(file File, index int) bool {
		for _, pat := range i.excludes.Patterns {
			if regexp.MustCompile(pat).MatchString(file.Name) {
				return true
			}
		}
		for _, g := range i.excludes.Globs {
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
		for _, s := range i.excludes.SizeBelow {
			below, err := units.FromHumanSize(s)
			if err != nil {
				continue
			}
			if size <= below {
				return true
			}
		}
		for _, s := range i.excludes.SizeAbove {
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
	return files
}

func (i *inventory) remove(target File) error {
	log.Printf("[DEBUG] deleting %v from inventory", target)
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
		log.Printf("[DEBUG] set inventory version: %d", inventoryVersion)
		i.Version = inventoryVersion
	}
}

func (i *inventory) filter(f func(File) bool) {
	files := make([]File, 0)
	for _, file := range i.Files {
		if f(file) {
			files = append(files, file)
		}
	}
	i.Files = files
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

func logOutput(env string) io.Writer {
	levels := []logutils.LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}
	minLevel := os.Getenv(env)
	if len(minLevel) == 0 {
		minLevel = "ERROR" // default log level
	}

	// default log writer is null
	writer := ioutil.Discard
	if minLevel != "" {
		writer = os.Stderr
	}

	filter := &logutils.LevelFilter{
		Levels:   levels,
		MinLevel: logutils.LogLevel(strings.ToUpper(minLevel)),
		Writer:   writer,
	}

	return filter
}

// func (c *CLI) configureLog() {
// 	writers := []io.Writer{logFile}
// 	if c.Bool("print-logs") || flag.SST_PRINT_LOGS {
// 		writers = append(writers, os.Stderr)
// 	}
// 	writer := io.MultiWriter(writers...)
// 	slog.SetDefault(
// 		slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
// 			Level: slog.LevelInfo,
// 		})),
// 	)
// 	debug.SetCrashOutput(logFile, debug.CrashOptions{})
// }

var debugLogger = sync.OnceValue(func() *slog.Logger {
	var w io.Writer = os.Stderr
	cached, err := os.UserCacheDir()
	if err == nil {
		logf := filepath.Join(cached, "blogsync", "tracedump.log")
		if err := os.MkdirAll(filepath.Dir(logf), 0755); err == nil {
			if f, err := os.OpenFile(logf, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				log.Printf("trace dumps are output to %s\n", logf)
				w = f
			}
		}
	}
	uuidV1, _ := uuid.NewUUID()
	// if err != nil {
	// 	return err
	// }
	// logger := slog.New(slog.NewJSONHandler(os.Stderr, logOption)).With("id", uuidV1)
	// slog.SetDefault(logger)
	slog.Debug("version", slog.String("version", Version))
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})).With("id", uuidV1)
})
