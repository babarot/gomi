package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/k0kubun/pp"
	"github.com/rs/xid"
)

const gomiDir = ".gomi"

type Option struct {
	Interactive bool `short:"i" description:"emulate -i option of rm command"`
	Recursive   bool `short:"r" description:"emulate -r option of rm command"`
	Force       bool `short:"f" description:"emulate -f option of rm command"`
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
	return nil
}

func (i *Inventory) Save(files []File) error {
	i.Files = files
	return nil
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
		var i int
		s := bufio.NewScanner(fp)
		for s.Scan() {
			i++
			content += fmt.Sprintf("  %s\n", s.Text())
			if i > 4 {
				content += "  ..."
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

func (c CLI) Run(args []string) error {
	if c.Option.Force {
		fmt.Println("force")
	}
	if c.Option.Recursive {
		fmt.Println("recursive")
	}

	id := xid.New().String()

	var files []File
	for _, arg := range args {
		file, err := c.Tag(GroupID(id), arg)
		if err != nil {
			return err
		}
		files = append(files, file)
		os.MkdirAll(filepath.Dir(file.To), 0777)
		if err := os.Rename(file.From, file.To); err != nil {
			return err
		}
	}

	pp.Println(files)
	return nil
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
