package tui

import (
	"fmt"
	"testing"
)

func TestZZDocs(t *testing.T) {
	m := drive(t, boot(t, true, 100, 30), key("D"))
	fmt.Println(stripANSI(m.View()))
}
