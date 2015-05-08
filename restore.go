package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"
	"unicode/utf8"

	"path/filepath"
	"runtime"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

type Ctx struct {
	result       string
	loop         bool
	mutex        sync.Mutex
	query        []rune
	dirty        bool
	cursorX      int
	selectedLine int
	lines        []Match
	trimedLines  []Match
	current      []Match
	quicklook    bool
}

type Match struct {
	line    string
	isdir   bool
	matches [][]int
}

var ctx = Ctx{
	"",
	true,
	sync.Mutex{},
	[]rune{},
	false,
	0,
	1,
	[]Match{},
	[]Match{},
	nil,
	false,
}

var timer *time.Timer

func cleanLog() error {
	var array []string
	for _, line := range fileToArray(rm_log) {
		s := logLineSplitter(filepath.Join(line))
		if len(s) < 2 {
			continue
		}
		if _, err := os.Stat(s[2]); err == nil {
			array = append(array, line)
		}
	}

	if err := func(lines []string, path string) error {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()

		w := bufio.NewWriter(file)
		for _, line := range lines {
			fmt.Fprintln(w, line)
		}
		return w.Flush()
	}(array, rm_log); err != nil {
		return err
	}

	return nil
}

func restore(path string) error {
	if err := cleanLog(); err != nil {
		return err
	}

	if d := pecoInterface(); d != "" && !ctx.quicklook {
		e := logLineSplitter(filepath.Join(d))
		src := e[2]
		dest := e[1]

		// gomi -r arg
		if path != "" {
			// case:
			// given `gomi -r dir/file` as arguments
			// --> if dir dose not exist
			if _, err := os.Stat(filepath.Dir(path)); err != nil {
				if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
					return err
				}
			}

			dest = path
			if info, err := os.Stat(path); err == nil {
				if info.IsDir() {
					dest = path + "/" + filepath.Base(src)
				}
			}
		}

		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("WARNING: %s overwrite? (y/N): ", dest)
			if !askForConfirmation() {
				err = fmt.Errorf("%s: already exists", dest)
				return err
			}
		}
		if err := os.Rename(src, dest); err != nil {
			return err
		}

		if err := cleanLog(); err != nil {
			return err
		}
	}

	return nil
}

func pecoInterface() string {
	var err error

	// Make ctx.lines
	lines := fileToArray(rm_log)
	for _, line := range reverseArray(lines) {
		isdir := false
		s := logLineSplitter(filepath.Join(line))
		if info, err := os.Stat(s[2]); err == nil && info.IsDir() {
			isdir = true
		}
		ctx.lines = append(ctx.lines, Match{line, isdir, nil})
	}

	// Make ctx.trimedLines
	for _, line := range reverseArray(lines) {
		s := logLineSplitter(filepath.Join(line))
		s2 := strings.Join(s[0:2], " ")
		isdir := false
		if info, err := os.Stat(s[2]); err == nil && info.IsDir() {
			isdir = true
		}
		ctx.trimedLines = append(ctx.trimedLines, Match{s2, isdir, nil})
	}

	// Termbox init
	err = termbox.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc)
	refreshScreen(0)
	mainLoop()

	ctx.result = logLineSearcher(ctx.result)
	return ctx.result
}

func printTB(x, y int, fg, bg termbox.Attribute, msg string) {
	for len(msg) > 0 {
		c, w := utf8.DecodeRuneInString(msg)
		msg = msg[w:]
		termbox.SetCell(x, y, c, fg, bg)
		x += w
	}
}

func filterLines() {
	// reset selected line
	ctx.selectedLine = 1

	ctx.current = []Match{}

	var str string
	switch string(ctx.query) {
	case "today":
		str = time.Now().Format("2006-01-02")
	default:
		str = string(ctx.query)
	}

	re := regexp.MustCompile(regexp.QuoteMeta(str))
	for _, line := range ctx.lines {
		linelines := logLineSplitter(filepath.Join(line.line))
		lineline := strings.Join(linelines[0:2], " ")
		ms := re.FindAllStringSubmatchIndex(lineline, 1)
		if ms == nil {
			continue
		}
		isdir := false
		if info, err := os.Stat(linelines[2]); err == nil && info.IsDir() {
			isdir = true
		}
		ctx.current = append(ctx.current, Match{lineline, isdir, ms})
	}
	if len(ctx.current) == 0 {
		ctx.current = nil
	}
}

func refreshScreen(delay time.Duration) {
	if timer == nil {
		timer = time.AfterFunc(delay, func() {
			if ctx.dirty {
				filterLines()
			}
			if ctx.quicklook {
				quickLook()
			} else {
				drawScreen()
			}
			ctx.dirty = false
		})
	} else {
		timer.Reset(delay)
	}
}

