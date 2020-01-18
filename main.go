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
	"github.com/k0kubun/pp"
	"github.com/manifoldco/promptui"
	"github.com/rs/xid"
	"golang.org/x/sync/errgroup"
)

const gomiDir = ".gomi"

type Option struct {
	Interactive bool `short:"i" description:"emulate -i option of rm command"`
	Recursive   bool `short:"r" description:"emulate -r option of rm command"`
	Force       bool `short:"f" description:"emulate -f option of rm command"`
	Restore     bool `long:"restore" description:"restore"`
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
}

func (i *Inventory) Open() error {
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

func (c CLI) Run(args []string) error {
	var files []File

	inv := Inventory{
		Path:  filepath.Join(os.Getenv("HOME"), gomiDir, "inventory.json"),
		Files: files,
	}
	inv.Open()

	if c.Option.Force {
		fmt.Println("force")
	}
	if c.Option.Recursive {
		fmt.Println("recursive")
	}

	if c.Option.Restore {
		if len(inv.Files) == 0 {
			return errors.New("no deleted files found")
		}
		file, err := c.Prompt(inv.Files)
		if err != nil {
			return err
		}
		pp.Println(file)
		return inv.Remove(file)
	}

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
	if err := eg.Wait(); err != nil {
		return err
	}

	return inv.Save(files)
}

func main() {
	var opt Option
	args, err := flags.ParseArgs(&opt, os.Args[1:])
	if err != nil {
		panic(err)
	}

	cli := CLI{
		Option: opt,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := cli.Run(args); err != nil {
		panic(err)
	}

	// https://golangcode.com/get-the-content-type-of-file/
}
