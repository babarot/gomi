package gomi

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func calcSize(size int64) string {
	if size == 0 {
		return "0"
	}

	f := float64(size) / 1024 / 1024
	sizeStr := fmt.Sprintf("%.2f", f)
	sizeStr = strings.TrimSuffix(sizeStr, filepath.Ext(sizeStr))
	if sizeStr != "0" {
		return fmt.Sprintf("%6.2fMB", f)
	}
	f = float64(size) / 1024
	sizeStr = fmt.Sprintf("%.2f", f)
	sizeStr = strings.TrimSuffix(sizeStr, filepath.Ext(sizeStr))
	//if sizeStr != "0" {
	return fmt.Sprintf("%6.2fKB", f)
	//}

	return strconv.Itoa(int(size))
}

func cleanLog() error {
	var array []string
	for _, line := range fileToArray(rm_log) {
		_, _, trashcan, err := logLineSplitter(line)
		if err != nil {
			continue
		}
		if _, err := os.Stat(trashcan); err == nil {
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

// Split line
// 2006-01-02 15:04:05 /Users/b4b4r07/work/README.md /Users/b4b4r07/.gomi/2006/01/02/README.md.15_04_05
// -->
// 0. 2006-01-02 15:04:05
// 1. /Users/b4b4r07/work/README.md
// 2. /Users/b4b4r07/.gomi/2006/01/02/README.md.15_04_05
func logLineSplitter(str string) (datetime, location, trashcan string, err error) {
	b := []byte(str)
	var assigned *regexp.Regexp

	if runtime.GOOS == "windows" {
		assigned = regexp.MustCompile(`(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d) (.:.*) (.:.*)`)
	} else {
		assigned = regexp.MustCompile(`(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d) (/.*) (/.*)`)
	}
	group := assigned.FindSubmatch(b)

	if len(group) < 4 {
		err = fmt.Errorf("parse error")
		return
	}

	datetime = string(group[1])
	location = filepath.Join(string(group[2]))
	trashcan = filepath.Join(string(group[3]))
	err = nil
	return
}

// Search line from rm_log
// 2006-01-02 15:04:05 /Users/b4b4r07/work/README.md
// -->
// 2006-01-02 15:04:05 /Users/b4b4r07/work/README.md /Users/b4b4r07/.gomi/2006/01/02/README.md.15_04_05
func logLineSearcher(str string) string {
	log := reverseArray(fileToArray(rm_log))
	for _, line := range log {
		if strings.Contains(line, str) {
			return line
		}
	}
	return ""
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
		panic(err)
	}

	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if serr := scanner.Err(); serr != nil {
		panic(err)
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "No content in %s", filePath)
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

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
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
}
