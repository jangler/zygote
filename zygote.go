package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
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

	// Status line
	statusFg   termbox.Attribute
	statusMsg  string
	promptMode int

	// Text buffers
	mainText   = tktext.New()
	promptText = tktext.New()
	focusText  = mainText

	// Event channels
	eventChan = make(chan termbox.Event)
	quitChan  = make(chan bool)
)

// Draw the given string in the given style, starting at the given screen
// coordinates
func drawString(x, y int, s string, fg, bg termbox.Attribute) {
	for i, ch := range s {
		termbox.SetCell(x+i, y, ch, fg, bg)
	}
}

// Draw the given string in the default style, starting at the given screen
// coordinates
func drawStringDefault(x, y int, s string) {
	drawString(x, y, s, termbox.ColorDefault, termbox.ColorDefault)
}

// Return index position as a string (e.g. "1,1-8") from the given buffer,
// index, and tabstop
func indexPos(t *tktext.TkText, index string, ts int) string {
	cursor := t.Index(index)
	tabCount := strings.Count(t.Get(index+" linestart", index), "\t")
	col := cursor.Char + tabCount*(ts-1)
	if cursor.Char == col {
		return fmt.Sprintf("%d,%d", cursor.Line, cursor.Char)
	}
	return fmt.Sprintf("%d,%d-%d", cursor.Line, cursor.Char, col)
}

// Return scroll percentage as a string (e.g. "50%") from TkText.YView() values
func scrollPercent(view1, view2 float64) string {
	frac := view1 / (1.0 - (view2 - view1))
	if view2 == 1 && view1 == 0 {
		frac = 1
	}
	return fmt.Sprintf("%d%%", int(frac*100))
}

// Set the status message to the given string, with normal attribute
func msgNormal(s string) {
	statusMsg = s
	statusFg = termbox.ColorDefault
}

// Set the status message to the given string, with error attribute
func msgError(s string) {
	statusMsg = s
	statusFg = termbox.ColorRed
}

// Draw the entire screen
func draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	width, height := termbox.Size()

	mainText.SetSize(width, height-1)
	mainText.See(cursorMark)
	for i, line := range mainText.GetScreenLines() {
		drawStringDefault(0, i, line)
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

		drawStringDefault(0, height-1, s)
		x := len(s)
		s = promptText.Get("1.0", "end")
		drawStringDefault(x, height-1, s)
		pos := promptText.Index(cursorMark)
		termbox.SetCursor(x+pos.Char, height-1)
	} else if statusMsg == "" {
		// Draw cursor row,col numbers and scroll percentage
		pos := indexPos(mainText, cursorMark, tabStop)
		drawStringDefault(width-17, height-1, pos)
		drawStringDefault(width-4, height-1, scrollPercent(mainText.YView()))
	} else {
		drawString(0, height-1, statusMsg, statusFg, termbox.ColorDefault)
	}

	err := termbox.Flush()
	if err != nil {
		msgError(err.Error())
	} else {
		msgNormal("")
	}
}

// Enter the rune into the focused buffer. Entering line feed into a prompt
// confirms it
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

// Change the cursor's display line by the given delta
func changeLine(d int) {
	if focusText != mainText {
		return
	}

	x, y := mainText.BBox(cursorMark)
	y += d
	mainText.MarkSet(cursorMark, fmt.Sprintf("@%d,%d", x, y))
}

// Attempt to read the file with the given path into the buffer
func openFile(path string) {
	if p, err := ioutil.ReadFile(path); err == nil {
		mainText.Delete("1.0", "end")
		mainText.Insert("1.0", string(p))
		mainText.MarkSet(cursorMark, "1.0")
		mainText.EditReset()
		msgNormal(fmt.Sprintf("Opened \"%s\".", path))
		filename = path
	} else {
		msgError(err.Error())
	}
}

// Enter the given prompt mode
func prompt(mode int) {
	promptText.Delete("1.0", "end")
	promptMode = mode
	focusText = promptText
}

// If no filename, prompt for one. Otherwise, attempt to write the buffer
func saveFile() {
	if focusText != mainText {
		return
	}

	if filename == "" {
		prompt(promptSave)
	} else {
		p := []byte(mainText.Get("1.0", "end"))
		if err := ioutil.WriteFile(filename, p, 0644); err == nil {
			msgNormal(fmt.Sprintf("Saved \"%s\".", filename))
		} else {
			msgError(err.Error())
		}
	}
}

// Suspend the process (like ^Z in bash)
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
		msgError(err.Error())
	}
}

// Cancel out of a prompt
func cancel() {
	if focusText == mainText {
		return
	}

	focusText = mainText
	msgNormal("Cancelled.")
}

// Initialize command-line flags and args
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

// Undo change to main buffer
func undo() {
	if focusText == mainText {
		if !mainText.EditUndo() {
			msgError("Nothing to undo.")
		}
	}
}

// Redo change to main buffer
func redo() {
	if focusText == mainText {
		if !mainText.EditRedo() {
			msgError("Nothing to redo.")
		}
	}
}

// Take appropriate action for the given termbox event
func handleEvent(event termbox.Event) {
	stop := false

	switch event.Type {
	case termbox.EventError:
		msgError(event.Err.Error())
		draw()
	case termbox.EventKey:
		sep := false // Whether an undo separator should be inserted

		switch event.Key {
		case termbox.KeyArrowDown, termbox.KeyCtrlJ:
			changeLine(1)
			sep = true
		case termbox.KeyArrowLeft, termbox.KeyCtrlH:
			focusText.MarkSet(cursorMark, cursorMark+"-1c")
			sep = true
		case termbox.KeyArrowRight, termbox.KeyCtrlL:
			focusText.MarkSet(cursorMark, cursorMark+"+1c")
			sep = true
		case termbox.KeyArrowUp, termbox.KeyCtrlK:
			changeLine(-1)
			sep = true
		case termbox.KeyBackspace2: // KeyBackspace == KeyCtrlH
			focusText.Delete(cursorMark+"-1c", cursorMark)
		case termbox.KeyDelete:
			focusText.Delete(cursorMark, cursorMark+"+1c")
		case termbox.KeyEnd, termbox.KeyCtrlE:
			focusText.MarkSet(cursorMark, cursorMark+" lineend")
			sep = true
		case termbox.KeyEnter:
			typeRune('\n')
		case termbox.KeyHome, termbox.KeyCtrlA:
			focusText.MarkSet(cursorMark, cursorMark+" linestart")
			sep = true
		case termbox.KeyPgdn, termbox.KeyCtrlN:
			_, height := termbox.Size()
			changeLine(height - 1)
			sep = true
		case termbox.KeyPgup, termbox.KeyCtrlP:
			_, height := termbox.Size()
			changeLine(-height + 1)
			sep = true
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
		case termbox.KeyCtrlR:
			redo()
		case termbox.KeyCtrlU:
			undo()
		case termbox.KeyCtrlZ:
			stop = true
			suspend()
		default:
			if event.Ch != 0 {
				typeRune(event.Ch)
			} else {
				msgError(fmt.Sprintf("Unbound key: 0x%04X", event.Key))
			}
		}

		if sep && focusText == mainText {
			mainText.EditSeparator()
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

// Poll for an event and send it to the event channel. Blocking function call
func getEvent() {
	eventChan <- termbox.PollEvent()
}

// Event loop
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

// Entry point
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