func drawScreen() {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()

	width, height := termbox.Size()
	_ = width
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	var targets []Match
	if ctx.current == nil {
		targets = ctx.trimedLines
	} else {
		targets = ctx.current
	}

	printTB(0, 0, termbox.ColorDefault, termbox.ColorDefault, "SEARCH>")
	printTB(8, 0, termbox.ColorDefault, termbox.ColorDefault, string(ctx.query))
	for n := 1; n+2 < height; n++ {
		if n-1 >= len(targets) {
			break
		}

		fgAttr := termbox.ColorDefault
		bgAttr := termbox.ColorDefault
		if n == ctx.selectedLine {
			//fgAttr = termbox.ColorBlack
			//bgAttr = termbox.ColorWhite
			fgAttr = termbox.AttrUnderline
		}

		target := targets[n-1]
		line := target.line
		if target.matches == nil {
			printTB(0, n, fgAttr, bgAttr, line)
			if target.isdir {
				l := logLineSplitter(filepath.Join(logLineSearcher(line)))
				printTB(20, n, fgAttr|termbox.ColorBlue, bgAttr, l[1])
			}

		} else {
			prev := 0
			for _, m := range target.matches {
				if m[0] > prev {
					printTB(prev, n, fgAttr, bgAttr, line[prev:m[0]])
					if target.isdir {
						l := logLineSplitter(filepath.Join(logLineSearcher(line[prev:m[0]])))
						printTB(20, n, fgAttr|termbox.ColorBlue, bgAttr, l[1])
					}
					prev += runewidth.StringWidth(line[prev:m[0]])
				}
				printTB(prev, n, fgAttr|termbox.ColorGreen, bgAttr, line[m[0]:m[1]])
				prev += runewidth.StringWidth(line[m[0]:m[1]])
			}

			m := target.matches[len(target.matches)-1]
			if m[0] > prev {
				printTB(prev, n, fgAttr|termbox.ColorGreen, bgAttr, line[m[0]:m[1]])
			} else if len(line) > m[1] {
				printTB(prev, n, fgAttr, bgAttr, line[m[1]:len(line)])
				if target.isdir {
					printTB(prev, n, fgAttr|termbox.ColorBlue, bgAttr, line[m[1]:len(line)])
				}
				l := logLineSplitter(filepath.Join(logLineSearcher(target.line)))
				printTB(0, n, fgAttr, bgAttr, l[0])
			}
		}
	}
	termbox.Flush()
}

func mainLoop() {
	for ctx.loop {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventError {
		} else if ev.Type == termbox.EventKey {
			handleKeyEvent(ev)
		}
	}
}

func handleKeyEvent(ev termbox.Event) {
	update := true
	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		if ctx.quicklook {
			ctx.quicklook = false
		} else {
			termbox.Close()
			os.Exit(1)
		}
	case termbox.KeyEnter:
		ctx.quicklook = false
		if ctx.selectedLine <= len(ctx.current) {
			ctx.result = ctx.current[ctx.selectedLine-1].line
		} else {
			ctx.result = ctx.lines[ctx.selectedLine-1].line
		}
		ctx.loop = false
	case termbox.KeyCtrlQ:
		if ctx.quicklook {
			ctx.quicklook = false
		} else {
			ctx.dirty = false
			ctx.quicklook = true
		}
	case termbox.KeyArrowUp, termbox.KeyCtrlP:
		if 1 < ctx.selectedLine {
			ctx.selectedLine--
		}
	case termbox.KeyArrowDown, termbox.KeyCtrlN:
		if ctx.selectedLine < len(ctx.lines) && ctx.selectedLine > len(ctx.current) {
			if ctx.selectedLine < len(ctx.lines) {
				ctx.selectedLine++
			}
		} else if ctx.selectedLine < len(ctx.lines) && ctx.selectedLine < len(ctx.current) {
			if ctx.selectedLine < len(ctx.current) {
				ctx.selectedLine++
			}
		}
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		if len(ctx.query) == 0 {
			update = false
		} else {
			ctx.query = ctx.query[:len(ctx.query)-1]
			ctx.dirty = true
		}
	default:
		if ev.Key == termbox.KeySpace {
			ev.Ch = ' '
		}

		if ev.Ch > 0 {
			ctx.query = append(ctx.query, ev.Ch)
			ctx.dirty = true
		}
	}

	if update {
		refreshScreen(10 * time.Millisecond)
	}
}

func reverseArray(input []string) []string {
	if len(input) == 0 {
		return input
	}
	return append(reverseArray(input[1:]), input[0])
}

func fileToArray(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if serr := scanner.Err(); serr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	if len(lines) == 0 {
		fmt.Fprintf(os.Stderr, "No content in %s\n", filePath)
		os.Exit(1)
	}

	return lines
}

// askForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling askForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)")
func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		panic(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
}

// You might want to put the following two functions in a separate utility package.
// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

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

// Split line
// 2006-01-02 15:04:05 /Users/b4b4r07/work/README.md /Users/b4b4r07/.gomi/2006/01/02/README.md.15_04_05
// -->
// 0. 2006-01-02 15:04:05
// 1. /Users/b4b4r07/work/README.md
// 2. /Users/b4b4r07/.gomi/2006/01/02/README.md.15_04_05
func logLineSplitter(line string) []string {
	str := []byte(line)
	var assigned *regexp.Regexp
	if runtime.GOOS == "windows" {
		assigned = regexp.MustCompile(`(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d) (C:.*) (C:.*)`)
	} else {
		assigned = regexp.MustCompile(`(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d) (/.*) (/.*)`)
	}
	group := assigned.FindSubmatch(str)

	var ret []string
	for i := 1; i < len(group); i++ {
		ret = append(ret, string(group[i]))
	}

	return ret
}

// Search line from rm_log
// 2006-01-02 15:04:05 /Users/b4b4r07/work/README.md
// -->
// 2006-01-02 15:04:05 /Users/b4b4r07/work/README.md /Users/b4b4r07/.gomi/2006/01/02/README.md.15_04_05
func logLineSearcher(line string) (ret string) {
	log_lines := reverseArray(fileToArray(rm_log))
	for _, logline := range log_lines {
		if strings.Contains(logline, line) {
			ret = logline
			break
		}
	}
	return
}
