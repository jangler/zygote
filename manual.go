package main

const manualString = `Zygote manual, alpha version.

Zygote is a console text editor designed to be simple to learn, yet quick and
powerful enough to contend with the eldritch contraptions in common use. Three
core concepts of Zygote are commands, modes, and registers.


COMMANDS

Commands perform single actions, and have key bindings that use the Ctrl key
(abbreviated as C-). Some commands prompt for further input. Most of the
commands and modes that work in the main buffer also work in the prompt buffer.

  C-_  Undo change to buffer
  C-a  Start of line
  C-b  Backward search
  C-c  Cancel prompt
  C-e  End of line
  C-f  Forward search
  C-h  Delete character
  C-o  Open file
  C-p  Put from register
  C-q  Quit
  C-r  Redo change to buffer
  C-s  Save file
  C-t  Type into register
  C-u  Delete line
  C-w  Delete word
  C-x  Execute from register
  C-y  Yank into register
  C-z  Suspend process


MODES

Modes toggle flags in the state of the editor, and have key bindings that use
the Alt or Meta key (abbreviated as M-). Modes are non-exclusive; that is, any
number of modes can be active at once.

  M-m  Manual
  M-s  Select
  M-v  View
  M-w  Word


REGISTERS

Registers are string variables identified by a single UTF-8 character. Some
registers identified by capital letters have special meaning to Zygote.

  C  Column number of cursor
  D  Last deletion
  F  Filename/path
  L  Line number of cursor
  S  Last search string
  T  Tab width


CONFIGURATION

When Zygote starts, it reads from a configuration file, by default ~/.zygoterc,
and interprets each line as if it were executed by the C-x command; that is,
as a series of key inputs. Non-printable keys and chords can be represented by
forms such as <Enter>, <C-w>, and <M-w>. To have such a form interpreted as
literal text, prefix it with a backslash, as in \<C-q>.


CONTRIBUTING

If you would like to report a bug in Zygote, make a suggestion, or contribute
to development, please do so via https://github.com/jangler/zygote, or send an
email to brandon at jangler dot info.
`
