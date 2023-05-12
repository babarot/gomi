package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	clilog "github.com/b4b4r07/go-cli-log"
	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/jessevdk/go-flags"
	"github.com/manifoldco/promptui"
	"github.com/rs/xid"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sync/errgroup"
)

// These variables are set in build step
var (
	Version  = "unset"
	Revision = "unset"
)

var gomiPath string
var inventoryPath string

// Load Gomi Path from environment variable
// $XDG_CONFIG_HOME/gomi > $HOME/.gomi
func init() {
	gomiDir := "gomi"
  inventoryFile := "inventory.json"
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		gomiPath = filepath.Join(os.Getenv("XDG_CONFIG_HOME"), gomiDir)
    inventoryPath = filepath.Join(gomiPath, inventoryFile)
	} else {
		gomiPath = filepath.Join(os.Getenv("HOME"), "."+gomiDir)
    inventoryPath = filepath.Join(gomiPath, inventoryFile)
	}
}

// Option represents application options
type Option struct {
	Restore      bool     `short:"b" long:"restore" description:"Restore deleted file"`
	RestoreGroup bool     `short:"B" long:"restore-by-group" description:"Restore deleted files based on one operation"`
	Version      bool     `long:"version" description:"Show version"`
	RmOption     RmOption `group:"Dummy options"`
}

// RmOption represents rm command option
// This should be not conflicts with app option
type RmOption struct {
	Interactive bool `short:"i" description:"To make compatible with rm command"`
	Recursive   bool `short:"r" description:"To make compatible with rm command"`
	Force       bool `short:"f" description:"To make compatible with rm command"`
	Directory   bool `short:"d" description:"To make compatible with rm command"`
	Verbose     bool `short:"v" description:"To make compatible with rm command"`
}

// Inventory represents the log data of deleted objects
type Inventory struct {
	Path  string `json:"path"`
	Files []File `json:"files"`
}

// File represents the metadata of deleted object itself
type File struct {
	Name      string    `json:"name"`     // file.go
	ID        string    `json:"id"`       // asfasfafd
	GroupID   string    `json:"group_id"` // zoapompji
	From      string    `json:"from"`     // $PWD/file.go
	To        string    `json:"to"`       // ~/.gomi/2020/01/16/zoapompji/file.go.asfasfafd
	Timestamp time.Time `json:"timestamp"`
}

// CLI represents this application itself
type CLI struct {
	Option    Option
	Inventory Inventory
	Stdout    io.Writer
	Stderr    io.Writer
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	clilog.Env = "GOMI_LOG"
	clilog.SetOutput()
	defer log.Printf("[INFO] finish main function")

	log.Printf("[INFO] Version: %s (%s)", Version, Revision)
	log.Printf("[INFO] gomiPath: %s", gomiPath)
	log.Printf("[INFO] inventoryPath: %s", inventoryPath)
	log.Printf("[INFO] Args: %#v", args)

	var opt Option
	args, err := flags.ParseArgs(&opt, args)
	if err != nil {
		return 2
	}

	cli := CLI{
		Option:    opt,
		Inventory: Inventory{Path: inventoryPath},
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}

	if err := cli.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	return 0
}

// Run runs gomi main logic
func (c CLI) Run(args []string) error {
	c.Inventory.Open()

	switch {
	case c.Option.Version:
		fmt.Fprintf(c.Stdout, "%s (%s)\n", Version, Revision)
		return nil
	case c.Option.Restore:
		return c.Restore()
	case c.Option.RestoreGroup:
		return c.RestoreGroup()
	default:
	}

	return c.Remove(args)
}

// Restore moves deleted file/dir to original place
func (c CLI) Restore() error {
	file, err := c.FilePrompt()
	if err != nil {
		return err
	}
	defer c.Inventory.Delete(file)
	_, err = os.Stat(file.From)
	if err == nil {
		// already exists so to prevent to overwrite
		// add id to the end of filename
		// TODO: Ask to overwrite?
		// e.g. using github.com/AlecAivazis/survey
		file.From = file.From + "." + file.ID
	}
	log.Printf("[DEBUG] restoring %q -> %q", file.To, file.From)
	return os.Rename(file.To, file.From)
}

