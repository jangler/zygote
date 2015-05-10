package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jangler/tktext"
	"github.com/jangler/zygote/tabs"

	"github.com/nsf/termbox-go"
)

const cursorMark = "c"

var (
	// Command-line flags/args
	tabStop  int
	filename string

	statusMsg  string
	mainText   *tktext.Text
	mainScroll int
)

func drawString(x, y int, s string, fg, bg termbox.Attribute) {
	for i, ch := range s {
		termbox.SetCell(x+i, y, ch, fg, bg)
	}
}

func drawTextArea(x, y, w, h, scroll int, s string, focused bool) {
	fg, bg := termbox.ColorDefault, termbox.ColorDefault
	lines := strings.Split(s, "\n")
	cursor := mainText.Index(cursorMark)
	cursor.Row -= 1

	for i, line := range lines {
		if i == cursor.Row && focused {
			cursor.Col = tabs.Columns(line[:cursor.Col], tabStop)
		}

		if line == "" {
			line = " " // Just a hack to make the loop logic simpler
		} else {
			line = tabs.Expand(line, tabStop)
		}

		for line != "" && h > 0 {
			max := len(line)
			if max > w {
				max = w
			}
			if scroll == 0 {
				drawString(x, y, line[:max], fg, bg)
				if i == cursor.Row && focused {
					if cursor.Col >= 0 && cursor.Col <= max {
						termbox.SetCursor(cursor.Col, y)
					}
				}
				y++
				h--
			} else {
				scroll--
			}
			line = line[max:]
			if i == cursor.Row && focused {
				cursor.Col -= max
			}
		}
	}
}

func draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	width, height := termbox.Size()

	drawTextArea(0, 0, width, height-1, mainScroll,
		mainText.Get("1.0", "end").String(), true)

	if statusMsg == "" {
		// Draw cursor row,col numbers
		cursor := mainText.Index(cursorMark)
		index := fmt.Sprintf("%d.0", cursor.Row)
		trueCol := tabs.Columns(mainText.Get(index, cursorMark).String(),
			tabStop)
		if cursor.Col == trueCol {
			statusMsg = fmt.Sprintf("%d,%d", cursor.Row, cursor.Col)
		} else {
			statusMsg = fmt.Sprintf("%d,%d-%d", cursor.Row, cursor.Col,
				trueCol)
		}
		x := width - 17
		drawTextArea(x, height-1, width-x, 1, 0, statusMsg, false)

		// Draw scroll percentage
		x = width - 4
		scrollPercent := 0
		if mainScroll > 0 && mainText.NumLines() >= height {
			scrollPercent = mainScroll * 100 /
				(mainText.NumLines() - (height - 1))
		}
		statusMsg = fmt.Sprintf("%d%%", scrollPercent)
		drawTextArea(x, height-1, width-x, 1, 0, statusMsg, false)
	} else {
		drawTextArea(0, height-1, width, 1, 0, statusMsg, false)
	}

	err := termbox.Flush()
	if err != nil {
		statusMsg = err.Error()
	} else {
		statusMsg = ""
	}
}

func typeRune(ch rune) {
	mainText.Insert(cursorMark, string(ch))
}

func moveMark(m string, d int, t *tktext.Text) {
	t.MarkSet(m, fmt.Sprintf("%s %d", m, d))
}

func scrollLines(d int) {
	mainScroll += d
	if mainScroll < 0 {
		mainScroll = 0
	}
}

func openFile() {
	if p, err := ioutil.ReadFile(filename); err == nil {
		mainText.Delete("1.0", "end")
		mainText.Insert("1.0", string(p))
		mainText.MarkSet(cursorMark, "1.0")
		statusMsg = fmt.Sprintf("Opened \"%s\".", filename)
	} else {
		statusMsg = err.Error()
	}
}

func saveFile() {
	p := mainText.Get("1.0", "end").Bytes()
	if err := ioutil.WriteFile(filename, p, 0644); err == nil {
		statusMsg = fmt.Sprintf("Saved \"%s\".", filename)
	} else {
		statusMsg = err.Error()
	}
}

func initFlags() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<option>...] [<file>]", os.Args[0])
		fmt.Fprint(os.Stderr, "\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.IntVar(&tabStop, "tabstop", 8, "number of columns in a tab stop")

	flag.Parse()

	if flag.NArg() == 1 {
		filename = flag.Arg(0)
	} else if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	initFlags()

	err := termbox.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	defer termbox.Close()

	mainText = tktext.New()
	mainText.MarkSet(cursorMark, "1.0")
	if filename != "" {
		openFile()
	}
	draw()

	loop := true
	for loop {
		switch event := termbox.PollEvent(); event.Type {
		case termbox.EventError:
			statusMsg = event.Err.Error()
			draw()
		case termbox.EventKey:
			switch event.Key {
			case termbox.KeyArrowDown:
				scrollLines(1)
			case termbox.KeyArrowLeft:
				moveMark(cursorMark, -1, mainText)
			case termbox.KeyArrowRight:
				moveMark(cursorMark, 1, mainText)
			case termbox.KeyArrowUp:
				scrollLines(-1)
			case termbox.KeyBackspace2: // KeyBackspace == KeyCtrlH
				mainText.Delete(fmt.Sprintf("%s -1", cursorMark), cursorMark)
			case termbox.KeyDelete:
				mainText.Delete(cursorMark, fmt.Sprintf("%s +1", cursorMark))
			case termbox.KeyEnter:
				typeRune('\n')
			case termbox.KeySpace:
				typeRune(' ')
			case termbox.KeyTab:
				typeRune('\t')
			case termbox.KeyCtrlS:
				saveFile()
			case termbox.KeyCtrlQ:
				loop = false
			default:
				if event.Ch != 0 {
					typeRune(event.Ch)
				} else {
					statusMsg = fmt.Sprintf("Unbound key: 0x%04X", event.Key)
				}
			}
			draw()
		case termbox.EventResize:
			draw()
		}
	}
}
