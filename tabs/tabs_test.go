package tabs

import "testing"

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

func TestExpand(t *testing.T) {
	strcmp(t, Expand("hello", 8), "hello")
	strcmp(t, Expand("hello\tworld", 8), "hello   world")
	strcmp(t, Expand("hello\t\tworld,  response\tnil", 8),
		"hello           world,  response        nil")
}

func TestColumns(t *testing.T) {
	intcmp(t, Columns("", 8), 0)
	intcmp(t, Columns("hello", 8), 5)
	intcmp(t, Columns("\thello", 8), 13)
	intcmp(t, Columns("hello\t", 8), 8)
}
