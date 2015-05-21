package gomi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

var (
	duration           = 10 * time.Millisecond
	scanning           = 0
	cursor_x, cursor_y int
	width, height      int
	timer              *time.Timer
)

type Ctx struct {
	lines     []Lines
	selected  []Lines
	input     []rune
	heading   bool
	mutex     sync.Mutex
	loop      bool
	dirty     bool
	quicklook bool
	update    bool
	help      bool
}

var ctx = Ctx{
	lines:     []Lines{},
	input:     []rune{},
	heading:   false,
	mutex:     sync.Mutex{},
	loop:      true,
	dirty:     true,
	quicklook: false,
	update:    false,
	help:      false,
}

type Lines struct {
	line     string
	disp     string
	datetime string
	location string
	trashcan string
	name     string
	isdir    bool
}

type matched struct {
	Lines
	pos1     int
	pos2     int
	selected bool
}

type filtered []matched

func (f filtered) Less(i, j int) bool {
	return (f[i].pos2 - f[i].pos1) < (f[j].pos2 - f[j].pos1)
}

func (f filtered) Len() int {
	return len(f)
}

func (f filtered) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

var current filtered

func filterLine() {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()

	defer func() {
		recover()
	}()

	if len(ctx.input) == 0 {
		current = make(filtered, len(ctx.lines))
		for n, f := range ctx.lines {
			datetime, location, trashcan, _ := Split(f.line)
			prev_selected := false
			for _, s := range ctx.selected {
				if f.disp == s.disp {
					prev_selected = true
					break
				}
			}
			current[n] = matched{
				Lines: Lines{
					line:     f.line,
					disp:     fmt.Sprintf("%s %s", datetime, location),
					datetime: datetime,
					location: location,
					trashcan: trashcan,
					name:     filepath.Base(f.location),
					isdir:    f.isdir,
				},
				pos1:     -1,
				pos2:     -1,
				selected: prev_selected,
			}
		}
	} else {
		pat := "(?i)(?:.*)("
		for _, r := range []rune(ctx.input) {
			pat += regexp.QuoteMeta(string(r)) + ".*?"
		}
		pat += ")"
		re := regexp.MustCompile(pat)

		current = make(filtered, 0, len(ctx.lines))
		for _, f := range ctx.lines {
			datetime, location, trashcan, _ := Split(f.line)
			ms := re.FindAllStringSubmatchIndex(f.disp, 1)
			if len(ms) != 1 || len(ms[0]) != 4 {
				continue
			}
			prev_selected := false
			for _, s := range ctx.selected {
				if f.disp == s.disp {
					prev_selected = true
					break
				}
			}
			current = append(current, matched{
				Lines: Lines{
					line:     f.line,
					disp:     fmt.Sprintf("%s %s", datetime, location),
					datetime: datetime,
					location: location,
					trashcan: trashcan,
					name:     filepath.Base(f.location),
					isdir:    f.isdir,
				},
				pos1:     len([]rune(f.disp[0:ms[0][2]])),
				pos2:     len([]rune(f.disp[0:ms[0][3]])),
				selected: prev_selected,
			})
		}
	}

	if len(ctx.input) > 0 {
		//sort.Sort(current)
	}

	if cursor_y < 0 {
		cursor_y = 0
	}
	if cursor_y >= len(current) {
		cursor_y = len(current) - 1
	}
}

