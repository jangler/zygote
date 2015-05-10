// Package text implements a text-editing buffer with an interface like that of
// the tcl/tk Text widget. The buffer is thread-safe.
package text

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var indexRegexp = regexp.MustCompile(`(\d+)\.(\w+)`)

// Position represents a position in a text buffer.
type Position struct {
	Row, Col int
}

// String returns a string representation of the position that can be used as
// an index in buffer functions.
func (p Position) String() string {
	return fmt.Sprintf("%d.%d", p.Row, p.Col)
}

type insertOp struct {
	sp, ep, s string
}

type deleteOp struct {
	sp, ep, s string
}

type separator bool

// Buffer represents a text buffer.
type Buffer struct {
	lines, undoStack, redoStack *list.List
	marks                       map[string]*Position
	mutex                       *sync.RWMutex
}

// NewBuffer returns an initialized Buffer.
func NewBuffer() *Buffer {
	b := Buffer{list.New(), list.New(), list.New(), make(map[string]*Position),
		&sync.RWMutex{}}
	b.lines.PushBack("")
	return &b
}

func mustParseInt(s string) int {
	n, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		panic(err)
	}
	return int(n)
}

func (b *Buffer) getLine(n int) *list.Element {
	i, line := 1, b.lines.Front()
	for i < n {
		line = line.Next()
		i++
	}
	return line
}

// Index returns the row and column numbers of an index into b.
func (b *Buffer) Index(index string) Position {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	var pos Position
	words := strings.Split(index, " ")

	// Parse initial index
	if words[0] == "end" {
		// End keyword
		pos.Row = b.lines.Len()
		pos.Col = len(b.lines.Back().Value.(string))
	} else if markPos, ok := b.marks[words[0]]; ok {
		// Marks
		pos.Row = markPos.Row
		pos.Col = markPos.Col
	} else {
		// Match "row.col" format
		matches := indexRegexp.FindStringSubmatch(words[0])
		if matches == nil {
			panic(errors.New(fmt.Sprintf("Bad index: %#v", index)))
		}

		// Parse row
		pos.Row = mustParseInt(matches[1])
		if pos.Row < 1 {
			pos.Row = 1
			pos.Col = 0
		} else if pos.Row > b.lines.Len() {
			pos.Row = b.lines.Len()
			pos.Col = len(b.lines.Back().Value.(string))
		} else {
			// Parse col
			length := len(b.getLine(pos.Row).Value.(string))
			if matches[2] == "end" {
				pos.Col = length
			} else {
				pos.Col = mustParseInt(matches[2])
				if pos.Col > length {
					pos.Col = length
				}
			}
		}
	}

	// Parse offsets
	for _, word := range words[1:] {
		// Keep in mind that a newline counts as a character
		offset := mustParseInt(word)
		if offset >= 0 {
			line := b.getLine(pos.Row)
			length := len(line.Value.(string))
			for offset+pos.Col > length && line.Next() != nil {
				offset -= length - pos.Col + 1
				pos.Row++
				pos.Col = 0
				line = line.Next()
				length = len(line.Value.(string))
			}
			if offset+pos.Col <= length {
				pos.Col += offset
			} else {
				pos.Col = length
			}
		} else {
			offset = -offset
			for offset > pos.Col && pos.Row > 1 {
				offset -= pos.Col + 1
				pos.Row--
				pos.Col = len(b.getLine(pos.Row).Value.(string))
			}
			if offset <= pos.Col {
				pos.Col -= offset
			} else {
				pos.Col = 0
			}
		}
	}

	return pos
}

// Get returns the text from start to end indices in b.
func (b *Buffer) Get(startIndex, endIndex string) *bytes.Buffer {
	// Parse indices
	start := b.Index(startIndex)
	end := b.Index(endIndex)

	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Find starting line
	i, line := 1, b.lines.Front()
	for i < start.Row {
		line = line.Next()
		i++
	}

	// Write text to buffer
	var text bytes.Buffer
	for i <= end.Row {
		if i != start.Row {
			text.WriteString("\n")
		}
		s := line.Value.(string)
		if i == start.Row {
			if i == end.Row {
				text.WriteString(s[start.Col:end.Col])
			} else {
				text.WriteString(s[start.Col:])
			}
		} else if i == end.Row {
			text.WriteString(s[:end.Col])
		} else {
			text.WriteString(s)
		}
		line = line.Next()
		i++
	}

	return &text
}

