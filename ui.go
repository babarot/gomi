package gomi

import (
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"
	"unicode/utf8"

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

func pecoInterface() (ret string) {
	var err error

	// Make ctx.lines
	lines := fileToArray(rm_log)
	for _, line := range reverseArray(lines) {
		isdir := false
		_, _, trashcan, _ := logLineSplitter(line)
		if info, err := os.Stat(trashcan); err == nil && info.IsDir() {
			isdir = true
		}
		ctx.lines = append(ctx.lines, Match{line, isdir, nil})
	}

	// Make ctx.trimedLines
	for _, line := range reverseArray(lines) {
		isdir := false
		datetime, location, trashcan, _ := logLineSplitter(line)
		line = fmt.Sprintf("%s %s", datetime, location)
		if info, err := os.Stat(trashcan); err == nil && info.IsDir() {
			isdir = true
		}
		ctx.trimedLines = append(ctx.trimedLines, Match{line, isdir, nil})
	}

	// Termbox init
	err = termbox.Init()
	if err != nil {
		return
	}
	if isTty() {
		termbox.SetInputMode(termbox.InputEsc)
	}

	defer termbox.Close()

	// Main
	termbox.SetInputMode(termbox.InputEsc)
	refreshScreen(0)
	mainLoop()

	ctx.result = logLineSearcher(ctx.result)
	ret = ctx.result

	return
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
	// Reset selected line
	// --> Go to top of line (peco interface)
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
		datetime, location, trashcan, _ := logLineSplitter(line.line)
		lineline := fmt.Sprintf("%s %s", datetime, location)
		ms := re.FindAllStringSubmatchIndex(lineline, 1)
		if ms == nil {
			continue
		}
		isdir := false
		if info, err := os.Stat(trashcan); err == nil && info.IsDir() {
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
	printTB(8+2, 0, termbox.ColorDefault, termbox.ColorDefault, string(ctx.query))
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
			printTB(0+2, n, fgAttr, bgAttr, line)
			if target.isdir {
				_, location, _, _ := logLineSplitter(logLineSearcher(line))
				printTB(20+2, n, fgAttr|termbox.ColorBlue, bgAttr, location)
			}

		} else {
			prev := 0
			for _, m := range target.matches {
				if m[0] > prev {
					printTB(prev+2, n, fgAttr, bgAttr, line[prev:m[0]])
					if target.isdir {
						_, location, _, _ := logLineSplitter(logLineSearcher(line[prev:m[0]]))
						printTB(20+2, n, fgAttr|termbox.ColorBlue, bgAttr, location)
					}
					prev += runewidth.StringWidth(line[prev:m[0]])
				}
				printTB(prev+2, n, fgAttr|termbox.ColorGreen, bgAttr, line[m[0]:m[1]])
				prev += runewidth.StringWidth(line[m[0]:m[1]])
			}

			m := target.matches[len(target.matches)-1]
			if m[0] > prev {
				printTB(prev+2, n, fgAttr|termbox.ColorGreen, bgAttr, line[m[0]:m[1]])
			} else if len(line) > m[1] {
				printTB(prev+2, n, fgAttr, bgAttr, line[m[1]:len(line)])
				if target.isdir {
					printTB(prev+2, n, fgAttr|termbox.ColorBlue, bgAttr, line[m[1]:len(line)])
				}
				datetime, _, _, _ := logLineSplitter(logLineSearcher(target.line))
				printTB(0+2, n, fgAttr, bgAttr, datetime)
			}
		}
		printTB(0, ctx.selectedLine, termbox.ColorRed, termbox.ColorDefault, "> ")
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
	defer func() {
		recover()
	}()

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
