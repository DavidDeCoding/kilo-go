package main

import (
	"fmt"
	"os"

	"golang.org/x/term" // https://pkg.go.dev/golang.org/x/term#section-readme
)

type EditorConfig struct {
	screenrows, screencols int
}

var editorConfig = EditorConfig{}

func editorDrawRows() {
	for rowNo := 0; rowNo < editorConfig.screenrows; rowNo++ {
		os.Stdout.Write([]byte("~"))

		if rowNo < editorConfig.screenrows-1 {
			os.Stdout.Write([]byte("\r\n"))
		}
	}
}

func editorRefreshScreen() {
	os.Stdout.Write([]byte("\x1b[2J"))
	os.Stdout.Write([]byte("\x1b[H"))

	editorDrawRows()

	os.Stdout.Write([]byte("\x1b[H"))
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
