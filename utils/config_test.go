package utils

import (
	"log"
	"testing"
)

func TestReadConfig(t *testing.T) {
	c := ReadConfig()

	log.Print(c)
}