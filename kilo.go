package main

import (
	"fmt"
	"os"

	// "github.com/pkg/term/termios"
	"golang.org/x/term" // https://pkg.go.dev/golang.org/x/term#section-readme
)

// var orig_termios = unix.Termios{}

func die(str string) {
	fmt.Println(str)
	os.Exit(1)
}

// func disableRawMode() {
// 	err := termios.Tcsetattr(uintptr(os.Stdin.Fd()), termios.TCSAFLUSH, &orig_termios)
// 	if err != nil {
// 		die(err.Error())
// 	}
// }

// func enableRawMode() {
// 	err := termios.Tcgetattr(uintptr(os.Stdin.Fd()), &orig_termios)
// 	if err != nil {
// 		die(err.Error())
// 	}

// 	raw := unix.Termios{}
// 	termios.Cfmakeraw(&raw)

// 	err = termios.Tcsetattr(uintptr(os.Stdin.Fd()), termios.TCSAFLUSH, &raw)
// 	if err != nil {
// 		die(err.Error())
// 	}
// }

func main() {
	// enableRawMode()
	// defer disableRawMode()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		die(err.Error())
	}

	defer term.Restore(int(os.Stdin.Fd()), oldState)

	for {
		var ch []byte = make([]byte, 1)
		os.Stdin.Read(ch)

		if ch[0] == 'q' {
			break
		}
	}
}
