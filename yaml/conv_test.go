package yaml

import (
	"log"
	"testing"
)

func TestConv(*testing.T) {
	strs := []string{"true", "false", "100", "3.14", "2012-05-30", "2011-09-19 14:33:04"}
	for _, str := range strs {
		log.Println("++>>>", str, string2Val(str))
	}
}
