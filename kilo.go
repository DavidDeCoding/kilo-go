package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term" // https://pkg.go.dev/golang.org/x/term#section-readme
)

var KILO_VERSION = "1.0.0"

func CONTROL_KEY(key byte) int {
	return int(key & 0x1f)
}

type EditorFileBuffer struct {
	buffer []string
	render []string
}

func (e *EditorFileBuffer) updateRender() {
	e.render = make([]string, 0)
	for rowNo := 0; rowNo < len(e.buffer); rowNo++ {
		renderedLine := strings.Builder{}
		for colNo := 0; colNo < len(e.buffer[rowNo]); colNo++ {
			if e.buffer[rowNo][colNo] == '\t' {
				renderedLine.WriteByte(' ')
			} else {
				renderedLine.WriteByte(e.buffer[rowNo][colNo])
			}
		}
		e.render = append(e.render, renderedLine.String())
	}
}

func (e *EditorFileBuffer) insert(pos_x int, pos_y int, character byte) {
	if pos_x < 0 || pos_x > len(e.buffer[pos_y]) {
		pos_x = len(e.buffer[pos_y])
	}

	rowBuilder := strings.Builder{}
	rowBuilder.WriteString(e.buffer[pos_y][:pos_x])
	rowBuilder.WriteByte(character)
	rowBuilder.WriteString(e.buffer[pos_y][pos_x:])
	e.buffer[pos_y] = rowBuilder.String()

	e.updateRender()
}

func (e *EditorFileBuffer) del(pos_x int, pos_y int) {
	if pos_x < 0 || pos_x > len(e.buffer[pos_y]) {
		return
	}

	if pos_x > 0 {
		rowBuilder := strings.Builder{}
		rowBuilder.WriteString(e.buffer[pos_y][:pos_x-1])
		rowBuilder.WriteString(e.buffer[pos_y][pos_x:])
		e.buffer[pos_y] = rowBuilder.String()
	} else {
		rowBuilder := strings.Builder{}
		rowBuilder.WriteString(e.buffer[pos_y-1])
		rowBuilder.WriteString(e.buffer[pos_y])
		e.buffer[pos_y-1] = rowBuilder.String()
		e.buffer = append(e.buffer[:pos_y], e.buffer[pos_y+1:]...)
	}

	e.updateRender()
}

func (e *EditorFileBuffer) insertNewAt(pos_x int, pos_y int, line string) {
	e.buffer = append(e.buffer, "")
	for rowNo := len(e.buffer) - 1; rowNo > pos_y; rowNo-- {
		e.buffer[rowNo] = e.buffer[rowNo-1]
	}
	if pos_x == 0 {
		e.buffer[pos_y] = line
	} else {
		actualLine := e.buffer[pos_y]
		e.buffer[pos_y] = actualLine[:pos_x]
		e.buffer[pos_y+1] = actualLine[pos_x:]
	}

	e.updateRender()
}

func (e *EditorFileBuffer) append(line string) {
	e.insertNewAt(0, e.len(), line)
}

func (e *EditorFileBuffer) len() int {
	return len(e.buffer)
}

func (e *EditorFileBuffer) line(number int) string {
	if len(e.buffer) <= number {
		die(fmt.Sprintf("No line %d", number))
	}
	return e.render[number]
}

type EditorConfig struct {
	cursor_y, cursor_x     int
	screenrows, screencols int
	fileBuffer             *EditorFileBuffer
	fileName               string
	offsetrows, offsetcols int
	statusMsg              string
	lastStatusUpdate       time.Time
}

var editorConfig = EditorConfig{}
var byteBuffer = bytes.Buffer{}
var termimalState *term.State

func editorInsertChar(character int) {
	if editorConfig.cursor_y == editorConfig.fileBuffer.len() {
		editorConfig.fileBuffer.append("")
	}
	editorConfig.fileBuffer.insert(
		editorConfig.cursor_x,
		editorConfig.cursor_y,
		byte(character),
	)
	editorConfig.cursor_x += 1
}

func editorDelChar() {
	if editorConfig.cursor_y == editorConfig.fileBuffer.len() {
		return
	}
	if editorConfig.cursor_x == 0 && editorConfig.cursor_y == 0 {
		return
	}

	prevRowSize := len(editorConfig.fileBuffer.line(editorConfig.cursor_y - 1))

	editorConfig.fileBuffer.del(
		editorConfig.cursor_x,
		editorConfig.cursor_y,
	)

	if editorConfig.cursor_x > 0 {
		editorConfig.cursor_x -= 1
	} else {
		editorConfig.cursor_x = prevRowSize
		editorConfig.cursor_y -= 1
	}
}

func editorInsertNewLine() {

	pos_x := editorConfig.cursor_x
	pos_y := editorConfig.cursor_y

	editorConfig.cursor_x = 0
	editorConfig.cursor_y += 1

	editorConfig.fileBuffer.insertNewAt(
		pos_x,
		pos_y,
		"",
	)
}

