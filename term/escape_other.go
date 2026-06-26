//go:build !unix

package term

import "time"

func ReadPendingTerminalBytes(_ time.Duration) string {
	return ""
}