// RestoreGroup moves deleted file(s)/dir(s) which are deleted in one operation to original place
func (c CLI) RestoreGroup() error {
	group, err := c.GroupPrompt()
	if err != nil {
		return err
	}
	defer func() {
		for _, file := range group.Files {
			c.Inventory.Delete(file)
		}
	}()
	var eg errgroup.Group
	for _, file := range group.Files {
		file := file
		_, err = os.Stat(file.From)
		if err == nil {
			// already exists so to prevent to overwrite
			// add id to the end of filename
			file.From = file.From + "." + file.ID
		}
		eg.Go(func() error {
			log.Printf("[DEBUG] restoring %q -> %q", file.To, file.From)
			return os.Rename(file.To, file.From)
		})
	}

	return eg.Wait()
}

// Remove moves files to gomi dir
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
			file.ToJSON(&buf)
			log.Printf("[DEBUG] generating file metadata: %s", buf.String())

			files[i] = file
			os.MkdirAll(filepath.Dir(file.To), 0777)
			log.Printf("[DEBUG] moving %q -> %q", file.From, file.To)
			return os.Rename(file.From, file.To)
		})
	}
	defer c.Inventory.Save(files)

	defer eg.Wait()
	if c.Option.RmOption.Force {
		// ignore errors when given rm -f option
		return nil
	}

	return eg.Wait()
}

// Open opens inventory file
func (i *Inventory) Open() error {
	log.Printf("[DEBUG] opening inventory")
	f, err := os.Open(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&i)
}

