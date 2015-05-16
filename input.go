package main

import "github.com/nsf/termbox-go"

var keyMap = map[termbox.Key]string{
	termbox.KeyArrowDown:  "<Down>",
	termbox.KeyArrowLeft:  "<Left>",
	termbox.KeyArrowRight: "<Right>",
	termbox.KeyArrowUp:    "<Up>",
	termbox.KeyBackspace2: "<Backspace>",
	termbox.KeyDelete:     "<Delete>",
	termbox.KeyEnd:        "<End>",
	termbox.KeyF10:        "<F10>",
	termbox.KeyF11:        "<F11>",
	termbox.KeyF12:        "<F12>",
	termbox.KeyF1:         "<F1>",
	termbox.KeyF2:         "<F2>",
	termbox.KeyF3:         "<F3>",
	termbox.KeyF4:         "<F4>",
	termbox.KeyF5:         "<F5>",
	termbox.KeyF6:         "<F6>",
	termbox.KeyF7:         "<F7>",
	termbox.KeyF8:         "<F8>",
	termbox.KeyF9:         "<F9>",
	termbox.KeyHome:       "<Home>",
	termbox.KeyInsert:     "<Insert>",
	termbox.KeyPgdn:       "<PgDn>",
	termbox.KeyPgup:       "<PgUp>",
	termbox.KeySpace:      "<Space>",

	termbox.KeyCtrl6:          "<C-6>",
	termbox.KeyCtrlA:          "<C-a>",
	termbox.KeyCtrlB:          "<C-b>",
	termbox.KeyCtrlBackslash:  "<C-\\>",
	termbox.KeyCtrlC:          "<C-c>",
	termbox.KeyCtrlD:          "<C-d>",
	termbox.KeyCtrlE:          "<C-e>",
	termbox.KeyCtrlF:          "<C-f>",
	termbox.KeyCtrlG:          "<C-g>",
	termbox.KeyCtrlH:          "<C-h>",
	termbox.KeyCtrlI:          "<C-i>",
	termbox.KeyCtrlJ:          "<C-j>",
	termbox.KeyCtrlK:          "<C-k>",
	termbox.KeyCtrlL:          "<C-l>",
	termbox.KeyCtrlLsqBracket: "<C-[>",
	termbox.KeyCtrlM:          "<C-m>",
	termbox.KeyCtrlN:          "<C-n>",
	termbox.KeyCtrlO:          "<C-o>",
	termbox.KeyCtrlP:          "<C-p>",
	termbox.KeyCtrlQ:          "<C-q>",
	termbox.KeyCtrlR:          "<C-r>",
	termbox.KeyCtrlRsqBracket: "<C-]>",
	termbox.KeyCtrlS:          "<C-s>",
	termbox.KeyCtrlSlash:      "<C-/>",
	termbox.KeyCtrlT:          "<C-t>",
	termbox.KeyCtrlTilde:      "<C-~>",
	termbox.KeyCtrlU:          "<C-u>",
	termbox.KeyCtrlV:          "<C-v>",
	termbox.KeyCtrlW:          "<C-w>",
	termbox.KeyCtrlX:          "<C-x>",
	termbox.KeyCtrlY:          "<C-y>",
	termbox.KeyCtrlZ:          "<C-z>",
}

// Converts a key event to a string representation
func keyString(event termbox.Event) string {
	var s string
	if event.Key != 0 {
		s = keyMap[event.Key]
	} else if event.Ch >= 0x20 {
		if event.Mod == termbox.ModAlt {
			s = "<M-" + string(event.Ch) + ">"
		} else {
			s = string(event.Ch)
		}
	}
	return s
}
