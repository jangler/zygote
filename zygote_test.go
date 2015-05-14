package main

import (
	"testing"

	"github.com/jangler/tktext"
)

func TestIndexPos(t *testing.T) {
	text := tktext.New()
	text.SetSize(10, 5)
	text.SetWrap(tktext.Char)
	if got, want := indexPos(text, "end", 8), "1,0"; got != want {
		t.Errorf("indexPos() == %#v; want %#v", got, want)
	}
	text.Insert("end", "func main() {\n\tprintln(\"hello\")\n}")
	if got, want := indexPos(text, "1.end", 8), "1,13"; got != want {
		t.Errorf("indexPos() == %#v; want %#v", got, want)
	}
	if got, want := indexPos(text, "2.end", 8), "2,17-24"; got != want {
		t.Errorf("indexPos() == %#v; want %#v", got, want)
	}
	if got, want := indexPos(text, "3.end", 8), "3,1"; got != want {
		t.Errorf("indexPos() == %#v; want %#v", got, want)
	}
}

func TestScrollPercent(t *testing.T) {
	if got, want := scrollPercent(0, 1), "All"; got != want {
		t.Errorf("scrollPercent(0, 1) == %#v; want %#v", got, want)
	}
	if got, want := scrollPercent(0, 0.5), "0%"; got != want {
		t.Errorf("scrollPercent(0, 0.5) == %#v; want %#v", got, want)
	}
	if got, want := scrollPercent(0.5, 0.75), "66%"; got != want {
		t.Errorf("scrollPercent(0.5, 0.75) == %#v; want %#v", got, want)
	}
	if got, want := scrollPercent(0.5, 1), "100%"; got != want {
		t.Errorf("scrollPercent(0.5, 1) == %#v; want %#v", got, want)
	}
}