// Update updates inventory file (this may overwrite the inventory file)
func (i *Inventory) Update(files []File) error {
	log.Printf("[DEBUG] updating inventory")
	f, err := os.Create(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = files
	return json.NewEncoder(f).Encode(&i)
}

// Save updates inventory file (this should not overwrite the inventory file)
func (i *Inventory) Save(files []File) error {
	log.Printf("[DEBUG] saving inventory")
	f, err := os.Create(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = append(i.Files, files...)
	return json.NewEncoder(f).Encode(&i)
}

// Delete deletes a file from the inventory file
// This should not delete the inventory file itself
func (i *Inventory) Delete(target File) error {
	log.Printf("[DEBUG] deleting %v from inventory", target)
	var files []File
	for _, file := range i.Files {
		if file.ID == target.ID {
			continue
		}
		files = append(files, file)
	}
	return i.Update(files)
}

// Filter filters inventory entries based on given function
func (i *Inventory) Filter(f func(File) bool) {
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

// ToJSON writes json objects based on File
func (f File) ToJSON(w io.Writer) {
	out, err := json.Marshal(&f)
	if err != nil {
		return
	}
	fmt.Fprint(w, string(out))
}

func isBinary(path string) bool {
	detectedMIME, err := mimetype.DetectFile(path)
	if err != nil {
		return true
	}
	isBinary := true
	for mime := detectedMIME; mime != nil; mime = mime.Parent() {
		if mime.Is("text/plain") {
			isBinary = false
		}
	}
	return isBinary
}

func head(path string) string {
	max := 5
	wrap := func(line string) string {
		line = strings.ReplaceAll(line, "\t", "  ")
		id := int(os.Stdout.Fd())
		width, _, _ := terminal.GetSize(id)
		if width < 10 {
			return line
		}
		if len(line) < width-10 {
			return line
		}
		return line[:width-10] + "..."
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "(panic: not found)"
	}
	content := func(lines []string) string {
		if len(lines) == 0 {
			return "(no content)"
		}
		var content string
		var i int
		for _, line := range lines {
			i++
			content += fmt.Sprintf("  %s\n", wrap(line))
			if i > max {
				content += "  ...\n"
				break
			}
		}
		return content
	}
	var lines []string
	switch {
	case fi.IsDir():
		lines = []string{"(directory)"}
		dirs, _ := ioutil.ReadDir(path)
		for _, dir := range dirs {
			lines = append(lines, fmt.Sprintf("%s\t%s", dir.Mode().String(), dir.Name()))
		}
	default:
		if isBinary(path) {
			return "(binary file)"
		}
		lines = []string{""}
		fp, _ := os.Open(path)
		defer fp.Close()
		s := bufio.NewScanner(fp)
		for s.Scan() {
			lines = append(lines, s.Text())
		}
	}
	return content(lines)
}

// FilePrompt prompts inventory entries, and select one and return it
func (c CLI) FilePrompt() (File, error) {
	// Filter out invalid logs
	c.Inventory.Filter(func(file File) bool {
		return file.ID != ""
	})

	files := c.Inventory.Files
	if len(files) == 0 {
		return File{}, errors.New("no deleted files found")
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Timestamp.After(files[j].Timestamp)
	})

	funcMap := promptui.FuncMap
	funcMap["time"] = humanize.Time
	funcMap["head"] = head
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   promptui.IconSelect + " {{ .Name | cyan }}",
		Inactive: "  {{ .Name | faint }}",
		Selected: promptui.IconGood + " {{ .Name }}",
		Details: `
{{ "Name:" | faint }}	{{ .Name }}
{{ "Path:" | faint }}	{{ .From }}
{{ "DeletedAt:" | faint }}	{{ .Timestamp | time }}
{{ "Content:" | faint }}	{{ .To | head }}
		`,
		FuncMap: funcMap,
	}

	searcher := func(input string, index int) bool {
		file := files[index]
		name := strings.Replace(strings.ToLower(file.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:             "Which to restore?",
		Items:             files,
		Templates:         templates,
		Searcher:          searcher,
		StartInSearchMode: true,
		HideSelected:      true,
	}

	i, _, err := prompt.Run()
	return files[i], err
}

// Group represents files ([]File) deleted by one operation
type Group struct {
	ID        string
	Dir       string
	Timestamp time.Time
	Files     []File
}

// GroupPrompt prompts inventory entries which are grouped by one operation
// and select one group and return it
func (c CLI) GroupPrompt() (Group, error) {
	// Filter out invalid logs
	c.Inventory.Filter(func(file File) bool {
		return file.ID != ""
	})

	files := c.Inventory.Files
	if len(files) == 0 {
		return Group{}, errors.New("no deleted files found")
	}

	m := map[string][]File{}
	for _, file := range c.Inventory.Files {
		m[file.GroupID] = append(m[file.GroupID], file)
	}

	hasMultiDirs := func(files []File) bool {
		if len(files) == 0 {
			return false
		}
		var dirs []string
		unique := map[string]bool{}
		for _, file := range files {
			dir := filepath.Dir(file.From)
			if !unique[dir] {
				unique[dir] = true
				dirs = append(dirs, dir)
			}
		}
		if len(dirs) > 1 {
			return true
		}
		return false
	}

	var groups []Group
	for id, files := range m {
		dir := filepath.Dir(files[0].From)
		if hasMultiDirs(files) {
			dir = "(multiple directories)"
		}
		groups = append(groups, Group{
			ID:        id,
			Dir:       dir,
			Timestamp: files[0].Timestamp,
			Files:     files,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Timestamp.After(groups[j].Timestamp)
	})

	funcMap := promptui.FuncMap
	funcMap["time"] = humanize.Time

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   promptui.IconSelect + " {{ .Dir | cyan }}",
		Inactive: "  {{ .Dir | faint }}",
		Selected: promptui.IconGood + " {{ .Dir }}",
		Details: `
{{ "DeletedAt:" | faint }}	{{ .Timestamp | time }}
{{ "Files:" | faint }}
    {{- range .Files }}
    - {{ .From }}
    {{- end }}
`,
		FuncMap: funcMap,
	}

	searcher := func(input string, index int) bool {
		files := groups[index].Files
		contains := func(Files []File, input string) bool {
			for _, file := range files {
				// ignorecase
				from := strings.ToLower(file.From)
				input = strings.ToLower(input)
				if strings.Contains(from, input) {
					return true
				}
			}
			return false
		}
		return contains(files, input)
	}

	prompt := promptui.Select{
		Label:             "Which to restore?",
		Items:             groups,
		Templates:         templates,
		Searcher:          searcher,
		StartInSearchMode: true,
		HideSelected:      true,
	}

	i, _, err := prompt.Run()
	return groups[i], err
}
