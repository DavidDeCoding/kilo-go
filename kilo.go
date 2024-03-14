package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"golang.org/x/term" // https://pkg.go.dev/golang.org/x/term#section-readme
)

var KILO_VERSION = "1.0.0"

func CONTROL_KEY(key byte) int {
	return int(key & 0x1f)
}

type EditorFileBuffer struct {
	buffer [][]byte
}

func (e *EditorFileBuffer) append(bytes []byte) {
	e.buffer = append(e.buffer, bytes)
}

func (e *EditorFileBuffer) len() int {
	return len(e.buffer)
}

type EditorConfig struct {
	cursor_y, cursor_x     int
	screenrows, screencols int
	fileBuffer             *EditorFileBuffer
}

var editorConfig = EditorConfig{}
var byteBuffer = bytes.Buffer{}

func editorDrawRows() {
	for rowNo := 0; rowNo < editorConfig.screenrows; rowNo++ {
		if rowNo >= editorConfig.fileBuffer.len() {
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
		} else {
			line := editorConfig.fileBuffer.buffer[0]
			if len(line) > editorConfig.screencols {
				line = line[:editorConfig.screencols]
			}
			byteBuffer.Write(line)
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
			editorConfig.cursor_y+1,
			editorConfig.cursor_x+1,
		),
	)

	os.Stdout.Write(byteBuffer.Bytes())
}

const (
	ARROW_LEFT = iota + 1000
	ARROW_RIGHT
	ARROW_UP
	ARROW_DOWN
	DEL_KEY
	HOME_KEY
	END_KEY
	PAGE_UP
	PAGE_DOWN
)

func editorMoveCursor(key int) {
	switch key {
	case ARROW_LEFT:
		if editorConfig.cursor_x != 0 {
			editorConfig.cursor_x -= 1
		}
	case ARROW_RIGHT:
		if editorConfig.cursor_x != editorConfig.screencols-1 {
			editorConfig.cursor_x += 1
		}
	case ARROW_UP:
		if editorConfig.cursor_y != 0 {
			editorConfig.cursor_y -= 1
		}
	case ARROW_DOWN:
		if editorConfig.cursor_y != editorConfig.screenrows-1 {
			editorConfig.cursor_y += 1
		}
	}
}

func editorProcessKeyPress() {
	ch := editorReadKey()

	switch ch {
	case CONTROL_KEY('q'):
		os.Stdout.Write([]byte("\x1b[2J"))
		os.Stdout.Write([]byte("\x1b[H"))

		os.Exit(0)

	case HOME_KEY:
		editorConfig.cursor_x = 0
	case END_KEY:
		editorConfig.cursor_x = editorConfig.screencols - 1
	case PAGE_UP, PAGE_DOWN:
		for times := editorConfig.screenrows; times > 0; times-- {
			if ch == PAGE_UP {
				editorMoveCursor(ARROW_UP)
			} else {
				editorMoveCursor(ARROW_DOWN)
			}
		}
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

	switch c {
	case 1:
		return int(ch[0])
	case 2:
		return '\x1b'
	case 3, 4:
		if ch[0] == '\x1b' {
			if ch[1] == '[' || ch[1] == 'O' {
				if ch[2] >= '0' && ch[2] <= '9' {
					if c <= 3 || ch[3] != '~' {
						return '\x1b'
					}

					switch ch[2] {
					case '1':
						return HOME_KEY
					case '3':
						return DEL_KEY
					case '4':
						return END_KEY
					case '5':
						return PAGE_UP
					case '6':
						return PAGE_DOWN
					case '7':
						return HOME_KEY
					case '8':
						return END_KEY
					}

				} else {

					switch ch[2] {
					case 'A':
						return ARROW_UP
					case 'B':
						return ARROW_DOWN
					case 'C':
						return ARROW_RIGHT
					case 'D':
						return ARROW_LEFT
					case 'H':
						return HOME_KEY
					case 'F':
						return END_KEY
					}

				}
			}
		}
	}
	return '\x1b'
}

func editorOpen(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		die(err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		editorConfig.fileBuffer.append(scanner.Bytes())
	}
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
	editorConfig.fileBuffer = &EditorFileBuffer{}
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
	if len(os.Args) > 1 {
		editorOpen(os.Args[1])
	}

	for {
		editorRefreshScreen()
		editorProcessKeyPress()
	}
}
