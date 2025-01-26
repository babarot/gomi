package debug

import (
	"fmt"
	"io"
	"os"

	"github.com/babarot/gomi/internal/env"
	"github.com/mattn/go-isatty"
	"github.com/nxadm/tail"
)

func Logs(w io.Writer) error {
	shouldFollow := isatty.IsTerminal(os.Stdout.Fd())
	t, err := tail.TailFile(env.GOMI_LOG_PATH, tail.Config{Follow: shouldFollow, ReOpen: shouldFollow})
	if err != nil {
		return err
	}
	for line := range t.Lines {
		fmt.Fprintln(w, line.Text)
	}
	return nil
}
