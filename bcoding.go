package main

import (
	"bufio"
	"fmt"
)

type BCode interface {
	renderBCode(*bufio.Writer)
}

type BString string
type BUint uint
type BList []BCode
type BMap map[string]BCode

func (s BString) renderBCode(w *bufio.Writer) {
	w.WriteString(fmt.Sprintf("%d:%s", len(s), s))
}

func (u BUint) renderBCode(w *bufio.Writer) {
	w.WriteString(fmt.Sprintf("i%de", u))
}

func (l BList) renderBCode(w *bufio.Writer) {
	w.WriteString("l")
	for _, e := range l {
		e.renderBCode(w)
	}
	w.WriteString("e")
}

func (p BMap) renderBCode(w *bufio.Writer) {
	w.WriteString("d")
	for k, v := range p {
		BString(k).renderBCode(w)
		v.renderBCode(w)
	}
	w.WriteString("e")
}
