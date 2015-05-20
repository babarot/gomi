package gomi

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/b4b4r07/ctime"
	"github.com/nsf/termbox-go"
)

func quickLook(lines Lines) error {
	ctx.quicklook = true
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	var details []string

	if info, err := os.Stat(lines.trashcan); err != nil {
		return err
	} else {
		if info.IsDir() {
			err := filepath.Walk(lines.trashcan,
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

					rel, err := filepath.Rel(lines.trashcan, path)
					w := width/2 - len(rel)
					details = append(details, fmt.Sprintf("%s %v %s %s",
						rel+strings.Repeat(" ", w),
						//mtime.Format("2006/01/02 15:04:05"),
						ct.Format("2006/01/02 15:04:05"),
						info.Mode(),
						calcSize(info.Size()),
					))
					return nil
				})
			if err != nil {
				return err
			}
		} else {
			f, err := os.Open(lines.trashcan)
			if err != nil {
				return err
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				details = append(details, scanner.Text())
			}
		}
	}

	fgAttr := termbox.ColorDefault
	bgAttr := termbox.ColorDefault

	info := []string{
		strings.Repeat("=", width),
		fmt.Sprintf("# File Name:     %s", strings.TrimSuffix(filepath.Base(lines.trashcan), filepath.Ext(filepath.Base(lines.trashcan)))),
		fmt.Sprintf("# Deleted time:  %s", lines.datetime),
		fmt.Sprintf("# Restore dest:  %s", filepath.Dir(lines.location)),
		fmt.Sprintf("# Repository:    %s", lines.trashcan),
		fmt.Sprintf("# Quick Help:"),
		fmt.Sprintf("#   %s", "<C-n> Next, <C-p> Previous, <C-q><Esc><C-c> Quit, <Enter> Restore"),
		strings.Repeat("=", width),
	}

	for i, e := range info {
		uprintTB(0, i, termbox.ColorRed, bgAttr, e)
	}

	for i, e := range details {
		uprintTB(0, len(info)+i, fgAttr, bgAttr, e)
		if i == height-1 {
			break
		}
	}

	uprintTB(0, height-1, termbox.ColorRed, bgAttr, strings.Repeat("=", width))

	return termbox.Flush()
}

func uprintTB(x, y int, fg, bg termbox.Attribute, msg string) {
	for len(msg) > 0 {
		c, w := utf8.DecodeRuneInString(msg)
		msg = msg[w:]
		termbox.SetCell(x, y, c, fg, bg)
		x += w
	}
}
