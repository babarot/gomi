package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"
	"unicode/utf8"

	"log"
	"path/filepath"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

type Ctx struct {
	result       string
	loop         bool
	mutex        sync.Mutex
	query        []rune
	dirty        bool // true if filtering must be redone
	cursorX      int
	selectedLine int
	lines        []Match
	trimedLines  []Match
	current      []Match
}

type Match struct {
	line    string
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
}

var timer *time.Timer

func restore(path string) error {
	if d := percol(); d != "" {
		e := strings.Split(d, " ")
		src := e[3]
		dest := e[2]

		// gomi -r arg
		if path != "" {
			// case:
			// given `gomi -r dir/file` as arguments
			// --> if dir dose not exist
			if _, err := os.Stat(filepath.Dir(path)); err != nil {
				if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
					log.Fatal(err)
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
			if askForConfirmation() {
				return nil
			} else {
				err = fmt.Errorf("%s: already exists", dest)
				return err
			}
		}
		if err := os.Rename(src, dest); err != nil {
			return err
		}

		deleteFromLog()
	}

	return nil
}

func percol() string {
	var err error

	lines := fileToArray(rm_log)
	for _, line := range reverseArray(lines) {
		ctx.lines = append(ctx.lines, Match{line, nil})
	}
	for _, line := range reverseArray(lines) {
		s := strings.Split(line, " ")
		s2 := strings.Join(s[0:3], " ")
		ctx.trimedLines = append(ctx.trimedLines, Match{s2, nil})
	}

	err = termbox.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc)
	refreshScreen(0)
	mainLoop()

	for _, line := range lines {
		if strings.Contains(line, ctx.result) {
			ctx.result = line
			break
		}
	}

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
		linelines := strings.Split(line.line, " ")
		lineline := strings.Join(linelines[0:3], " ")
		ms := re.FindAllStringSubmatchIndex(lineline, 1)
		if ms == nil {
			continue
		}
		ctx.current = append(ctx.current, Match{lineline, ms})
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
			drawScreen()
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
		//targets = ctx.lines
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
			//fgAttr = termbox.AttrUnderline
			fgAttr = termbox.ColorBlack
			bgAttr = termbox.ColorBlue
		}

		target := targets[n-1]
		line := target.line
		if target.matches == nil {
			printTB(0, n, fgAttr, bgAttr, line)
		} else {
			prev := 0
			for _, m := range target.matches {
				if m[0] > prev {
					printTB(prev, n, fgAttr, bgAttr, line[prev:m[0]])
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
			}
		}
	}
	termbox.Flush()
}

func mainLoop() {
	for ctx.loop {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventError {
			//update = false
		} else if ev.Type == termbox.EventKey {
			handleKeyEvent(ev)
		}
	}
}

func handleKeyEvent(ev termbox.Event) {
	update := true
	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		termbox.Close()
		os.Exit(1)
		/*
			case termbox.KeyHome, termbox.KeyCtrlA:
				cursor_x = 0
			case termbox.KeyEnd, termbox.KeyCtrlE:
				cursor_x = len(input)
		*/
	case termbox.KeyEnter:
		if ctx.selectedLine <= len(ctx.current) {
			ctx.result = ctx.current[ctx.selectedLine-1].line
		} else {
			ctx.result = ctx.lines[ctx.selectedLine-1].line
		}
		ctx.loop = false
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
		//fmt.Fprintf(os.Stderr, "File %s could not read: %v\n", filePath, err)
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

func deleteFromLog() {
	var logline []string

	for _, line := range fileToArray(rm_log) {
		if line == ctx.result {
			continue
		}
		logline = append(logline, line)
	}

	// delete ctx.result from log
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
	}(logline, rm_log); err != nil {
		log.Fatal(err)
	}
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
		log.Fatal(err)
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
