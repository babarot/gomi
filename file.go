package main

import (
	//"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var rm_trash string = os.Getenv("HOME") + "/.rmtrash"
var rm_log string = rm_trash + "/log"

func main() {
	if d := restore(); d != "" {
		e := strings.Split(d, " ")
		fmt.Println(e[3], e[2])
	}

	//for _, gomi := range os.Args[1:] {
	//	if path, err := remove(gomi); err != nil {
	//		fmt.Println(err)
	//	} else {
	//		gomi, _ = filepath.Abs(gomi)
	//		if err := logging(gomi, path); err != nil {
	//			fmt.Println(err)
	//		}
	//	}
	//}
}

//func reverseArray(input []string) []string {
//	if len(input) == 0 {
//		return input
//	}
//	return append(reverseArray(input[1:]), input[0])
//}
//
//func fileToArray(filePath string) []string {
//	f, err := os.Open(filePath)
//	if err != nil {
//		fmt.Fprintf(os.Stderr, "File %s could not read: %v\n", filePath, err)
//		os.Exit(1)
//	}
//
//	defer f.Close()
//
//	var lines []string
//	scanner := bufio.NewScanner(f)
//	for scanner.Scan() {
//		lines = append(lines, scanner.Text())
//	}
//	if serr := scanner.Err(); serr != nil {
//		fmt.Fprintf(os.Stderr, "File %s scan error: %v\n", filePath, err)
//	}
//
//	return lines
//}

func logging(src, dest string) (err error) {
	f, err := os.OpenFile(rm_log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}

	defer f.Close()

	text := fmt.Sprintf("%s %s %s\n", time.Now().Format("2006/01/02 15:04:05"), src, dest)
	if _, err = f.WriteString(text); err != nil {
		return
	}

	return
}

func remove(src string) (dest string, err error) {
	// Check if rm_trash exists
	path, err := os.Stat(rm_trash)
	if os.IsNotExist(err) {
		return
	}
	if !path.IsDir() {
		err = errors.New("an error")
		return
	}

	// Check if src exists
	_, err = os.Stat(src)
	if os.IsNotExist(err) {
		return
	}

	// Make directory for trash
	dest = rm_trash + "/" + time.Now().Format("2006/01/02")
	err = os.MkdirAll(dest, 0777)
	if err != nil {
		return
	}

	// Remove
	dest = dest + "/" + filepath.Base(src) + "." + time.Now().Format("15_04_05")
	err = os.Rename(src, dest)
	if err != nil {
		return
	}
	return
}