func editorDrawRows() {
	for rowNo := 0; rowNo < editorConfig.screenrows; rowNo++ {
		offsetRowNo := rowNo + editorConfig.offsetrows
		if offsetRowNo >= editorConfig.fileBuffer.len() {
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
			line := editorConfig.fileBuffer.line(offsetRowNo)
			if len(line) > editorConfig.screencols {
				line = line[editorConfig.offsetcols : editorConfig.screencols-1+editorConfig.offsetcols]
			}
			byteBuffer.WriteString(line)
		}

		byteBuffer.WriteString("\x1b[K")

		byteBuffer.WriteString("\r\n")
	}
}

func editorDrawStatusBar() {
	byteBuffer.WriteString("\x1b[7m")
	fileName := "[No Name]"
	if editorConfig.fileName != "" {
		fileName = editorConfig.fileName
	}
	status := strings.Builder{}
	status.WriteString(fmt.Sprintf(
		"%.20s - %d lines",
		fileName,
		editorConfig.fileBuffer.len()))
	cursorStatus := fmt.Sprintf("%d/%d",
		editorConfig.cursor_y+1,
		editorConfig.fileBuffer.len())
	for status.Len() < editorConfig.screencols {
		if editorConfig.screencols-status.Len() == len(cursorStatus) {
			status.WriteString(cursorStatus)
		} else {
			status.WriteByte(' ')
		}

	}
	byteBuffer.WriteString(status.String())
	byteBuffer.WriteString("\x1b[m")
	byteBuffer.WriteString("\r\n")
}

func editorDrawMessageBar() {
	byteBuffer.WriteString("\x1b[K")
	msg := editorConfig.statusMsg
	if len(msg) > editorConfig.screencols {
		msg = msg[:editorConfig.screencols-1]
	}
	if msg != "" && time.Since(editorConfig.lastStatusUpdate).Seconds() < 5.0 {
		byteBuffer.WriteString(msg)
	}
}

func editorPrompt(prefix string) string {
	buffer := bytes.Buffer{}
	for {
		editorSetStatusMessage(prefix, buffer.String())
		editorRefreshScreen()

		ch := editorReadKey()
		switch ch {
		case '\x1b':
			editorSetStatusMessage("")
			return ""
		case '\r':
			editorSetStatusMessage("")
			return buffer.String()
		default:
			if ch < 128 {
				buffer.WriteByte(byte(ch))
			}
		}
	}
}

func editorSetStatusMessage(msgs ...string) {
	msgsBuilder := strings.Builder{}
	for _, msg := range msgs {
		msgsBuilder.WriteString(msg)
	}
	editorConfig.statusMsg = msgsBuilder.String()
	editorConfig.lastStatusUpdate = time.Now()
}

func editorRefreshScreen() {
	editorScroll()

	byteBuffer.WriteString("\x1b[?25l")
	byteBuffer.WriteString("\x1b[H")

	editorDrawRows()
	editorDrawStatusBar()
	editorDrawMessageBar()

	// byteBuffer.Write([]byte("\x1b[H"))
	byteBuffer.WriteString(
		fmt.Sprintf(
			"\x1b[%d;%dH",
			(editorConfig.cursor_y-editorConfig.offsetrows)+1,
			(editorConfig.cursor_x-editorConfig.offsetcols)+1,
		),
	)

	byteBuffer.WriteString("\x1b[?25h")

	os.Stdout.Write(byteBuffer.Bytes())
}

