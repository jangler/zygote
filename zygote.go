package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/jangler/tktext"
	"github.com/nsf/termbox-go"
)

const cursorMark = "c"

const (
	promptOpen = iota
	promptSave
)

var (
	// Command-line flags/args
	tabStop  int
	filename string

	statusMsg  string
	promptMode int

	mainText   = tktext.New()
	promptText = tktext.New()
	focusText  = mainText

	eventChan = make(chan termbox.Event)
	quitChan  = make(chan bool)
)

func drawString(x, y int, s string, fg, bg termbox.Attribute) {
	for i, ch := range s {
		termbox.SetCell(x+i, y, ch, fg, bg)
	}
}

func draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	width, height := termbox.Size()

	mainText.SetSize(width, height-1)
	mainText.See(cursorMark)
	for i, line := range mainText.GetScreenLines() {
		drawString(0, i, line, termbox.ColorDefault, termbox.ColorDefault)
	}
	termbox.SetCursor(mainText.BBox(cursorMark))

	if focusText == promptText {
		var s string
		switch promptMode {
		case promptOpen:
			s = "Open file: "
		case promptSave:
			s = "Save as: "
		}

		drawString(0, height-1, s, termbox.ColorDefault, termbox.ColorDefault)
		x := len(s)
		s = promptText.Get("1.0", "end")
		drawString(x, height-1, s, termbox.ColorDefault, termbox.ColorDefault)
		pos := promptText.Index(cursorMark)
		termbox.SetCursor(x+pos.Char, height-1)
	} else if statusMsg == "" {
		// Draw cursor row,col numbers
		cursor := mainText.Index(cursorMark)
		col, _ := mainText.BBox(cursorMark)
		if cursor.Char == col {
			statusMsg = fmt.Sprintf("%d,%d", cursor.Line, cursor.Char)
		} else {
			statusMsg = fmt.Sprintf("%d,%d-%d", cursor.Line, cursor.Char, col)
		}
		x := width - 17
		drawString(x, height-1, statusMsg, termbox.ColorDefault,
			termbox.ColorDefault)

		// Draw scroll percentage
		x = width - 4
		view1, view2 := mainText.YView()
		scrollFraction := view1 / (1.0 - (view2 - view1))
		if view2 == 1 && view1 == 0 {
			scrollFraction = 1
		}
		statusMsg = fmt.Sprintf("%d%%", int(scrollFraction*100))
		drawString(x, height-1, statusMsg, termbox.ColorDefault,
			termbox.ColorDefault)
	} else {
		drawString(0, height-1, statusMsg, termbox.ColorDefault,
			termbox.ColorDefault)
	}

	err := termbox.Flush()
	if err != nil {
		statusMsg = err.Error()
	} else {
		statusMsg = ""
	}
}

func typeRune(ch rune) {
	if ch == '\n' && focusText == promptText {
		focusText = mainText
		switch promptMode {
		case promptOpen:
			openFile(promptText.Get("1.0", "end"))
		case promptSave:
			filename = promptText.Get("1.0", "end")
			saveFile()
		}
	} else {
		focusText.Insert(cursorMark, string(ch))
	}
}

func changeLine(d int) {
	if focusText != mainText {
		return
	}

	x, y := mainText.BBox(cursorMark)
	y += d
	mainText.MarkSet(cursorMark, fmt.Sprintf("@%d,%d", x, y))
}

func openFile(fn string) {
	if p, err := ioutil.ReadFile(fn); err == nil {
		mainText.Delete("1.0", "end")
		mainText.Insert("1.0", string(p))
		mainText.MarkSet(cursorMark, "1.0")
		statusMsg = fmt.Sprintf("Opened \"%s\".", fn)
		filename = fn
	} else {
		statusMsg = err.Error()
	}
}

func prompt(mode int) {
	promptText.Delete("1.0", "end")
	promptMode = mode
	focusText = promptText
}

