package main

const manualString = `Zygote manual. Press M-m to return.

Key bindings fall into two categories: commands and modes. A command performs
a single action, and has a binding that uses the Ctrl key (abbreviated as C-).
A mode toggles a flag in the state of the editor, and has a binding that uses
the Alt or Meta key (abbreviated as M-).

Registers are string variables identified by a single UTF-8 character. Some
registers identified by a capital letter have special meaning to Zygote:

  F  Filename/path
  L  Line number of cursor
  T  Tab width

Commands:

  C-c  Cancel prompt
  C-o  Open file
  C-p  Put from register
  C-q  Quit
  C-r  Redo
  C-s  Save file
  C-u  Undo
  C-w  Write into register
  C-y  Yank into register
  C-z  Suspend process

Modes:

  M-m  Manual
  M-s  Select
  M-v  View
  M-w  Word
`
