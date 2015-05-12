package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

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
func checkPath(cmd string) (ret string, err error) {
	ret, err = exec.LookPath(cmd)
	if err != nil {
		err = fmt.Errorf("%s: executable file not found in $PATH", cmd)
		return
	}

	return ret, nil
}
