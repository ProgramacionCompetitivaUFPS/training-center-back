// Command compare is the default output checker: it decides whether a
// contestant's output matches the expected one by comparing whitespace-
// delimited tokens. It runs inside a sandbox container, so the outputs never
// have to travel through the worker just to be compared.
//
// It takes the same three file arguments as a custom checker (input, expected,
// contestant) even though the input is unused, so every checker — custom or
// default — is invoked exactly the same way.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// maxTokenBytes bounds a single token. strings.Fields, the tokenizer this
// replaces, had no such limit, so this is set well beyond anything a realistic
// output reaches; exceeding it is reported as a checker failure rather than
// silently as a wrong answer.
const maxTokenBytes = 16 << 20

const (
	exitAccepted = 0
	exitRejected = 1
	exitFailure  = 3
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	if len(args) != 3 {
		fmt.Fprintln(stderr, "usage: compare <input> <expected> <contestant>")
		return exitFailure
	}

	expected, closeExpected, err := openTokens(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "reading expected output: %v\n", err)
		return exitFailure
	}
	defer closeExpected()

	contestant, closeContestant, err := openTokens(args[2])
	if err != nil {
		fmt.Fprintf(stderr, "reading contestant output: %v\n", err)
		return exitFailure
	}
	defer closeContestant()

	for position := 1; ; position++ {
		// Both are advanced every iteration, deliberately without short-circuiting:
		// knowing whether the contestant still has tokens is what detects a count
		// mismatch, so this must not become a && .
		expOK, conOK := expected.Scan(), contestant.Scan()

		if err := expected.Err(); err != nil {
			fmt.Fprintf(stderr, "reading expected output: %v\n", err)
			return exitFailure
		}
		if err := contestant.Err(); err != nil {
			fmt.Fprintf(stderr, "reading contestant output: %v\n", err)
			return exitFailure
		}

		switch {
		case !expOK && !conOK:
			return exitAccepted
		case expOK != conOK:
			// Reports the position but never the token itself. CheckResult.Message
			// is discarded today, but surfacing a checker's message is the natural
			// thing to add later, and keeping values out means that can be done
			// without leaking the expected output.
			fmt.Fprintf(stderr, "token count mismatch at position %d\n", position)
			return exitRejected
		case expected.Text() != contestant.Text():
			fmt.Fprintf(stderr, "token mismatch at position %d\n", position)
			return exitRejected
		}
	}
}

// openTokens streams a file as whitespace-delimited tokens. bufio.ScanWords
// splits on the same character set as strings.Fields, so the verdict is
// identical to the in-process comparison it replaces — but memory stays
// constant instead of growing with the size of the output.
func openTokens(path string) (*bufio.Scanner, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxTokenBytes)
	scanner.Split(bufio.ScanWords)
	return scanner, func() { _ = f.Close() }, nil
}
