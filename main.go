package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	// "github.com/babarot/gomi/ui"
	// "github.com/barthr/redo/ui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/k0kubun/pp"
	"github.com/rs/xid"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

const (
	appName = "gomi"
	gomiDir = ".gomi"
	envLog  = "GOMI_LOG"

	inventoryVersion = 1
	inventoryFile    = "inventory.json"
)

const listHeight = 20

var excludeItems = []string{
	".DS_Store",
	"oil:",
}

// These variables are set in build step
var (
	Version  = "unset"
	Revision = "unset"
)

var (
	gomiPath      = filepath.Join(os.Getenv("HOME"), gomiDir)
	inventoryPath = filepath.Join(gomiPath, inventoryFile)
)

type Option struct {
	Restore      bool     `short:"b" long:"restore" description:"Restore deleted file"`
	RestoreGroup bool     `short:"B" long:"restore-by-group" description:"Restore deleted files based on one operation"`
	Version      bool     `long:"version" description:"Show version"`
	RmOption     RmOption `group:"Dummy Options (compatible with rm)"`
}

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
	Path    string `json:"path"`
	Files   []File `json:"files"`
}

type File struct {
	Name      string    `json:"name"`     // file.go
	ID        string    `json:"id"`       // asfasfafd
	GroupID   string    `json:"group_id"` // zoapompji
	From      string    `json:"from"`     // $PWD/file.go
	To        string    `json:"to"`       // ~/.gomi/2020/01/16/zoapompji/file.go.asfasfafd
	Timestamp time.Time `json:"timestamp"`
}

func (f File) isSelected() bool {
	return selectionManager.Contains(f)
}

type CLI struct {
	Option    Option
	inventory inventory
	Stdout    io.Writer
	Stderr    io.Writer
}

func main() {
	if err := runMain(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] error occured while processing %s: %v\n", appName, err)
		os.Exit(1)
	}
}

func runMain() error {
	log.SetOutput(logOutput(envLog))
	defer log.Printf("[INFO] finish main function")

	log.Printf("[INFO] Version: %s (%s)", Version, Revision)
	log.Printf("[INFO] gomiPath: %s", gomiPath)
	log.Printf("[INFO] inventoryPath: %s", inventoryPath)

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

	cli := CLI{
		Option:    opt,
		inventory: inventory{Path: inventoryPath},
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}

	log.Printf("[INFO] Args: %#v", args)
	return cli.Run(args)
}

func (c CLI) Run(args []string) error {
	c.inventory.open()

	switch {
	case c.Option.Version:
		fmt.Fprintf(c.Stdout, "%s %s (%s)\n", appName, Version, Revision)
		return nil
	case c.Option.Restore:
		return c.Restore()
	case c.Option.RestoreGroup:
		return nil
	default:
	}

	return c.Remove(args)
}

func (c CLI) initModel() model {
	const defaultWidth = 20

	var files []list.Item
	for _, file := range c.inventory.Files {
		files = append(files, file)
	}

	// TODO: configable
	// l := list.New(files, ClassicDelegate{}, defaultWidth, listHeight)
	l := list.New(files, NewRestoreDelegate(), defaultWidth, listHeight)

	// TODO: which one?
	// l.Paginator.Type = paginator.Arabic
	l.Title = ""
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{listAdditionalKeys.Enter, listAdditionalKeys.Info}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{listAdditionalKeys.Enter, listAdditionalKeys.Info, keys.Quit, keys.Select, keys.DeSelect}
	}
	l.DisableQuitKeybindings()
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	m := model{
		navState: INVENTORY_LIST,
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

func (c CLI) Remove(args []string) error {
	if len(args) == 0 {
		return errors.New("too few arguments")
	}

	files := make([]File, len(args))
	groupID := xid.New().String()

	var eg errgroup.Group

	for i, arg := range args {
		i, arg := i, arg // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			_, err := os.Stat(arg)
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: no such file or directory", arg)
			}
			file, err := makeFile(groupID, arg)
			if err != nil {
				return err
			}

			// For debugging
			var buf bytes.Buffer
			file.toJSON(&buf)
			log.Printf("[DEBUG] generating file metadata: %s", buf.String())

			files[i] = file
			os.MkdirAll(filepath.Dir(file.To), 0777)
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
	f, err := os.Open(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&i); err != nil {
		return err
	}
	log.Printf("[DEBUG] get inventory version: %d", i.Version)
	log.Printf("[DEBUG] filter out: $#v", excludeItems)
	i.exclude()
	return nil
}

func (i *inventory) update(files []File) error {
	log.Printf("[DEBUG] updating inventory")
	f, err := os.Create(i.Path)
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
	f, err := os.Create(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = append(i.Files, files...)
	i.setVersion()
	return json.NewEncoder(f).Encode(&i)
}

func (i *inventory) exclude() {
	i.Files = lo.Filter(i.Files, func(file File, index int) bool {
		return !slices.Contains(excludeItems, file.Name)
	})
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

func makeFile(groupID string, arg string) (File, error) {
	id := xid.New().String()
	name := filepath.Base(arg)
	from, err := filepath.Abs(arg)
	if err != nil {
		return File{}, err
	}
	now := time.Now()
	return File{
		Name:    name,
		ID:      id,
		GroupID: groupID,
		From:    from,
		To: filepath.Join(
			gomiPath,
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
			groupID, fmt.Sprintf("%s.%s", name, id),
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
