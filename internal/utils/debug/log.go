package debug

import (
	"fmt"
	"io"
	"os"

	"github.com/babarot/gomi/internal/utils/env"
	"github.com/mattn/go-isatty"
	"github.com/nxadm/tail"
)

func Logs(w io.Writer, live bool) error {
	shouldFollow := isatty.IsTerminal(os.Stdout.Fd())
	tailConfig := tail.Config{
		ReOpen: shouldFollow,
		Follow: shouldFollow,
		Poll:   true,
		Logger: tail.DiscardingLogger,
	}
	if live {
		tailConfig.Location = &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd,
		}
	}
	t, err := tail.TailFile(env.GOMI_LOG_PATH, tailConfig)
	if err != nil {
		return err
	}
	for line := range t.Lines {
		fmt.Fprintln(w, line.Text)
	}
	return nil
}
