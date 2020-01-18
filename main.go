package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"
	"github.com/manifoldco/promptui"
	"github.com/rs/xid"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sync/errgroup"
)

const gomiDir = ".gomi"

var (
	gomiPath      = filepath.Join(os.Getenv("HOME"), gomiDir)
	inventoryFile = "inventory.json"
	inventoryPath = filepath.Join(gomiPath, inventoryFile)
)

type Option struct {
	Restore  bool     `short:"b" long:"restore" description:"Restore deleted file"`
	RmOption RmOption `group:"Emulation Options for rm command"`
}

type RmOption struct {
	Interactive bool `short:"i" description:"Emulate -i option of rm command"`
	Recursive   bool `short:"r" description:"Emulate -r option of rm command"`
	Force       bool `short:"f" description:"Emulate -f option of rm command"`
}

type Inventory struct {
	Path  string `json:"path"`
	Files []File `json:"files"`
}

type File struct {
	Name      string    `json:"name"`     // file.go
	ID        string    `json:"id"`       // asfasfafd
	GroupID   string    `json:"group_id"` // zoapompji
	From      string    `json:"from"`     // $PWD/file.go
	To        string    `json:"to"`       // ~/.gomi/2020/01/16/zoapompji/file.go.asfasfafd
	Timestamp time.Time `json:"timestamp"`
}

type CLI struct {
	Option    Option
	Inventory Inventory
}

func (i *Inventory) Open() error {
	f, err := os.Open(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&i)
}

func (i *Inventory) Update(files []File) error {
	f, err := os.Create(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = files
	return json.NewEncoder(f).Encode(&i)
}

func (i *Inventory) Save(files []File) error {
	f, err := os.Create(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = append(i.Files, files...)
	return json.NewEncoder(f).Encode(&i)
}

func (i *Inventory) Delete(target File) error {
	var files []File
	for _, file := range i.Files {
		if file.ID == target.ID {
			continue
		}
		files = append(files, file)
	}
	return i.Update(files)
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

func head(path string) string {
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
	var content string
	if fi.IsDir() {
		return "(directory)"
	} else {
		content += "\n"
		var i int
		fp, _ := os.Open(path)
		defer fp.Close()
		s := bufio.NewScanner(fp)
		for s.Scan() {
			i++
			content += fmt.Sprintf("  %s\n", wrap(s.Text()))
			if i > 4 {
				content += "  ...\n"
				break
			}
		}
	}
	return content
}

func (c CLI) Prompt() (File, error) {
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

func (c CLI) Restore() error {
	file, err := c.Prompt()
	if err != nil {
		return err
	}
	defer c.Inventory.Delete(file)
	_, err = os.Stat(file.From)
	if err == nil {
		file.From = file.From + "." + file.ID
	}
	return os.Rename(file.To, file.From)
}

func (c CLI) Remove(args []string) error {
	if len(args) == 0 {
		return errors.New("too few aruments")
	}

	var files []File
	groupID := xid.New().String()

	var eg errgroup.Group
	for _, arg := range args {
		_, err := os.Stat(arg)
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: no such file or directory", arg)
		}
		file, err := makeFile(groupID, arg)
		if err != nil {
			return err
		}
		files = append(files, file)
		eg.Go(func() error {
			os.MkdirAll(filepath.Dir(file.To), 0777)
			return os.Rename(file.From, file.To)
		})
	}
	defer c.Inventory.Save(files)
	return eg.Wait()
}

func (c CLI) Run(args []string) error {
	c.Inventory.Open()

	switch {
	case c.Option.Restore:
		return c.Restore()
	default:
	}

	return c.Remove(args)
}

func main() {
	var option Option

	// if making error output, ignore PrintErrors from Default
	// flags.Default&^flags.PrintErrors
	// https://godoc.org/github.com/jessevdk/go-flags#pkg-constants
	parser := flags.NewParser(&option, flags.HelpFlag|flags.PrintErrors|flags.PassDoubleDash)
	args, err := parser.Parse()
	if err != nil {
		os.Exit(2)
	}

	cli := CLI{
		Option:    option,
		Inventory: Inventory{Path: inventoryPath},
	}

	if err := cli.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}