func (b *Buffer) del(startIndex, endIndex string, undo bool) {
	// Parse indices
	start := b.Index(startIndex)
	end := b.Index(endIndex)

	b.mutex.Lock()

	// Find starting line
	i, line := 1, b.lines.Front()
	for i < start.Row {
		line = line.Next()
		i++
	}

	// Delete text
	db := &bytes.Buffer{}
	for i <= end.Row {
		if i == start.Row {
			s := line.Value.(string)
			if i == end.Row {
				line.Value = s[:start.Col] + s[end.Col:]
				db.WriteString(s[start.Col:end.Col])
			} else {
				line.Value = s[:start.Col]
				db.WriteString(s[start.Col:] + "\n")
			}
		} else if i == end.Row {
			endLine := line.Next()
			line.Value = line.Value.(string) + endLine.Value.(string)[end.Col:]
			db.WriteString(endLine.Value.(string)[:end.Col])
			b.lines.Remove(endLine)
		} else {
			next := line.Next()
			db.WriteString(next.Value.(string) + "\n")
			b.lines.Remove(next)
		}
		i++
	}

	// Update marks
	for _, pos := range b.marks {
		if pos.Row == end.Row && pos.Col >= end.Col {
			pos.Col += start.Col - end.Col
		} else if pos.Row == start.Row && pos.Col >= start.Col {
			pos.Col = start.Col
		}
		if start.Row != end.Row &&
			((pos.Row == start.Row && pos.Col > start.Col) ||
				(pos.Row > start.Row && pos.Row < end.Row) ||
				(pos.Row == end.Row && pos.Col < end.Col)) {
			pos.Col = start.Col
		}
		if pos.Row >= end.Row {
			pos.Row -= end.Row - start.Row
		}
	}

	b.mutex.Unlock()

	if undo {
		sp := start.String()
		ep := end.String()
		b.mutex.Lock()
		front := b.undoStack.Front()
		collapsed := false
		if front != nil {
			switch v := front.Value.(type) {
			case deleteOp:
				if v.sp == sp {
					ep = fmt.Sprintf("%s +%d", ep, len(v.s))
					front.Value = deleteOp{sp, ep, v.s + db.String()}
					collapsed = true
				} else if v.sp == ep {
					ep = fmt.Sprintf("%s +%d", ep, len(v.s))
					front.Value = deleteOp{sp, ep, db.String() + v.s}
					collapsed = true
				}
			}
		}
		if !collapsed {
			b.undoStack.PushFront(deleteOp{sp, ep, db.String()})
		}
		b.mutex.Unlock()
	}
}

// Delete deletes the text from start to end indices in b.
func (b *Buffer) Delete(startIndex, endIndex string) {
	if b.Index(startIndex) != b.Index(endIndex) {
		b.del(startIndex, endIndex, true)
		b.mutex.Lock()
		b.redoStack.Init()
		b.mutex.Unlock()
	}
}

// DeleteNoUndo is a Delete that is not pushed to the undo stack.
func (b *Buffer) DeleteNoUndo(startIndex, endIndex string) {
	b.del(startIndex, endIndex, false)
}

func (b *Buffer) insert(index, s string, undo bool) {
	start := b.Index(index)

	b.mutex.Lock()

	// Find insert index
	i, line := 1, b.lines.Front()
	for i < start.Row && line.Next() != nil {
		line = line.Next()
		i++
	}

	// Insert lines
	startLine := line
	lines := strings.Split(s, "\n")
	for _, insertLine := range lines {
		line = b.lines.InsertAfter(insertLine, line)
	}

	// Update marks
	for _, pos := range b.marks {
		if pos.Row > start.Row {
			pos.Row += len(lines) - 1
		} else if pos.Row == start.Row && pos.Col >= start.Col {
			pos.Row += len(lines) - 1
			if len(lines) == 1 {
				pos.Col += len(s)
			} else {
				pos.Col += len(line.Value.(string)) - start.Col
			}
		}
	}

	// Splice initial line together with inserted lines
	line.Value = line.Value.(string) + startLine.Value.(string)[start.Col:]
	startLine.Value = startLine.Value.(string)[:start.Col] +
		b.lines.Remove(startLine.Next()).(string)

	b.mutex.Unlock()

	if undo {
		sp := start.String()
		end := b.Index(fmt.Sprintf("%s +%d", start.String(), len(s)))
		ep := end.String()
		b.mutex.Lock()
		front := b.undoStack.Front()
		collapsed := false
		if front != nil {
			switch v := front.Value.(type) {
			case insertOp:
				if v.ep == sp {
					front.Value = insertOp{v.sp, ep, v.s + s}
					collapsed = true
				} else if v.sp == sp {
					b.mutex.Unlock()
					end = b.Index(fmt.Sprintf("%s +%d", index, len(s+v.s)))
					b.mutex.Lock()
					ep = end.String()
					front.Value = insertOp{sp, ep, s + v.s}
					collapsed = true
				}
			}
		}
		if !collapsed {
			b.undoStack.PushFront(insertOp{sp, ep, s})
		}
		b.mutex.Unlock()
	}
}

