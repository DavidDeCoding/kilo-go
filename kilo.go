package main

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/term" // https://pkg.go.dev/golang.org/x/term#section-readme
)

var KILO_VERSION = "1.0.0"

type EditorConfig struct {
	cursor_y, cursor_x int
	screenrows, screencols int
}

var editorConfig = EditorConfig{}
var byteBuffer = bytes.Buffer{}

func editorDrawRows() {
	for rowNo := 0; rowNo < editorConfig.screenrows; rowNo++ {
		if rowNo == editorConfig.screenrows/3 {

			welcome := fmt.Sprintf("Kilo editor -- version %s", KILO_VERSION)

			padding := (editorConfig.screencols - len(welcome)) / 2
			if padding > 0 {
				byteBuffer.Write([]byte("~"))
			}
			for ; padding > 0; padding-- {
				byteBuffer.Write([]byte(" "))
			}

			byteBuffer.WriteString(welcome)

		} else {
			byteBuffer.Write([]byte("~"))
		}

		byteBuffer.Write([]byte("\x1b[K"))

		if rowNo < editorConfig.screenrows-1 {
			byteBuffer.Write([]byte("\r\n"))
		}
	}
}

func editorRefreshScreen() {
	byteBuffer.Write([]byte("\x1b[H"))

	editorDrawRows()

	// byteBuffer.Write([]byte("\x1b[H"))
	byteBuffer.WriteString(
		fmt.Sprintf(
			"\x1b[%d;%dH", 
			editorConfig.cursor_y + 1,
			editorConfig.cursor_x + 1,
		),
	)

	os.Stdout.Write(byteBuffer.Bytes())
}

const (
	ARROW_LEFT = iota + 1000
	ARROW_RIGHT
	ARROW_UP
	ARROW_DOWN
)

func editorMoveCursor(key int) {
	switch key {
	case ARROW_LEFT:
		if editorConfig.cursor_x != 0 {
			editorConfig.cursor_x -= 1
		}
	case ARROW_RIGHT:
		if editorConfig.cursor_x != editorConfig.screencols - 1 {
			editorConfig.cursor_x += 1
		}
	case ARROW_UP:
		if editorConfig.cursor_y != 0 {
			editorConfig.cursor_y -= 1
		}
	case ARROW_DOWN:
		if editorConfig.cursor_y != editorConfig.screenrows - 1 {
			editorConfig.cursor_y += 1
		}
	}
}

func editorProcessKeyPress() {
	ch := editorReadKey()

	switch ch {
	case 'q':
		os.Stdout.Write([]byte("\x1b[2J"))
		os.Stdout.Write([]byte("\x1b[H"))

		os.Exit(0)
	
	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		editorMoveCursor(ch)
	}
}

func editorReadKey() int {
	var ch []byte = make([]byte, 4)
	c, err := os.Stdin.Read(ch)
	if err != nil {
		die(err.Error())
	}
	
	if c == 1 {
		return int(ch[0])
	} else if c == 2 {
		return '\x1b'
	} else if ch[0] == '\x1b' {
		if ch[1] == '[' {
			switch ch[2] {
			case 'A':
				return ARROW_UP
			case 'B':
				return ARROW_DOWN
			case 'C':
				return ARROW_RIGHT
			case 'D':
				return ARROW_LEFT
			}
		}
	}
	
	return '\x1b'
}

func initEditor() {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		die(err.Error())
	}
	editorConfig.screenrows = height
	editorConfig.screencols = width
	editorConfig.cursor_y = 0
	editorConfig.cursor_x = 0
}

func enableRawMode() *term.State {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		die(err.Error())
	}

	return oldState
}

func die(str string) {
	os.Stdout.Write([]byte("\x1b[2J"))
	os.Stdout.Write([]byte("\x1b[H"))

	fmt.Println(str)
	os.Exit(1)
}

func main() {
	oldState := enableRawMode()
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	initEditor()

	for {
		editorRefreshScreen()
		editorProcessKeyPress()
	}
}
