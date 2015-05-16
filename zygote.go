package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jangler/tktext"
	"github.com/nsf/termbox-go"
)

const (
	cursorMark = "c"
	selMark    = "s"

	selFg = termbox.ColorBlack
	selBg = termbox.ColorBlue

	promptOpen = iota
	promptOpenYN
	promptPut
	promptQuitYN
	promptSave
	promptSaveYN
	promptWrite
	promptWriteWhich
	promptExecute
	promptYank
)

var (
	// Command-line flags/args
	tabStop  = 8
	filename string

	// Status line
	statusFg   termbox.Attribute
	statusMsg  string
	promptMode int

	// Text buffers
	mainText   = tktext.New()
	promptText = tktext.New()
	focusText  = mainText

	manualText *tktext.TkText // Not initialized unless we need it

	cursorCol = map[*tktext.TkText]int{
		mainText:   0,
		manualText: 0,
	}

	register = map[rune]string{
		'F': filename,
		'L': "1",
		'T': fmt.Sprintf("%d", tabStop),
	}
	regRune rune

	// Event channels
	eventChan = make(chan termbox.Event)
	quitChan  = make(chan bool)

	// Modes
	modeManual, modeSelect, modeView, modeWord bool

	// Regexps
	wordRegexp  = regexp.MustCompile(`\w`)
	spaceRegexp = regexp.MustCompile(`\s`)
	formRegexp  = regexp.MustCompile(`^<.+?>`)
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
		return "All"
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

// Returns a status line string describing active modes
func modeString() string {
	modes := make([]string, 0)
	if modeManual {
		modes = append(modes, "manual (M-m)")
	}
	if modeSelect {
		modes = append(modes, "select (M-s)")
	}
	if modeView {
		modes = append(modes, "view (M-v)")
	}
	if modeWord {
		modes = append(modes, "word (M-w)")
	}
	if len(modes) > 0 {
		return "Modes: " + strings.Join(modes, ", ")
	}
	return ""
}

// Draw the entire screen
func draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	width, height := termbox.Size()

	drawText := mainText
	if modeManual {
		drawText = manualText
		drawText.EditUndo() // No changing the manual!
	}
	drawText.SetSize(width, height-1)
	if !modeView {
		drawText.See(cursorMark)
	}
	curX, curY := drawText.BBox(cursorMark)
	selX, selY := curX, curY
	if modeSelect {
		selX, selY = drawText.BBox(selMark)
	}
	if selY > curY || (selY == curY && selX > curX) {
		selX, selY, curX, curY = curX, curY, selX, selY
	}
	for i, line := range drawText.GetScreenLines() {
		if !modeSelect {
			drawStringDefault(0, i, line)
		} else if i > selY {
			if i < curY {
				drawString(0, i, line, selFg, selBg)
			} else if i == curY {
				drawString(0, i, line[:curX], selFg, selBg)
				drawStringDefault(curX, i, line[curX:])
			} else {
				drawStringDefault(0, i, line)
			}
		} else if i == selY {
			if i < curY {
				drawStringDefault(0, i, line[:selX])
				drawString(selX, i, line[selX:], selFg, selBg)
			} else if i == curY {
				drawStringDefault(0, i, line[:selX])
				drawString(selX, i, line[selX:curX], selFg, selBg)
				drawStringDefault(curX, i, line[curX:])
			}
		} else {
			drawStringDefault(0, i, line)
		}
	}
	curX, curY = drawText.BBox(cursorMark)
	if curY >= 0 && curY < height-1 {
		termbox.SetCursor(curX, curY)
	} else {
		termbox.HideCursor()
	}

	if focusText == promptText {
		var s string
		switch promptMode {
		case promptOpen:
			s = "Open file: "
		case promptOpenYN, promptQuitYN:
			s = "Abandon unsaved changes? (y/n): "
		case promptPut:
			s = "Put from register: "
		case promptSave:
			s = "Save as: "
		case promptSaveYN:
			s = "Overwrite file? (y/n): "
		case promptWrite:
			s = "Write: "
		case promptWriteWhich:
			s = "Write into register: "
		case promptExecute:
			s = "Execute from register: "
		case promptYank:
			s = "Yank into register: "
		}

		drawStringDefault(0, height-1, s)
		x := len(s)
		s = promptText.Get("1.0", "end")
		if modeSelect {
			selX, _ := promptText.BBox(selMark)
			curX, _ := promptText.BBox(cursorMark)
			if selX > curX {
				selX, curX = curX, selX
			}
			drawStringDefault(x, height-1, s[:selX])
			drawString(x+selX, height-1, s[selX:curX], selFg, selBg)
			drawStringDefault(x+curX, height-1, s[curX:])
		} else {
			drawStringDefault(x, height-1, s)
		}
		pos := promptText.Index(cursorMark)
		termbox.SetCursor(x+pos.Char, height-1)
	} else if statusMsg == "" {
		// Draw modes, cursor row,col numbers, and scroll percentage
		drawStringDefault(0, height-1, modeString())
		pos := indexPos(drawText, cursorMark, tabStop)
		drawStringDefault(width-17, height-1, pos)
		drawStringDefault(width-4, height-1, scrollPercent(drawText.YView()))
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

// Reset the focus when leaving a prompt
func unprompt() {
	if modeManual {
		focusText = manualText
	} else {
		focusText = mainText
	}
}

func getRegister(ch rune) string {
	var s string
	switch ch {
	case 'F':
		s = filename
	case 'L':
		s = fmt.Sprintf("%d", focusText.Index(cursorMark).Line)
	case 'T':
		s = fmt.Sprintf("%d", tabStop)
	default:
		s = register[ch]
	}
	return s
}

func setRegister(ch rune, s string) {
	switch ch {
	case 'F':
		filename = s
	case 'L':
		if n, err := strconv.ParseInt(s, 10, 0); err == nil {
			if n < 0 {
				n = 0
			}
			pos := focusText.Index(cursorMark)
			focusText.MarkSet(cursorMark,
				fmt.Sprintf("%d.%d", n, pos.Char))
		} else {
			msgError(err.Error())
		}
	case 'T':
		if n, err := strconv.ParseInt(s, 10, 0); err == nil {
			if n < 1 {
				n = 1
			}
			tabStop = int(n)
			mainText.SetTabStop(tabStop)
		} else {
			msgError(err.Error())
		}
	default:
		register[ch] = s
	}
}

func getSelText() string {
	if modeSelect && focusText.Compare(cursorMark, selMark) != 0 {
		if focusText.Compare(selMark, cursorMark) < 0 {
			return focusText.Get(selMark, cursorMark)
		} else {
			return focusText.Get(cursorMark, selMark)
		}
	}
	return focusText.Get(cursorMark, cursorMark+"+1c")
}

func handleKey(s string) bool {
	stop := false
	sep := false     // Whether an undo separator should be inserted
	resetCol := true // Whether cursorCol should be reset

	switch s {
	case "<Down>":
		if modeView && focusText != promptText {
			focusText.YViewScroll(1)
		} else {
			changeLine(1)
			sep, resetCol = true, false
		}
	case "<Left>":
		moveCursor("-1c")
		sep = true
	case "<Right>":
		moveCursor("+1c")
		sep = true
	case "<Up>":
		if modeView && focusText != promptText {
			focusText.YViewScroll(-1)
		} else {
			changeLine(-1)
			sep, resetCol = true, false
		}
	case "<Backspace>", "<C-8>":
		del("-1c")
	case "<Delete>":
		del("+1c")
	case "<End>":
		focusText.MarkSet(cursorMark, cursorMark+" lineend")
		sep = true
	case "<Enter>", "<C-m>":
		typeRune('\n')
	case "<Home>":
		focusText.MarkSet(cursorMark, cursorMark+" linestart")
		sep = true
	case "<PgDn>":
		_, height := termbox.Size()
		if modeView && focusText != promptText {
			focusText.YViewScroll(height - 1)
		} else {
			changeLine(height - 1)
			sep, resetCol = true, false
		}
	case "<PgUp>":
		_, height := termbox.Size()
		if modeView && focusText != promptText {
			focusText.YViewScroll(-(height - 1))
		} else {
			changeLine(-(height - 1))
			sep, resetCol = true, false
		}
	case "<Space>":
		typeRune(' ')
	case "<Tab>", "<C-i>":
		typeRune('\t')
	case "<C-c>":
		cancel()
	case "<C-o>":
		if mainText.EditGetModified() {
			prompt(promptOpenYN)
		} else {
			prompt(promptOpen)
		}
	case "<C-p>":
		prompt(promptPut)
	case "<C-s>":
		saveFile(true)
	case "<C-q>":
		if mainText.EditGetModified() {
			prompt(promptQuitYN)
		} else {
			stop = true
			quitChan <- true
		}
	case "<C-r>":
		redo()
	case "<C-u>":
		undo()
	case "<C-w>":
		prompt(promptWriteWhich)
	case "<C-x>":
		prompt(promptExecute)
	case "<C-y>":
		prompt(promptYank)
	case "<C-z>":
		stop = true
		suspend()
	case "<M-m>":
		toggleManual()
	case "<M-s>":
		toggleSelect()
	case "<M-v>":
		modeView = !modeView
	case "<M-w>":
		modeWord = !modeWord
	default:
		if len(s) > 1 {
			msgError("Unbound key: " + s)
		} else {
			// Loop only iterates once
			for _, ch := range s {
				stop = typeRune(ch)
			}
		}
	}

	if resetCol {
		cursorCol[focusText] = 0
	}
	if sep && focusText == mainText {
		mainText.EditSeparator()
	}
	return stop
}

func execString(s string) {
	for len(s) > 0 {
		if match := formRegexp.FindString(s); match != "" {
			handleKey(match)
			s = s[len(match):]
		} else {
			handleKey(s[0:1])
			s = s[1:]
		}
	}
}

// Enter the rune into the focused buffer. Entering line feed into a prompt
// confirms it. Returns true if the event loop should stop
func typeRune(ch rune) bool {
	if focusText == promptText && (promptMode == promptOpenYN ||
		promptMode == promptSaveYN || promptMode == promptQuitYN) {
		if ch == 'y' {
			switch promptMode {
			case promptOpenYN:
				prompt(promptOpen)
			case promptSaveYN:
				unprompt()
				saveFile(true)
			case promptQuitYN:
				quitChan <- true
				return true
			}
		} else if ch == 'n' {
			unprompt()
		}
	} else if focusText == promptText && (promptMode == promptPut ||
		promptMode == promptWriteWhich || promptMode == promptYank ||
		promptMode == promptExecute) {
		regRune = ch
		unprompt()
		switch promptMode {
		case promptPut:
			focusText.Insert(cursorMark, getRegister(ch))
		case promptWriteWhich:
			prompt(promptWrite)
		case promptExecute:
			execString(getRegister(ch))
		case promptYank:
			setRegister(ch, getSelText())
		}
	} else if ch == '\n' && focusText == promptText {
		unprompt()
		switch promptMode {
		case promptOpen:
			openFile(promptText.Get("1.0", "end"))
		case promptSave:
			filename = promptText.Get("1.0", "end")
			saveFile(false)
		case promptWrite:
			setRegister(regRune, promptText.Get("1.0", "end"))
		}
	} else {
		s := string(ch)

		// Autoindent
		if ch == '\n' {
			prevLine := focusText.Get(cursorMark+" linestart", cursorMark)
			i := 0
			for _, c := range prevLine {
				if c != ' ' && c != '\t' {
					break
				}
				i++
			}
			s += prevLine[:i]

			// Delete empty lines
			if i == len(prevLine) {
				focusText.Delete(cursorMark+" linestart", cursorMark)
			}
		}

		focusText.Insert(cursorMark, s)
	}

	return false
}

// Change the cursor's display line by the given delta
func changeLine(d int) {
	if focusText != promptText {
		x, y := focusText.BBox(cursorMark)
		if x > cursorCol[focusText] {
			cursorCol[focusText] = x
		} else {
			x = cursorCol[focusText]
		}
		y += d
		focusText.MarkSet(cursorMark, fmt.Sprintf("@%d,%d", x, y))
	}
}

// Attempt to read the file with the given path into the buffer
func openFile(path string) {
	if p, err := ioutil.ReadFile(path); err == nil {
		mainText.Delete("1.0", "end")
		mainText.Insert("1.0", string(p))
		mainText.MarkSet(cursorMark, "1.0")
		mainText.EditReset()
		mainText.EditSetModified(false)
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
func saveFile(overwrite bool) {
	if focusText != mainText {
		return
	}

	if filename == "" {
		prompt(promptSave)
	} else {
		_, err := os.Stat(filename)
		if err == nil && !overwrite {
			prompt(promptSaveYN)
			return
		}

		p := []byte(mainText.Get("1.0", "end"))

		// Ensure file has final newline
		if len(p) > 0 && p[len(p)-1] != '\n' {
			p = append(p, '\n')
		}

		if err := ioutil.WriteFile(filename, p, 0644); err == nil {
			mainText.EditSetModified(false)
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
		termbox.SetInputMode(termbox.InputAlt)
		draw()
		getEvent()
	} else {
		msgError(err.Error())
	}
}

// Cancel out of a prompt
func cancel() {
	if focusText == promptText {
		unprompt()
		msgNormal("Cancelled.")
	}
}

// Initialize command-line flags and args
func initFlags() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<file>]\n", os.Args[0])
		//fmt.Fprint(os.Stderr, "\n\nOptions:\n")
		//flag.PrintDefaults()
	}

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
		if !mainText.EditUndo(cursorMark) {
			msgError("Nothing to undo.")
		}
	}
}

// Redo change to main buffer
func redo() {
	if focusText == mainText {
		if !mainText.EditRedo(cursorMark) {
			msgError("Nothing to redo.")
		}
	}
}

// Toggle manual mode
func toggleManual() {
	if focusText != promptText {
		modeManual = !modeManual
		if manualText == nil {
			manualText = tktext.New()
			manualText.Insert("end", manualString)
			manualText.EditReset()
			manualText.MarkSet(cursorMark, "1.0")
			manualText.MarkSet(selMark, cursorMark)
			manualText.MarkSetGravity(selMark, tktext.Left)
		}
		unprompt()
	}
}

// Toggle select mode
func toggleSelect() {
	modeSelect = !modeSelect
	mainText.MarkSet(selMark, cursorMark)
	if manualText != nil {
		manualText.MarkSet(selMark, cursorMark)
	}
	promptText.MarkSet(selMark, cursorMark)
}

// This function is nasty.
func moveCursor(modifier string) {
	if modeWord {
		pos := focusText.Index(cursorMark)
		line := focusText.Get(cursorMark+" linestart", cursorMark+" lineend")
		if modifier == "+1c" {
			llen := len(line)
			if pos.Char == llen {
				if focusText != promptText {
					pos.Line++
					pos = focusText.Index(pos.String() + " linestart")
				}
			} else {
				if wordRegexp.Match([]byte{line[pos.Char]}) {
					for pos.Char < llen &&
						wordRegexp.Match([]byte{line[pos.Char]}) {
						pos.Char++
					}
				} else {
					for pos.Char < llen &&
						!wordRegexp.Match([]byte{line[pos.Char]}) {
						pos.Char++
					}
				}
				for pos.Char < llen &&
					spaceRegexp.Match([]byte{line[pos.Char]}) {
					pos.Char++
				}
			}
			focusText.MarkSet(cursorMark, pos.String())
		} else if modeWord && modifier == "-1c" {
			if pos.Char == 0 {
				if focusText != promptText {
					pos.Line--
					pos = focusText.Index(pos.String() + " lineend")
				}
			} else {
				if wordRegexp.Match([]byte{line[pos.Char-1]}) {
					for pos.Char > 0 &&
						wordRegexp.Match([]byte{line[pos.Char-1]}) {
						pos.Char--
					}
				} else {
					for pos.Char > 0 &&
						!wordRegexp.Match([]byte{line[pos.Char-1]}) {
						pos.Char--
					}
				}
				for pos.Char > 0 &&
					spaceRegexp.Match([]byte{line[pos.Char-1]}) {
					pos.Char--
				}
			}
			focusText.MarkSet(cursorMark, pos.String())
		} else {
			focusText.MarkSet(cursorMark, cursorMark+modifier)
		}
	} else {
		focusText.MarkSet(cursorMark, cursorMark+modifier)
	}
}

// Delete text. If there is a selection, delete the selection. Otherwise,
// select text from the cursor to the modifier, then delete it.
func del(modifier string) {
	if !modeSelect || focusText.Compare(selMark, cursorMark) == 0 {
		focusText.MarkSet(selMark, cursorMark)
		moveCursor(modifier)
	}
	if focusText.Compare(selMark, cursorMark) < 0 {
		focusText.Delete(selMark, cursorMark)
	} else {
		focusText.Delete(cursorMark, selMark)
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
		stop = handleKey(keyString(event))
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
	termbox.SetInputMode(termbox.InputAlt)

	mainText.SetWrap(tktext.Char)
	mainText.SetTabStop(tabStop)
	mainText.MarkSet(cursorMark, "end")
	mainText.MarkSet(selMark, cursorMark)
	mainText.MarkSetGravity(selMark, tktext.Left)
	promptText.MarkSet(cursorMark, "end")
	promptText.MarkSet(selMark, cursorMark)
	promptText.MarkSetGravity(selMark, tktext.Left)
	msgNormal("Zygote, alpha version. Press M-m to view manual.")
	if filename != "" {
		openFile(filename)
	}
	draw()

	go getEvent()
	handleEvents()
}