func saveFile() {
	if focusText != mainText {
		return
	}

	if filename == "" {
		prompt(promptSave)
	} else {
		p := []byte(mainText.Get("1.0", "end"))
		if err := ioutil.WriteFile(filename, p, 0644); err == nil {
			statusMsg = fmt.Sprintf("Saved \"%s\".", filename)
		} else {
			statusMsg = err.Error()
		}
	}
}

func suspend() {
	if proc, err := os.FindProcess(os.Getpid()); err == nil {
		// Clean up and send SIGSTOP to this process
		termbox.Close()
		proc.Signal(syscall.SIGSTOP)

		// Hope that 100ms is enough for the process to receive the signal
		time.Sleep(time.Second / 10)

		// Hopefully by now we've got SIGCONT and can re-init things
		termbox.Init()
		draw()
		getEvent()
	} else {
		statusMsg = err.Error()
	}
}

func cancel() {
	if focusText == mainText {
		return
	}

	focusText = mainText
	statusMsg = "Cancelled."
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

func handleEvent(event termbox.Event) {
	stop := false

	switch event.Type {
	case termbox.EventError:
		statusMsg = event.Err.Error()
		draw()
	case termbox.EventKey:
		switch event.Key {
		case termbox.KeyArrowDown, termbox.KeyCtrlJ:
			changeLine(1)
		case termbox.KeyArrowLeft, termbox.KeyCtrlH:
			focusText.MarkSet(cursorMark, cursorMark+"-1c")
		case termbox.KeyArrowRight, termbox.KeyCtrlL:
			focusText.MarkSet(cursorMark, cursorMark+"+1c")
		case termbox.KeyArrowUp, termbox.KeyCtrlK:
			changeLine(-1)
		case termbox.KeyBackspace2: // KeyBackspace == KeyCtrlH
			focusText.Delete(cursorMark+"-1c", cursorMark)
		case termbox.KeyDelete:
			focusText.Delete(cursorMark, cursorMark+"+1c")
		case termbox.KeyEnd, termbox.KeyCtrlE:
			focusText.MarkSet(cursorMark, cursorMark+" lineend")
		case termbox.KeyEnter:
			typeRune('\n')
		case termbox.KeyHome, termbox.KeyCtrlA:
			focusText.MarkSet(cursorMark, cursorMark+" linestart")
		case termbox.KeyPgdn, termbox.KeyCtrlD:
			_, height := termbox.Size()
			changeLine(height - 1)
		case termbox.KeyPgup, termbox.KeyCtrlU:
			_, height := termbox.Size()
			changeLine(-height + 1)
		case termbox.KeySpace:
			typeRune(' ')
		case termbox.KeyTab:
			typeRune('\t')
		case termbox.KeyCtrlC:
			cancel()
		case termbox.KeyCtrlO:
			prompt(promptOpen)
		case termbox.KeyCtrlS:
			saveFile()
		case termbox.KeyCtrlQ:
			stop = true
			quitChan <- true
		case termbox.KeyCtrlZ:
			stop = true
			suspend()
		default:
			if event.Ch != 0 {
				typeRune(event.Ch)
			} else {
				statusMsg = fmt.Sprintf("Unbound key: 0x%04X", event.Key)
			}
		}
		if !stop {
			draw()
		}
	case termbox.EventResize:
		draw()
	}

	if !stop {
		getEvent()
	}
}

func getEvent() {
	eventChan <- termbox.PollEvent()
}

func handleEvents() {
	for {
		select {
		case event := <-eventChan:
			go handleEvent(event)
		case <-quitChan:
			return
		}
	}
}

func main() {
	initFlags()

	if err := termbox.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	defer termbox.Close()

	mainText.SetWrap(tktext.Char)
	mainText.SetTabStop(tabStop)
	mainText.MarkSet(cursorMark, "end")
	promptText.MarkSet(cursorMark, "end")
	if filename != "" {
		openFile(filename)
	}
	draw()

	go getEvent()
	handleEvents()
}
