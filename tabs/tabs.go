// Package tabs implements functions for expanding tabs in text.
package tabs

var p []rune // Reuse the same array for efficiency's sake

// Expand replaces tabs in a string with spaces according to the given tab
// stop, and returns the resulting string.
func Expand(s string, tabStop int) string {
	if len(p) < tabStop*len(s) {
		p = make([]rune, tabStop*len(s))
	}
	col := 0

	for _, ch := range s {
		if ch == '\t' {
			p[col] = ' '
			col++
			for col%tabStop != 0 {
				p[col] = ' '
				col++
			}
		} else {
			p[col] = ch
			col++
		}
	}

	return string(p[:col])
}

// Columns returns the number of columns needed to display a string expanded
// according to the given tab stop.
func Columns(s string, tabStop int) int {
	col := 0
	for _, ch := range s {
		if ch == '\t' {
			col += tabStop - col%tabStop
		} else {
			col++
		}
	}
	return col
}