func drawScreen() {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()

	defer func() {
		recover()
	}()

	width, height = termbox.Size()
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	pat := ""
	for _, r := range ctx.input {
		pat += regexp.QuoteMeta(string(r)) + ".*?"
	}
	for n := 0; n < height-3; n++ {
		if n >= len(current) {
			break
		}
		x := 2
		w := 0
		line := current[n].line
		line = current[n].disp

		pos1 := current[n].pos1
		pos2 := current[n].pos2
		isdir := current[n].isdir
		selected := current[n].selected
		if pos1 >= 0 {
			pwidth := runewidth.StringWidth(string([]rune(current[n].line)[0:pos1]))
			if !ctx.heading && pwidth > width/2 {
				rline := []rune(line)
				wwidth := 0
				for i := 0; i < len(rline); i++ {
					w = runewidth.RuneWidth(rline[i])
					if wwidth+w > width/2 {
						line = "..." + string(rline[i:])
						pos1 -= i - 3
						pos2 -= i - 3
						break
					}
					wwidth += w
				}
			}
		}
		swidth := runewidth.StringWidth(line)
		if swidth+2 > width {
			rline := []rune(line)
			line = string(rline[0:width-5]) + "..."
		}
		for f, c := range []rune(line) {
			w = runewidth.RuneWidth(c)
			if x+w > width {
				break
			}
			if pos1 <= f && f < pos2 {
				if selected {
					//termbox.SetCell(x, height-4-n, c, termbox.ColorRed|termbox.AttrBold, termbox.ColorDefault)
					termbox.SetCell(x, height-4-n, c, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
				} else if cursor_y == n {
					termbox.SetCell(x, height-4-n, c, termbox.ColorYellow|termbox.AttrUnderline, termbox.ColorDefault)
				} else {
					termbox.SetCell(x, height-4-n, c, termbox.ColorRed|termbox.AttrBold, termbox.ColorDefault)
				}
			} else {
				if selected {
					//termbox.SetCell(x, height-4-n, c, termbox.ColorRed|termbox.AttrBold, termbox.ColorDefault)
					termbox.SetCell(x, height-4-n, c, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
				} else if cursor_y == n {
					termbox.SetCell(x, height-4-n, c, termbox.ColorYellow|termbox.AttrUnderline, termbox.ColorDefault)
				} else {
					if isdir {
						if f >= 20 {
							termbox.SetCell(x, height-4-n, c, termbox.ColorBlue, termbox.ColorDefault)
						} else {
							termbox.SetCell(x, height-4-n, c, termbox.ColorDefault, termbox.ColorDefault)
						}
					} else {
						termbox.SetCell(x, height-4-n, c, termbox.ColorDefault, termbox.ColorDefault)
					}
				}
			}
			x += w
		}
	}
	if cursor_y >= 0 {
		printTB(0, height-4-cursor_y, termbox.ColorRed|termbox.AttrBold, termbox.ColorBlack, "> ")
	}
	if scanning >= 0 {
		printTB(0, height-3, termbox.ColorGreen|termbox.AttrBold, termbox.ColorBlack, string([]rune("-\\|/")[scanning%4]))
		scanning++
	}
	printfTB(2, height-3, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "%d/%d(%d)", len(current), len(ctx.lines), len(ctx.selected))
	printTB(0, height-2, termbox.ColorBlue|termbox.AttrBold, termbox.ColorBlack, "> ")
	printTB(2, height-2, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, string(ctx.input))
	termbox.SetCursor(2+runewidth.StringWidth(string(ctx.input[0:cursor_x])), height-2)
	termbox.Flush()
}

func NewLines(line string) Lines {
	datetime, location, trashcan, _ := Split(line)
	isdir := false
	if info, err := os.Stat(trashcan); err == nil {
		if info.IsDir() {
			isdir = true
		} else {
			isdir = false
		}
	}
	lines := Lines{
		line:     line,
		disp:     fmt.Sprintf("%s %s", datetime, location),
		datetime: datetime,
		location: location,
		trashcan: trashcan,
		name:     filepath.Base(location),
		isdir:    isdir,
	}
	return lines
}

func Init() error {
	// Check rm_trash
	if info, err := os.Stat(rm_trash); err == nil {
		if info.IsDir() {
			// ok
		} else {
			return fmt.Errorf("%s: already exists, but it is not a directory", rm_trash)
		}
	} else {
		err = os.MkdirAll(rm_trash, 0777)
		if err != nil {
			return fmt.Errorf("cannot create %s", rm_trash)
		}
	}

	rm_log = filepath.Join(rm_trash, "log")

	// Check rm_log
	if info, err := os.Stat(rm_log); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s: already exists, but it is not a file", rm_log)
		} else {
			// ok
		}
	} else {
		return ioutil.WriteFile(rm_log, []byte{}, os.ModePerm)
	}

	config := &Config{}
	if err := config.ReadConfig(); err != nil {
		return err
	}
	if config.Root != "" {
		if config.Root[:2] == "~/" {
			rm_trash = strings.Replace(config.Root, "~", os.Getenv("HOME"), 1)
		} else if !strings.HasPrefix(config.Root, "/") {
			rm_trash = filepath.Join(os.Getenv("HOME"), config.Root)
		}
	}

	// Clean rm_log
	return cleanLog()
}

