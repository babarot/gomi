package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"
	"github.com/manifoldco/promptui"
	"github.com/rs/xid"
	"golang.org/x/sync/errgroup"
)

const gomiDir = ".gomi"

type Option struct {
	Restore  bool     `long:"restore" description:"Restore deleted file"`
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

type GroupID string

type File struct {
	Name      string    `json:"name"`   // file.go
	ID        string    `json:"id"`     // asfasfafd
	GID       GroupID   `json:"gid"`    // zoapompji
	IsDir     bool      `json:"is_dir"` // false
	From      string    `json:"from"`   // $PWD/file.go
	To        string    `json:"to"`     // ~/.gomi/2020/01/16/zoapompji/file.go.asfasfafd
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"` // 5 lines from head
}

type CLI struct {
	Option         Option
	Stdout, Stderr io.Writer
	Inventory      Inventory
}

func (i *Inventory) Open(path string) error {
	i.Path = path
	f, err := os.Open(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&i)
}

func (i *Inventory) Save(files []File) error {
	f, err := os.Create(i.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = files
	return json.NewEncoder(f).Encode(&i)
}

func (i *Inventory) Remove(remove File) error {
	var files []File
	for _, file := range i.Files {
		if file.ID == remove.ID {
			continue
		}
		files = append(files, file)
	}
	return i.Save(files)
}

func (c CLI) Tag(gid GroupID, arg string) (File, error) {
	id := xid.New().String()
	name := filepath.Base(arg)
	from, err := filepath.Abs(arg)
	if err != nil {
		return File{}, err
	}
	now := time.Now()
	fi, err := os.Stat(arg)
	if err != nil {
		return File{}, err
	}
	fp, err := os.Open(arg)
	if err != nil {
		return File{}, err
	}
	defer fp.Close()
	var content string
	if !fi.IsDir() {
		content += "\n"
		var i int
		s := bufio.NewScanner(fp)
		for s.Scan() {
			i++
			content += fmt.Sprintf("  %s\n", s.Text())
			if i > 4 {
				content += "  ...\n"
				break
			}
		}
		if err := s.Err(); err != nil {
			return File{}, err
		}
	}
	return File{
		Name:  name,
		ID:    id,
		GID:   gid,
		IsDir: fi.IsDir(),
		From:  from,
		To: filepath.Join(
			os.Getenv("HOME"), gomiDir,
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
			string(gid), fmt.Sprintf("%s.%s", name, id),
		),
		Timestamp: now,
		Content:   content,
	}, nil
}

func (c CLI) Prompt(files []File) (File, error) {
	funcMap := promptui.FuncMap
	funcMap["time"] = humanize.Time
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   promptui.IconSelect + " {{ .Name | cyan }}",
		Inactive: "  {{ .Name | faint }}",
		Selected: promptui.IconGood + " {{ .Name }}",
		Details: `
{{ "ID:" | faint }}	{{ .ID }}
{{ "Path:" | faint }}	{{ .From }}
{{ "IsDir:" | faint }}	{{ not .IsDir }}
{{ "DeletedAt:" | faint }}	{{ .Timestamp | time }}
{{ "Content:" | faint }}	{{ .Content }}
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
		Label:             "Select a page",
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
	files := c.Inventory.Files
	if len(files) == 0 {
		return errors.New("no deleted files found")
	}
	file, err := c.Prompt(files)
	if err != nil {
		return err
	}
	defer c.Inventory.Remove(file)
	_, err = os.Stat(file.From)
	if err == nil {
		file.From = file.From + "." + file.ID
	}
	return os.Rename(file.To, file.From)
}

func (c CLI) Remove(args []string) error {
	var files []File
	id := xid.New().String()

	var eg errgroup.Group
	for _, arg := range args {
		file, err := c.Tag(GroupID(id), arg)
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
	f := filepath.Join(os.Getenv("HOME"), gomiDir, "inventory.json")
	c.Inventory.Open(f)

	switch {
	case c.Option.Restore:
		return c.Restore()
	default:
	}

	if len(args) == 0 {
		return errors.New("too few aruments")
	}

	return c.Remove(args)
}

func main() {
	var opt Option

	// if making error output, ignore PrintErrors from Default
	// flags.Default&^flags.PrintErrors
	// https://godoc.org/github.com/jessevdk/go-flags#pkg-constants
	parser := flags.NewParser(&opt, flags.HelpFlag|flags.PrintErrors|flags.PassDoubleDash)
	args, err := parser.Parse()
	if err != nil {
		os.Exit(2)
	}

	cli := CLI{
		Option: opt,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cli.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}
