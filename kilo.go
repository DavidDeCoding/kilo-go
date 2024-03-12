package main

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/term" // https://pkg.go.dev/golang.org/x/term#section-readme
)

type EditorConfig struct {
	screenrows, screencols int
}

var editorConfig = EditorConfig{}
var byteBuffer = bytes.Buffer{}
var KILO_VERSION = "1.0.0"

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

	byteBuffer.Write([]byte("\x1b[H"))

	os.Stdout.Write(byteBuffer.Bytes())
}

func editorProcessKeyPress() {
	var ch []byte = make([]byte, 1)
	os.Stdin.Read(ch)

	if ch[0] == 'q' {
		os.Stdout.Write([]byte("\x1b[2J"))
		os.Stdout.Write([]byte("\x1b[H"))

		os.Exit(0)
	}
}

func initEditor() {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		die(err.Error())
	}
	editorConfig.screenrows = height
	editorConfig.screencols = width
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