const (
	BACKSPACE  = iota + 127
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

func editorScroll() {
	if editorConfig.cursor_y < editorConfig.offsetrows {
		editorConfig.offsetrows = editorConfig.cursor_y
	}

	if editorConfig.cursor_y >= (editorConfig.screenrows + editorConfig.offsetrows) {
		editorConfig.offsetrows = editorConfig.cursor_y - editorConfig.screenrows + 1
	}

	if editorConfig.cursor_x < editorConfig.offsetcols {
		editorConfig.offsetcols = editorConfig.cursor_x
	}

	if editorConfig.cursor_x >= (editorConfig.screencols + editorConfig.offsetcols) {
		editorConfig.offsetcols = editorConfig.cursor_x - editorConfig.screencols + 1
	}
}

func editorMoveCursor(key int) {
	switch key {
	case ARROW_LEFT:
		if editorConfig.cursor_x != 0 {
			editorConfig.cursor_x -= 1
		}
	case ARROW_RIGHT:
		if editorConfig.cursor_y < editorConfig.fileBuffer.len() {
			if editorConfig.cursor_x < len(editorConfig.fileBuffer.line(editorConfig.cursor_y)) {
				editorConfig.cursor_x += 1
			}
		}

	case ARROW_UP:
		if editorConfig.cursor_y != 0 {
			editorConfig.cursor_y -= 1
			if editorConfig.cursor_x > len(editorConfig.fileBuffer.line(editorConfig.cursor_y)) {
				editorConfig.cursor_x = len(editorConfig.fileBuffer.line(editorConfig.cursor_y))
			}
		}
	case ARROW_DOWN:
		if editorConfig.cursor_y < editorConfig.fileBuffer.len() {
			editorConfig.cursor_y += 1
			if editorConfig.cursor_y < editorConfig.fileBuffer.len() &&
				editorConfig.cursor_x > len(editorConfig.fileBuffer.line(editorConfig.cursor_y)) {
				editorConfig.cursor_x = len(editorConfig.fileBuffer.line(editorConfig.cursor_y))
			} else {
				editorConfig.cursor_x = 0
			}
		}
	}
}

func editorProcessKeyPress() {
	ch := editorReadKey()

	switch ch {
	case '\r':
		editorInsertNewLine()

	case CONTROL_KEY('q'):
		os.Stdout.Write([]byte("\x1b[2J"))
		os.Stdout.Write([]byte("\x1b[H"))
		term.Restore(int(os.Stdout.Fd()), termimalState)

		os.Exit(0)

	case CONTROL_KEY('s'):
		editorSave()

	case CONTROL_KEY('f'):
		editorFind()

	case HOME_KEY:
		editorConfig.cursor_x = 0
	case END_KEY:
		if editorConfig.cursor_y < editorConfig.fileBuffer.len() {
			editorConfig.cursor_x = len(editorConfig.fileBuffer.line(editorConfig.cursor_y))
		}

	case BACKSPACE, CONTROL_KEY('h'), DEL_KEY:
		if ch == DEL_KEY {
			editorMoveCursor(ARROW_RIGHT)
		}
		editorDelChar()

	case PAGE_UP, PAGE_DOWN:
		if ch == PAGE_UP {
			editorConfig.cursor_y = editorConfig.offsetrows
		} else {
			editorConfig.cursor_y = editorConfig.offsetrows + editorConfig.screenrows - 1
			if editorConfig.cursor_y > editorConfig.fileBuffer.len() {
				editorConfig.cursor_y = editorConfig.fileBuffer.len()
			}
		}
		for times := editorConfig.screenrows; times > 0; times-- {
			if ch == PAGE_UP {
				editorMoveCursor(ARROW_UP)
			} else {
				editorMoveCursor(ARROW_DOWN)
			}
		}
	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		editorMoveCursor(ch)
	default:
		editorInsertChar(ch)
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

func editorSave() {
	if editorConfig.fileName == "" {
		editorConfig.fileName = editorPrompt("Save as: ")
	}

	file, err := os.OpenFile(editorConfig.fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		die(err.Error())
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range editorConfig.fileBuffer.buffer {
		w.WriteString(line)
		w.WriteByte('\n')
	}
	err = w.Flush()
	if err != nil {
		die(err.Error())
	}
}

func editorFind() {
	saved_cx := editorConfig.cursor_x
	saved_cy := editorConfig.cursor_y
	saved_offsetcols := editorConfig.offsetcols
	saved_offsetrows := editorConfig.offsetrows

	query := editorPrompt("Search: ")

	if query == "" {
		editorConfig.cursor_x = saved_cx
		editorConfig.cursor_y = saved_cy
		editorConfig.offsetcols = saved_offsetcols
		editorConfig.offsetrows = saved_offsetrows
		return
	}

	for idx := editorConfig.cursor_y; idx < editorConfig.fileBuffer.len(); idx++ {
		row := editorConfig.fileBuffer.line(idx)

		matchIdx := strings.Index(row, query)
		if matchIdx != -1 {
			editorConfig.cursor_x = matchIdx
			editorConfig.cursor_y = idx
			editorConfig.offsetrows = editorConfig.fileBuffer.len()
			break
		}
	}
}

func editorOpen(filepath string) {
	editorConfig.fileName = filepath
	_, err := os.Stat(filepath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			die(err.Error())
		}
		return
	}
	file, err := os.Open(filepath)
	if err != nil {
		die(err.Error())
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		editorConfig.fileBuffer.append(scanner.Text())
	}
}

func initEditor() {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		die(err.Error())
	}
	editorConfig.screenrows = height - 2
	editorConfig.screencols = width
	editorConfig.cursor_y = 0
	editorConfig.cursor_x = 0
	editorConfig.fileBuffer = &EditorFileBuffer{}
	editorConfig.offsetrows = 0
	editorConfig.offsetcols = 0
}

func enableRawMode() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	termimalState = oldState

	if err != nil {
		die(err.Error())
	}
}

func die(str string) {
	os.Stdout.Write([]byte("\x1b[2J"))
	os.Stdout.Write([]byte("\x1b[H"))
	term.Restore(int(os.Stdout.Fd()), termimalState)

	fmt.Println(str)
	os.Exit(1)
}

func main() {
	enableRawMode()

	initEditor()
	if len(os.Args) > 1 {
		editorOpen(os.Args[1])
	}

	editorSetStatusMessage("HELP: Ctrl-F = find | Ctrl-S = save | Ctrl-Q = quit")

	for {
		editorRefreshScreen()
		editorProcessKeyPress()
	}
}
