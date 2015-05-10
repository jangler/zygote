package text

import (
	"testing"
)

func poscmp(t *testing.T, got Position, wantRow, wantCol int) {
	if wantRow != got.Row || wantCol != got.Col {
		t.Errorf("got %d.%d, want %d.%d", got.Row, got.Col, wantRow, wantCol)
	}
}

func strcmp(t *testing.T, got, want string) {
	if want != got {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func intcmp(t *testing.T, got, want int) {
	if want != got {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestNew(t *testing.T) {
	if b := NewBuffer(); b == nil {
		t.FailNow()
	}
}

func TestParse(t *testing.T) {
	b := NewBuffer()
	strings := []string{"bad", "1.bad"}
	for _, pos := range strings {
		func() {
			defer func() {
				if err := recover(); err == nil {
					t.Error("Bad position did not cause panic")
				}
			}()
			b.Get("1.0", pos)
		}()
	}
}

func TestIndex(t *testing.T) {
	b := NewBuffer()
	b.Insert("1.0", "hello\nworld")
	poscmp(t, b.Index("0.0"), 1, 0)
	poscmp(t, b.Index("1.3"), 1, 3)
	poscmp(t, b.Index("1.9"), 1, 5)
	poscmp(t, b.Index("5.0"), 2, 5)
	poscmp(t, b.Index("1.0 -5"), 1, 0)
	poscmp(t, b.Index("1.0 +3"), 1, 3)
	poscmp(t, b.Index("1.0 +6"), 2, 0)
	poscmp(t, b.Index("2.0 -1"), 1, 5)
	poscmp(t, b.Index("2.0 +9"), 2, 5)
}

func TestGet(t *testing.T) {
	b := NewBuffer()
	b.Insert("1.0", "hello")
	strcmp(t, b.Get("1.1", "1.1").String(), "")
	strcmp(t, b.Get("1.1", "1.4").String(), "ell")
	strcmp(t, b.Get("1.1", "1.end").String(), "ello")
	strcmp(t, b.Get("1.0", "end").String(), "hello")
	b.Insert("end", "\nworld")
	strcmp(t, b.Get("2.0", "end").String(), "world")
}

func TestInsert(t *testing.T) {
	b := NewBuffer()
	b.Insert("1.0", "")
	strcmp(t, b.Get("1.0", "end").String(), "")
	b.Insert("1.0", "alpha")
	strcmp(t, b.Get("1.0", "end").String(), "alpha")
	b.Insert("1.0", "beta ")
	strcmp(t, b.Get("1.0", "end").String(), "beta alpha")
	b.Insert("1.5", "gamma ")
	strcmp(t, b.Get("1.0", "end").String(), "beta gamma alpha")
	b.Insert("2.0", " delta")
	strcmp(t, b.Get("1.0", "end").String(), "beta gamma alpha delta")

	b = NewBuffer()
	b.Insert("1.0", "alpha\nbeta gamma\ndelta")
	strcmp(t, b.Get("1.0", "end").String(), "alpha\nbeta gamma\ndelta")
	b.Insert("2.5", "epsilon\nzeta ")
	strcmp(t, b.Get("1.0", "end").String(),
		"alpha\nbeta epsilon\nzeta gamma\ndelta")
	b.Insert("2.5", "eta ")
	strcmp(t, b.Get("1.0", "end").String(),
		"alpha\nbeta eta epsilon\nzeta gamma\ndelta")
}

func TestDelete(t *testing.T) {
	b := NewBuffer()
	b.Insert("1.0", "chased")
	b.Delete("1.2", "1.2")
	strcmp(t, b.Get("1.0", "end").String(), "chased")
	b.Delete("1.3", "1.5")
	strcmp(t, b.Get("1.0", "end").String(), "chad")
	b.Delete("1.0", "end")
	strcmp(t, b.Get("1.0", "end").String(), "")
	b.Insert("1.0", "alpha\nbeta\ngamma\ndelta")
	b.Delete("2.3", "4.3")
	strcmp(t, b.Get("1.0", "end").String(), "alpha\nbetta")
}

func TestReplace(t *testing.T) {
	b := NewBuffer()
	b.Replace("1.0", "1.0", "hello")
	strcmp(t, b.Get("1.0", "end").String(), "hello")
	b.Replace("1.1", "1.4", "ipp")
	strcmp(t, b.Get("1.0", "end").String(), "hippo")
	b.Replace("1.4", "1.5", "o\npotamus")
	strcmp(t, b.Get("1.0", "end").String(), "hippo\npotamus")
	b.Replace("1.1", "2.6", "and")
	strcmp(t, b.Get("1.0", "end").String(), "hands")
}

func TestMarkSet(t *testing.T) {
	b := NewBuffer()
	b.Insert("end", "hello")
	b.MarkSet("1", "1.1")
	b.MarkSet("2", "1.4")
	strcmp(t, b.Get("1", "2").String(), "ell")

	b.Insert("1.0", "\n")
	strcmp(t, b.Get("1", "2").String(), "ell")
	b.Insert("1.0", "\n")
	strcmp(t, b.Get("1", "2").String(), "ell")
	b.Insert("3.2", "y he")
	strcmp(t, b.Get("1", "2").String(), "ey hell")
	b.Insert("3.4", "and\n")
	strcmp(t, b.Get("1", "2").String(), "ey and\nhell")

	b.Delete("1.0", "2.0")
	strcmp(t, b.Get("1", "2").String(), "ey and\nhell")
	b.Delete("1", "3.1")
	strcmp(t, b.Get("1", "2").String(), "ell")
	b.Delete("1", "2")
	strcmp(t, b.Get("1", "2").String(), "")
	b.Delete("1.0", "end")
	strcmp(t, b.Get("1", "2").String(), "")
}

func TestMarkUnset(t *testing.T) {
	b := NewBuffer()
	b.MarkSet("1", "1.0")
	b.MarkUnset("1")
	defer func() {
		if err := recover(); err == nil {
			t.Error("MarkUnset did not remove mark")
		}
	}()
	b.Get("1", "1")
}

func TestNumLines(t *testing.T) {
	b := NewBuffer()
	intcmp(t, b.NumLines(), 1)
	b.Insert("1.0", "hello\nworld\n")
	intcmp(t, b.NumLines(), 3)
}

func TestUndo(t *testing.T) {
	b := NewBuffer()
	b.MarkSet("mark", "end")
	b.Separator()
	if b.Undo() {
		t.Error("Undo returned true for new buffer")
	}
	if b.Redo() {
		t.Error("Redo returned true for new buffer")
	}
	b.Insert("1.0", "hello")
	if !b.Undo() {
		t.Error("Undo returned false for non-empty stack")
	}
	if b.Undo() {
		t.Error("Undo returned true for empty stack")
	}
	strcmp(t, b.Get("1.0", "end").String(), "")
	if !b.Redo() {
		t.Error("Redo returned false for non-empty stack")
	}
	if b.Redo() {
		t.Error("Redo returned true for empty stack")
	}
	strcmp(t, b.Get("1.0", "end").String(), "hello")
	b.Undo()
	strcmp(t, b.Get("1.0", "end").String(), "")

	b.Insert("1.0", "there")
	if b.Redo() {
		t.Error("Redo returned true after edit operation")
	}
	b.Insert("1.0", "hello ")
	b.Insert("end", " world")
	b.Undo("mark")
	strcmp(t, b.Get("1.0", "end").String(), "")
	b.Redo("mark")
	strcmp(t, b.Get("1.0", "end").String(), "hello there world")
	b.Separator()
	b.Separator()
	b.Delete("1.8", "1.10")
	b.Delete("1.8", "1.10")
	b.Delete("1.4", "1.8")
	b.Undo("mark")
	strcmp(t, b.Get("1.0", "end").String(), "hello there world")
	b.Redo("mark")
	strcmp(t, b.Get("1.0", "end").String(), "hellworld")
	b.Undo()
	b.Undo()
	strcmp(t, b.Get("1.0", "end").String(), "")
	b.Redo()
	strcmp(t, b.Get("1.0", "end").String(), "hello there world")
	b.Redo()
	strcmp(t, b.Get("1.0", "end").String(), "hellworld")

	b.EditReset()
	if b.Undo() {
		t.Error("Undo returned true after buffer reset")
	}
	if b.Redo() {
		t.Error("Redo returned true after buffer reset")
	}
}
