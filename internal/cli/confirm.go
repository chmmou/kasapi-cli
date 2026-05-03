package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Confirm prompts the user on out with prompt + " [y/N]: " and reads a
// reply from in. It returns true only on an explicit "y" or "yes"
// (case-insensitive); empty input or any other string returns false.
//
// Confirm is intended for destructive write operations (#13). When the
// global --yes flag is set, callers should skip the prompt entirely
// rather than feeding the answer through this helper.
func Confirm(in io.Reader, out io.Writer, prompt string) (bool, error) {
	if _, err := fmt.Fprintf(out, "%s [y/N]: ", prompt); err != nil {
		return false, err
	}
	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
