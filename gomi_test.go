package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestRemoveError(t *testing.T) {
	actual, err := remove("nofile")
	expected := ""
	if actual != expected {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
	if err == nil {
		t.Errorf("%v is nil", err)
	}
}

func TestRemoveSuccess(t *testing.T) {
	actual, err := remove("./README.md")
	f := fmt.Sprintf("%s/.gomi/%s/%s/%s/README.md.%s_%s_%s",
		os.Getenv("HOME"),
		time.Now().Format("2006"),
		time.Now().Format("01"),
		time.Now().Format("02"),
		time.Now().Format("15"),
		time.Now().Format("04"),
		time.Now().Format("05"),
	)
	expected := f
	if actual != expected {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
	if err != nil {
		t.Errorf("%v is not nil", err)
	}
}
