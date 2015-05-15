package gomi

import (
	"bufio"
	"fmt"
	"github.com/nsf/termbox-go"
	"os"
	"path/filepath"
	"strings"

	"github.com/b4b4r07/ctime"
	//"syscall"
	//"time"
)

func quickLook() {
	ctx.quicklook = true
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	width, height := termbox.Size()

	// Get selected line string
	var selected string
	if ctx.current == nil {
		selected = ctx.lines[ctx.selectedLine-1].line
	} else {
		selected = ctx.current[ctx.selectedLine-1].line
	}

	selected = logLineSearcher(selected)

	// Get gomi-ed file name
	datetime, location, trashcan, err := logLineSplitter(selected)
	if err != nil {
	}
	attr := ""
	var lines []string

	if info, err := os.Stat(trashcan); err != nil {
		panic(err)
	} else {
		if info.IsDir() {
			attr = "directory"
			err := filepath.Walk(trashcan,
				func(path string, info os.FileInfo, err error) error {
					//fi, err := os.Stat(path)
					//if err != nil {
					//	return err
					//}
					//mtime := fi.ModTime()
					ct, err := ctime.Stat(path)
					if err != nil {
						return err
					}

					if info.IsDir() && filepath.HasPrefix(info.Name(), ".") {
						return filepath.SkipDir
					}

					rel, err := filepath.Rel(trashcan, path)
					w := width/2 - len(rel)
					lines = append(lines, fmt.Sprintf("%s %v %s %s",
						rel+strings.Repeat(" ", w),
						//mtime.Format("2006/01/02 15:04:05"),
						ct.Format("2006/01/02 15:04:05"),
						info.Mode(),
						calcSize(info.Size()),
					))
					return nil
				})
			if err != nil {
				panic(err)
			}
		} else {
			attr = "file"
			f, err := os.Open(trashcan)
			if err != nil {
				panic(err)
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
		}
	}

	fgAttr := termbox.ColorDefault
	bgAttr := termbox.ColorDefault

	_ = attr
	info := []string{
		strings.Repeat("=", width),
		fmt.Sprintf("# File Name:     %s", strings.TrimSuffix(filepath.Base(trashcan), filepath.Ext(filepath.Base(trashcan)))),
		fmt.Sprintf("# Deleted time:  %s", datetime),
		fmt.Sprintf("# Restore dest:  %s", filepath.Dir(location)),
		fmt.Sprintf("# Repository:    %s", trashcan),
		fmt.Sprintf("# Quick Help:"),
		fmt.Sprintf("#   %s", "<C-n> Next, <C-p> Previous, <C-q><Esc><C-c> Quit, <Enter> Restore"),
		strings.Repeat("=", width),
	}

	for i, e := range info {
		printTB(0, i, termbox.ColorRed, bgAttr, e)
	}
	for i, e := range lines {
		printTB(0, len(info)+i, fgAttr, bgAttr, e)
		if i == height-1 {
			break
		}
	}
	printTB(0, height-1, termbox.ColorRed, bgAttr, strings.Repeat("=", width))

	termbox.Flush()
}