// Insert inserts text at an index in b.
func (b *Buffer) Insert(index, s string) {
	if s != "" {
		b.insert(index, s, true)
		b.mutex.Lock()
		b.redoStack.Init()
		b.mutex.Unlock()
	}
}

// InsertNoUndo is an Insert that is not pushed to the undo stack.
func (b *Buffer) InsertNoUndo(index, s string) {
	b.insert(index, s, false)
}

// Replace replaces the text from start to end indices in b with string s.
func (b *Buffer) Replace(startIndex, endIndex, s string) {
	b.Delete(startIndex, endIndex)
	b.Insert(startIndex, s)
}

// MarkSet associates a name with an index into b. The name must not contain a
// space character.
func (b *Buffer) MarkSet(name, index string) {
	pos := b.Index(index)
	b.mutex.Lock()
	b.marks[name] = &pos
	b.mutex.Unlock()
}

// MarkUnset removes a mark from b.
func (b *Buffer) MarkUnset(name string) {
	b.mutex.Lock()
	delete(b.marks, name)
	b.mutex.Unlock()
}

// NumLines returns the number of lines of text in the buffer.
func (b *Buffer) NumLines() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.lines.Len()
}

// Undo undoes changes to the buffer until a separator is encountered or the
// undo stack is empty. Undone changes are pushed onto the redo stack. The
// given marks are placed at position of the redone operation.
func (b *Buffer) Undo(marks ...string) bool {
	i, loop := 0, true
	for loop {
		b.mutex.RLock()
		front := b.undoStack.Front()
		b.mutex.RUnlock()
		if front == nil {
			break
		}
		switch v := front.Value.(type) {
		case separator:
			if i != 0 {
				loop = false
			}
		case insertOp:
			b.DeleteNoUndo(v.sp, v.ep)
			pos := b.Index(v.sp)
			b.mutex.Lock()
			for _, k := range marks {
				b.marks[k] = &Position{pos.Row, pos.Col}
			}
			b.mutex.Unlock()
		case deleteOp:
			b.InsertNoUndo(v.sp, v.s)
			pos := b.Index(v.ep)
			b.mutex.Lock()
			for _, k := range marks {
				b.marks[k] = &Position{pos.Row, pos.Col}
			}
			b.mutex.Unlock()
		}
		if loop {
			b.mutex.Lock()
			b.redoStack.PushFront(b.undoStack.Remove(front))
			b.mutex.Unlock()
			i++
		}
	}
	return i > 0
}

// Redo redoes changes to the buffer until a separator is encountered or the
// undo stack is empty. Redone changes are pushed onto the undo stack. The
// given marks are placed at position of the redone operation.
func (b *Buffer) Redo(marks ...string) bool {
	i, loop, redone := 0, true, false
	for loop {
		b.mutex.RLock()
		front := b.redoStack.Front()
		b.mutex.RUnlock()
		if front == nil {
			break
		}
		switch v := front.Value.(type) {
		case separator:
			if i != 0 {
				loop = false
			}
		case insertOp:
			b.InsertNoUndo(v.sp, v.s)
			pos := b.Index(v.ep)
			b.mutex.Lock()
			for _, k := range marks {
				b.marks[k] = &Position{pos.Row, pos.Col}
			}
			b.mutex.Unlock()
			redone = true
		case deleteOp:
			b.DeleteNoUndo(v.sp, v.ep)
			pos := b.Index(v.sp)
			b.mutex.Lock()
			for _, k := range marks {
				b.marks[k] = &Position{pos.Row, pos.Col}
			}
			b.mutex.Unlock()
			redone = true
		}
		if loop {
			b.mutex.Lock()
			b.undoStack.PushFront(b.redoStack.Remove(front))
			b.mutex.Unlock()
			i++
		}
	}
	return redone
}

// Separator pushes an edit separator onto the undo stack if a separator is not
// already on top.
func (b *Buffer) Separator() {
	b.mutex.Lock()
	front := b.undoStack.Front()
	var sep separator
	if front != nil {
		switch front.Value.(type) {
		case separator:
			// Do nothing
		default:
			b.undoStack.PushFront(sep)
		}
	}
	b.mutex.Unlock()
}

// EditReset clears the buffer's undo and redo stacks.
func (b *Buffer) EditReset() {
	b.mutex.Lock()
	b.undoStack.Init()
	b.redoStack.Init()
	b.mutex.Unlock()
}
