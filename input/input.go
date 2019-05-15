package input

import (
	"bufio"
	"fmt"
	"github.com/awnumar/memguard"
	"os"
	"syscall"
)

// GetMaskedInput prompts the user for input and then reads a single newline-terminated line
// from stdin and returns it as a memguard.LockedBuffer with the terminating newline removed.
// User input is not displayed in the console.
//
// It also enforces the user to provide a limited number of characters bound by limitMax and limitMin.
// If an empty input is allowed (indicated by lmitMin = 0) GetInput can optionally return a default value.
//
// If standard input is not interactive (ex. redirected from another process) all prompts, limit checks
// and fallbacks are disabled and raw input is returned.
func GetMaskedInput(prompt, def, after string, limitMax, limitMin int) (b *memguard.LockedBuffer, err error) {
	attrs := syscall.ProcAttr{
		Dir:   "",
		Env:   []string{},
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
		Sys:   nil}
	var ws syscall.WaitStatus
	var stat os.FileInfo
	var pid int

	// Ugly hack to hide even uglier warning messages when receiving password input from redirected stdin.
	stat, err = os.Stdin.Stat()
	if err != nil {
		return
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		b, err = readInputBare()
		if err != nil {
			return
		}

		return memguard.Trim(b, 0, b.Size()-1)
	}

	pid, err = syscall.ForkExec(
		"/bin/stty",
		[]string{"stty", "-echo"},
		&attrs)
	if err != nil {
		return
	}

	_, err = syscall.Wait4(pid, &ws, 0, nil)
	if err != nil {
		return
	}

	b, err = GetInput(prompt, def, after+"\n", limitMax, limitMin)
	if err != nil {
		return
	}

	pid, err = syscall.ForkExec(
		"/bin/stty",
		[]string{"stty", "echo"},
		&attrs)
	if err != nil {
		return
	}

	_, err = syscall.Wait4(pid, &ws, 0, nil)

	return
}

// GetInput prompts the user for input and then reads a single newline-terminated line
// from stdin and returns it as a memguard.LockedBuffer with the terminating newline removed.
//
// It also enforces the user to provide a limited number of characters bound by limitMax and limitMin.
// If an empty input is allowed (indicated by lmitMin = 0) GetInput can optionally return a default value.
func GetInput(prompt, def, after string, limitMax, limitMin int) (b *memguard.LockedBuffer, err error) {
	if def != "" {
		prompt = fmt.Sprintf("%s (default is %s)", prompt, def)
	}

	b, err = readInput(prompt, after)
	if err != nil {
		return
	}

	for b.Size()-1 > limitMax || b.Size()-1 < limitMin {
		if b.Size()-1 > limitMax {
			fmt.Printf("Input is too long. Maximum of %d characters allowed.\n", limitMax)
		} else {
			fmt.Printf("Input must be at least %d characters.\n", limitMin)
		}
		b.Destroy()

		b, err = readInput(prompt, after)
		if err != nil {
			return
		}
	}

	if b.Size() == 1 && def != "" {
		return memguard.NewImmutableFromBytes([]byte(def))
	}
	if b.Size() > 1 {
		return memguard.Trim(b, 0, b.Size()-1)
	}

	// If empty imput is allowed and no default is passed, return a nil pointer since memguard can't create an empty buffer.
	return nil, nil
}

// readInput is the prompting and reading primitive
func readInput(prompt, after string) (b *memguard.LockedBuffer, err error) {
	fmt.Print(prompt + ": ")
	b, err = readInputBare()
	fmt.Print(after)
	return
}

// readInputBare can just read user input without any prompts/trailers
func readInputBare() (b *memguard.LockedBuffer, err error) {
	var in []byte
	reader := bufio.NewReader(os.Stdin)

	in, err = reader.ReadBytes('\n')
	if err != nil {
		return
	}

	return memguard.NewImmutableFromBytes(in)
}