func Interface() (selected []Lines, err error) {
	data := reverseArray(fileToArray(rm_log))
	ctx.lines = make([]Lines, 0)
	for _, line := range data {
		ctx.lines = append(ctx.lines, NewLines(line))
	}

	err = termbox.Init()
	if err != nil {
		return
	}

	if isTty() {
		termbox.SetInputMode(termbox.InputEsc)
	}
	defer termbox.Close()

	// Termbox init
	termbox.SetInputMode(termbox.InputEsc)
	refreshScreen(0)
	mainLoop()

	selected = ctx.selected
	if len(selected) == 0 {
		err = fmt.Errorf("no selected")
		return
	}

	return
}

func handleKeyEvent(ev termbox.Event) {
	defer func() {
		recover()
	}()

	switch ev.Key {
	case termbox.KeyCtrlQ:
		if ctx.quicklook {
			ctx.quicklook = false
		} else {
			ctx.quicklook = true
		}
	case termbox.KeyTab:
		if ctx.help {
			ctx.help = false
		} else {
			ctx.help = true
		}
	case termbox.KeyEsc, termbox.KeyCtrlC:
		if ctx.quicklook && ctx.help {
			if ctx.help {
				ctx.help = false
			}
		} else if ctx.quicklook {
			ctx.quicklook = false
		} else if ctx.help {
			ctx.help = false
		} else {
			termbox.Close()
			os.Exit(1)
		}
	case termbox.KeyHome, termbox.KeyCtrlA:
		cursor_x = 0
	case termbox.KeyEnd, termbox.KeyCtrlE:
		cursor_x = len(ctx.input)
	case termbox.KeyEnter:
		ctx.quicklook = false
		if cursor_y >= 0 && cursor_y < len(current) {
			if len(ctx.selected) == 0 {
				ctx.selected = append(ctx.selected, current[cursor_y].Lines)
			}
			ctx.loop = false
		}
	case termbox.KeyArrowLeft, termbox.KeyCtrlB:
		if cursor_x > 0 {
			cursor_x--
		}
	case termbox.KeyArrowRight, termbox.KeyCtrlF:
		if cursor_x < len([]rune(ctx.input)) {
			cursor_x++
		}
	case termbox.KeyArrowUp, termbox.KeyCtrlK, termbox.KeyCtrlP:
		if cursor_y < len(current)-1 {
			if cursor_y < height-4 {
				cursor_y++
			}
		}
	case termbox.KeyArrowDown, termbox.KeyCtrlJ, termbox.KeyCtrlN:
		if cursor_y > 0 {
			cursor_y--
		}
	case termbox.KeyCtrlS:
		sort.Sort(current)
		ctx.update = true
	//case termbox.KeyCtrlI:
	//	ctx.heading = !ctx.heading
	case termbox.KeyCtrlL:
		ctx.update = true
	case termbox.KeyCtrlU:
		cursor_x = 0
		ctx.input = []rune{}
		ctx.update = true
	case termbox.KeyCtrlW:
		part := string(ctx.input[0:cursor_x])
		rest := ctx.input[cursor_x:len(ctx.input)]
		pos := regexp.MustCompile(`\s+`).FindStringIndex(part)
		if len(pos) > 0 && pos[len(pos)-1] > 0 {
			println(pos[len(pos)-1])
			ctx.input = []rune(part[0 : pos[len(pos)-1]-1])
			ctx.input = append(ctx.input, rest...)
		} else {
			ctx.input = []rune{}
		}
		cursor_x = len(ctx.input)
		ctx.update = true
	case termbox.KeyCtrlUnderscore:
		var deleted []string
		if len(ctx.selected) == 0 {
			deleted = append(deleted, current[cursor_y].Lines.trashcan)
		} else {
			for _, f := range ctx.selected {
				deleted = append(deleted, f.trashcan)
			}
		}
		for _, f := range deleted {
			os.RemoveAll(f)
		}
		ctx.lines = nil
		ctx.selected = nil
		cleanLog()
		for _, line := range reverseArray(fileToArray(rm_log)) {
			ctx.lines = append(ctx.lines, NewLines(line))
		}
		ctx.update = true
	case termbox.KeyCtrlV:
		found := -1
		line := current[cursor_y].line
		for i, s := range ctx.selected {
			if line == s.line {
				found = i
				break
			}
		}
		if found == -1 {
			ctx.selected = append(ctx.selected, current[cursor_y].Lines)
		} else {
			ctx.selected = append(ctx.selected[:found], ctx.selected[found+1:]...)
		}
		if cursor_y < len(current)-1 {
			if cursor_y < height-4 {
				cursor_y++
			}
		}
		ctx.update = true
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		if cursor_x > 0 {
			ctx.input = append(ctx.input[0:cursor_x-1], ctx.input[cursor_x:len(ctx.input)]...)
			cursor_x--
			ctx.update = true
		}
	case termbox.KeyDelete:
		if cursor_x < len([]rune(ctx.input)) {
			ctx.input = append(ctx.input[0:cursor_x], ctx.input[cursor_x+1:len(ctx.input)]...)
			ctx.update = true
		}
	default:
		if ev.Key == termbox.KeySpace {
			ev.Ch = ' '
		}
		if ev.Ch > 0 {
			out := []rune{}
			out = append(out, ctx.input[0:cursor_x]...)
			out = append(out, ev.Ch)
			ctx.input = append(out, ctx.input[cursor_x:len(ctx.input)]...)
			cursor_x++
			ctx.update = true
		}
	}

	// If need to update, start timer
	if scanning != -1 {
		if ctx.update {
			ctx.dirty = true
			timer.Reset(duration)
		} else {
			timer.Reset(1)
		}
	} else {
		if ctx.update {
			filterLine()
		} else {
			//drawScreen()
		}
		drawScreen()
	}
}

func refreshScreen(delay time.Duration) {
	if timer == nil {
		timer = time.AfterFunc(delay, func() {
			if ctx.dirty {
				filterLine()
			}
			if ctx.help {
				ctx.input = []rune{}
				termbox.HideCursor()
				if err := HelpKeymap(); err != nil {
					//ctx.help = false
					//panic(err)
				}
			} else if ctx.quicklook {
				ctx.input = []rune{}
				termbox.HideCursor()
				if err := quickLook(current[cursor_y].Lines); err != nil {
					//ctx.quicklook = false
					//panic(err)
				}
			} else {
				drawScreen()
			}
			ctx.dirty = false
		})
	} else {
		timer.Reset(delay)
	}
}

func mainLoop() {
	for ctx.loop {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventError {
			ctx.update = false
		} else if ev.Type == termbox.EventKey {
			handleKeyEvent(ev)
		}
	}
}

func printTB(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range []rune(msg) {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func printfTB(x, y int, fg, bg termbox.Attribute, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	printTB(x, y, fg, bg, s)
}
