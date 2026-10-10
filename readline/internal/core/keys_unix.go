//go:build unix

package core

import (
	"bytes"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/chainreactors/tui/readline/internal/term"
	"golang.org/x/sys/unix"
)

const cursorPosTimeout = 200 * time.Millisecond

var errCursorPosTimeout = errors.New("cursor position read timeout")

// GetCursorPos returns the current cursor position in the terminal.
// It is safe to call this function even if the shell is reading input.
func (k *Keys) GetCursorPos() (x, y int) {
	disable := func() (int, int) { return -1, -1 }

	var cursor []byte
	var pending []byte
	defer func() {
		// A direction key or partial cursor reply is still keyboard input
		// when the query expires. Return it to the editor in arrival order.
		if len(pending) > 0 {
			k.mutex.Lock()
			k.buf = append(k.buf, pending...)
			k.mutex.Unlock()
		}
	}()

	deadline := time.Now().Add(cursorPosTimeout)

drain:
	for {
		select {
		case <-k.cursor:
		default:
			break drain
		}
	}

	// Echo the query and wait for the main key
	// reading routine to send us the response back.
	term.Print("\x1b[6n")
	// In order not to get stuck with an input that might be user-one
	// (like when the user typed before the shell is fully started, and yet not having
	// queried cursor yet), we keep reading from stdin until we find the cursor response.
	// Everything else is passed back as user input.
	for {
		switch {
		case k.waiting, k.reading:
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return -1, -1
			}

			select {
			case cursor = <-k.cursor:
			case <-time.After(remaining):
				return -1, -1
			}

			indices := rxRcvCursorPos.FindSubmatchIndex(cursor)
			if indices == nil {
				k.mutex.Lock()
				k.buf = append(k.buf, cursor...)
				k.mutex.Unlock()
				continue
			}

			y, err := strconv.Atoi(string(cursor[indices[2]:indices[3]]))
			if err != nil {
				return disable()
			}

			x, err = strconv.Atoi(string(cursor[indices[4]:indices[5]]))
			if err != nil {
				return disable()
			}

			return x, y

		default:
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return -1, -1
			}

			buf := make([]byte, keyScanBufSize)

			read, err := k.readInputWithTimeout(buf, remaining)
			pending = append(pending, buf[:read]...)
			if err != nil {
				if errors.Is(err, errCursorPosTimeout) {
					return -1, -1
				}
				return disable()
			}

			indices := rxRcvCursorPos.FindSubmatchIndex(pending)
			if indices != nil {
				prefix := pending[:indices[0]]
				suffix := pending[indices[1]:]
				if len(prefix) > 0 || len(suffix) > 0 {
					k.mutex.Lock()
					k.buf = append(k.buf, prefix...)
					k.buf = append(k.buf, suffix...)
					k.mutex.Unlock()
				}
				y, err := strconv.Atoi(string(pending[indices[2]:indices[3]]))
				if err != nil {
					return disable()
				}

				x, err = strconv.Atoi(string(pending[indices[4]:indices[5]]))
				if err != nil {
					return disable()
				}

				pending = nil
				return x, y
			}

			// No cursor response yet: flush anything that cannot be part of it.
			start := bytes.LastIndex(pending, []byte{0x1b, '['})
			switch {
			case start == -1:
				if len(pending) > 0 {
					k.mutex.Lock()
					k.buf = append(k.buf, pending...)
					k.mutex.Unlock()
					pending = nil
				}
			case start > 0:
				k.mutex.Lock()
				k.buf = append(k.buf, pending[:start]...)
				k.mutex.Unlock()
				pending = pending[start:]
			}
		}
	}
}

func (k *Keys) readInputFiltered() (keys []byte, err error) {
	// Start reading from os.Stdin in the background.
	// We will either read keys from user, or an EOF
	// send by ourselves, because we pause reading.
	buf := make([]byte, keyScanBufSize)

	read, err := k.readInputWithTimeout(buf, 0)
	if read == 0 && err != nil {
		return
	}

	// Always attempt to extract cursor position info.
	// If found, strip it and keep the remaining keys.
	cursor, keys := k.extractCursorPos(buf[:read])

	if len(cursor) > 0 {
		select {
		case k.cursor <- cursor:
		default:
		}
	}

	return keys, nil
}

func (k *Keys) readInputWithTimeout(buf []byte, timeout time.Duration) (int, error) {
	k.inputReadMu.Lock()
	defer k.inputReadMu.Unlock()
	reader := k.inputReader()
	file, ok := reader.(*os.File)
	if !ok {
		// Arbitrary streams cannot cancel Read. Keep one outstanding read
		// across query timeouts so the next editor read receives its bytes.
		if k.inputRead == nil {
			ch := make(chan inputReadResult, 1)
			k.inputRead = ch
			go func() {
				data := make([]byte, keyScanBufSize)
				n, err := reader.Read(data)
				ch <- inputReadResult{data: data[:n], err: err}
			}()
		}
		var expired <-chan time.Time
		var refresh <-chan struct{}
		if timeout > 0 {
			timer := time.NewTimer(timeout)
			defer timer.Stop()
			expired = timer.C
		} else {
			refresh = k.refresh
		}
		select {
		case result := <-k.inputRead:
			k.inputRead = nil
			return copy(buf, result.data), result.err
		case <-expired:
			return 0, errCursorPosTimeout
		case <-refresh:
			return 0, errInputRefresh
		}
	}
	if timeout <= 0 {
		return file.Read(buf)
	}

	fds := []unix.PollFd{{
		Fd:     int32(file.Fd()),
		Events: unix.POLLIN,
	}}

	deadline := time.Now().Add(timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, errCursorPosTimeout
		}

		timeoutMs := int(remaining / time.Millisecond)
		if timeoutMs <= 0 && remaining > 0 {
			timeoutMs = 1
		}

		n, err := unix.Poll(fds, timeoutMs)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return 0, err
		}
		if n == 0 {
			return 0, errCursorPosTimeout
		}

		revents := fds[0].Revents
		if revents&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
			return 0, errCursorPosTimeout
		}

		return file.Read(buf)
	}
}

// GetTerminalResize for Unix systems using SIGWINCH signal
func GetTerminalResize(keys *Keys) <-chan bool {
	resizeChan := make(chan bool, 1)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		for {
			<-sigChan
			isWaiting := keys.waiting

			if !isWaiting {
				select {
				case resizeChan <- true:
				default:
				}
			}
		}
	}()

	return resizeChan
}
