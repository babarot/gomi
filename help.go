package gomi

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nsf/termbox-go"
)

func HelpKeymap() error {
	ctx.help = true
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	var details []string
	f, err := os.Open("/Users/b4b4r07/.zshrc")
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		details = append(details, scanner.Text())
	}

	fgAttr := termbox.ColorDefault
	bgAttr := termbox.ColorDefault
	_ = fgAttr

	info := []string{
		strings.Repeat("=", width),
		//fmt.Sprintf("# Ctrl-b               :"),
		//fmt.Sprintf("# Ctrl-r               :"),
		//fmt.Sprintf("# Ctrl-s               :"),
		//fmt.Sprintf("# Ctrl-t               :"),
		//fmt.Sprintf("# Ctrl-v               :"),
		//fmt.Sprintf("# Ctrl-x               :"),
		//fmt.Sprintf("# Ctrl-y               :"),
		//fmt.Sprintf("# Ctrl-g               :"),
		//fmt.Sprintf("# Ctrl-h               :"),
		//fmt.Sprintf("# Ctrl-m               :"),
		//fmt.Sprintf("# Ctrl-o               :"),
		fmt.Sprintf("                                                                iiii   "),
		fmt.Sprintf("                                                               i::::i  "),
		fmt.Sprintf("                                                                iiii   "),
		fmt.Sprintf(""),
		fmt.Sprintf("    ggggggggg   ggggg   ooooooooooo      mmmmmmm    mmmmmmm   iiiiiii  "),
		fmt.Sprintf("   g:::::::::ggg::::g oo:::::::::::oo  mm:::::::m  m:::::::mm i:::::i  "),
		fmt.Sprintf("  g:::::::::::::::::go:::::::::::::::om::::::::::mm::::::::::m i::::i  "),
		fmt.Sprintf(" g::::::ggggg::::::ggo:::::ooooo:::::om::::::::::::::::::::::m i::::i  "),
		fmt.Sprintf(" g:::::g     g:::::g o::::o     o::::om:::::mmm::::::mmm:::::m i::::i  "),
		fmt.Sprintf(" g:::::g     g:::::g o::::o     o::::om::::m   m::::m   m::::m i::::i  "),
		fmt.Sprintf(" g:::::g     g:::::g o::::o     o::::om::::m   m::::m   m::::m i::::i  "),
		fmt.Sprintf(" g::::::g    g:::::g o::::o     o::::om::::m   m::::m   m::::m i::::i  "),
		fmt.Sprintf(" g:::::::ggggg:::::g o:::::ooooo:::::om::::m   m::::m   m::::mi::::::i "),
		fmt.Sprintf("  g::::::::::::::::g o:::::::::::::::om::::m   m::::m   m::::mi::::::i "),
		fmt.Sprintf("   gg::::::::::::::g  oo:::::::::::oo m::::m   m::::m   m::::mi::::::i "),
		fmt.Sprintf("     gggggggg::::::g    ooooooooooo   mmmmmm   mmmmmm   mmmmmmiiiiiiii "),
		fmt.Sprintf("             g:::::g"),
		fmt.Sprintf(" gggggg      g:::::g"),
		fmt.Sprintf(" g:::::gg   gg:::::g"),
		fmt.Sprintf("  g::::::ggg:::::::g"),
		fmt.Sprintf("   gg:::::::::::::g"),
		fmt.Sprintf("     ggg::::::ggg"),
		fmt.Sprintf("        gggggg"),
		fmt.Sprintf(""),
		fmt.Sprintf("# Enter                : Restore a file under the cursor or selected files"),
		fmt.Sprintf("# Ctrl-a               : Move caret to the beginning of line"),
		fmt.Sprintf("# Ctrl-c, Esc          : Exits from Restore mode or Quick Look mode with success status"),
		fmt.Sprintf("# Ctrl-_               : Remove a file under the cursor or selected files"),
		fmt.Sprintf("# Ctrl-e               : Move caret to the end of line"),
		fmt.Sprintf("# Ctrl-f, Right        : Move caret forward 1 character"),
		fmt.Sprintf("# Ctrl-i, Tab          : Toggle Help message"),
		fmt.Sprintf("# Ctrl-l               : Redraws the screen"),
		fmt.Sprintf("# Ctrl-n, Ctrl-j, Down : Moves the selected line cursor to one line below"),
		fmt.Sprintf("# Ctrl-p, Ctrl-k, Up   : Moves the selected line cursor to one line above"),
		fmt.Sprintf("# Ctrl-q               : Toggle Quick Look"),
		fmt.Sprintf("# Ctrl-u               : Delete the characters under the cursor backward until the beginning of the line"),
		fmt.Sprintf("# Ctrl-w               : Delete one word backward"),
		fmt.Sprintf("# Ctrl-v               : Select multiple lines"),
		strings.Repeat("=", width),
	}

	for i, e := range info {
		uprintTB(0, i, termbox.ColorBlue, bgAttr, e)
	}

	return termbox.Flush()
}
