package main

import (
	"bufio"
	"fmt"
	"github.com/nsf/termbox-go"
	"os"
	"path/filepath"
	"strings"
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

	// Check if rm_log contains selected line string
	//log_lines := reverseArray(fileToArray(rm_log))
	//for _, line := range log_lines {
	//	if strings.Contains(line, selected) {
	//		selected = line
	//		break
	//	}
	//}
	selected = logLineSearcher(selected)

	// Get gomi-ed file name
	splited_line := logLineSplitter(filepath.Join(selected))
	file := splited_line[2]
	attr := ""
	var lines []string

	if info, err := os.Stat(file); err != nil {
		panic(err)
	} else {
		if info.IsDir() {
			attr = "directory"
			err := filepath.Walk(file,
				func(path string, info os.FileInfo, err error) error {
					if info.IsDir() && filepath.HasPrefix(info.Name(), ".") {
						return filepath.SkipDir
					}

					rel, err := filepath.Rel(file, path)
					w := width/2 - len(rel)
					//c, _ := info.Sys().(*syscall.Stat_t).Ctimespec.Unix()
					//lines = append(lines, fmt.Sprintf("%s %s %s %d\n", rel+strings.Repeat(" ", w), time.Unix(c, 0).Format("2006-01-02 15:04:05"), info.Mode(), info.Size()))
					lines = append(lines, fmt.Sprintf("%s %s %d\n", rel+strings.Repeat(" ", w), info.Mode(), info.Size()))
					return nil
				})
			if err != nil {
				panic(err)
			}
		} else {
			attr = "file"
			f, err := os.Open(file)
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

	info := []string{
		strings.Repeat("=", width),
		fmt.Sprintf(" filename:    %s (%s)\n", filepath.Base(splited_line[2]), attr),
		fmt.Sprintf(" delete-date: %s\n", splited_line[0]),
		fmt.Sprintf(" dest:        %s\n", filepath.Dir(splited_line[1])),
		fmt.Sprintf(" store-dest:  %s\n", splited_line[2]),
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
