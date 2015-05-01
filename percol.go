package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"
	"unicode/utf8"

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

func restore() error {
	if d := percol(); d != "" {
		e := strings.Split(d, " ")

		if _, err := os.Stat(e[2]); err == nil {
			//log.Fatal("already exists")
			return err
		}
		if err := os.Rename(e[3], e[2]); err != nil {
			//log.Fatal(err)
			return err
		}

		deleteFromLog()
	}
	return nil
}

func percol() string {
	var err error

	//defer func() {
	//	if ctx.result != "" {
	//		os.Stdout.WriteString(ctx.result)
	//	}
	//}()

	//var input *os.File

	//// receive input from either a file or Stdin
	//input, err = os.Open(os.Getenv("HOME") + "/.rmtrash/log")
	//if err != nil {
	//	os.Exit(1)
	//}

	//rdr := bufio.NewReader(input)
	//for {
	//	line, err := rdr.ReadString('\n')
	//	if err != nil {
	//		break
	//	}

	//	ctx.lines = append(ctx.lines, Match{line, nil})
	//}
	lines := fileToArray(os.Getenv("HOME") + "/.rmtrash/log")
	//for _, line := range reverseArray(lines) {
	for _, line := range trimRecord(reverseArray(lines), " ", 0, 3) {
		ctx.trimedLines = append(ctx.trimedLines, Match{line, nil})
	}
	for _, line := range reverseArray(lines) {
		ctx.lines = append(ctx.lines, Match{line, nil})
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
	//for _, line := range ctx.lines {
	for _, line := range ctx.lines {
		ms := re.FindAllStringSubmatchIndex(line.line, 1)
		if ms == nil {
			continue
		}
		ctx.current = append(ctx.current, Match{line.line, ms})
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
		//targets = ctx.current
		for _, t := range ctx.current {
			tmp := strings.Split(t.line, " ")
			t.line = strings.Join(tmp[0:3], " ")
			targets = append(targets, Match{t.line, t.matches})
		}
	}

	printTB(0, 0, termbox.ColorDefault, termbox.ColorDefault, "QUERY>")
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
	case termbox.KeyEsc:
		termbox.Close()
		os.Exit(1)
		/*
			case termbox.KeyHome, termbox.KeyCtrlA:
				cursor_x = 0
			case termbox.KeyEnd, termbox.KeyCtrlE:
				cursor_x = len(input)
		*/
	case termbox.KeyCtrlC:
		ctx.loop = false
	case termbox.KeyEnter:
		//if len(ctx.current) == 1 {
		//	ctx.result = ctx.current[0].line
		//} else if ctx.selectedLine > 0 && ctx.selectedLine < len(ctx.current) {
		//	ctx.result = ctx.current[ctx.selectedLine].line
		//}

		//if ctx.selectedLine < len(ctx.lines) && ctx.selectedLine > len(ctx.current) {
		//	ctx.result = ctx.lines[ctx.selectedLine-1].line
		//} else if ctx.selectedLine < len(ctx.lines) && ctx.selectedLine < len(ctx.current) {
		//	ctx.result = ctx.current[ctx.selectedLine-1].line
		//}

		//if ctx.selectedLine < len(ctx.lines) && ctx.selectedLine > len(ctx.current) {
		//	ctx.result = ctx.lines[ctx.selectedLine-1].line
		//} else if ctx.selectedLine < len(ctx.lines) && ctx.selectedLine < len(ctx.current) {
		//	ctx.result = ctx.current[ctx.selectedLine-1].line
		//} else {
		//	ctx.result = ctx.lines[ctx.selectedLine-1].line
		//}

		//if ctx.selectedLine < len(ctx.lines) && ctx.selectedLine < len(ctx.current) {
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
		// filer when if ctx.selectedLine < len(ctx.current) {
		// normal whn if ctx.selectedLine < len(ctx.lines) {

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
		fmt.Fprintf(os.Stderr, "File %s could not read: %v\n", filePath, err)
		os.Exit(1)
	}

	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if serr := scanner.Err(); serr != nil {
		fmt.Fprintf(os.Stderr, "File %s scan error: %v\n", filePath, err)
	}

	return lines
}

func trimRecord(data []string, delimiter string, start, end int) []string {
	var ret, tmp []string
	for _, line := range data {
		tmp = strings.Split(line, delimiter)
		line = strings.Join(tmp[start:end], delimiter)
		ret = append(ret, line)
	}
	return ret
}

func deleteFromLog() {
	var logline []string

	for _, line := range fileToArray(os.Getenv("HOME") + "/.rmtrash/log") {
		if line == ctx.result {
			continue
		}
		logline = append(logline, line)
	}

	// delete ctx.result from log
	func(lines []string, path string) error {
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
	}(logline, os.Getenv("HOME")+"/.rmtrash/log")

}
