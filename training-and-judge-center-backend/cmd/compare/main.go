// Command compare is the default output checker: it decides whether a
// contestant's output matches the expected one by comparing whitespace-
// delimited tokens. It runs inside a sandbox container, so the outputs never
// have to travel through the worker just to be compared.
//
// It takes the same three file arguments as a custom checker, in testlib's
// order — the input, the contestant's output, the jury's answer — so every
// checker, custom or default, is invoked exactly the same way. The input is
// never read.
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
		fmt.Fprintln(stderr, "usage: compare <input> <output> <answer>")
		return exitFailure
	}

	contestant, closeContestant, err := openTokens(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "reading the contestant's output: %v\n", err)
		return exitFailure
	}
	defer closeContestant()

	answer, closeAnswer, err := openTokens(args[2])
	if err != nil {
		fmt.Fprintf(stderr, "reading the jury's answer: %v\n", err)
		return exitFailure
	}
	defer closeAnswer()

	for position := 1; ; position++ {
		// Both are advanced every iteration, deliberately without short-circuiting:
		// knowing whether the contestant still has tokens is what detects a count
		// mismatch, so this must not become a && .
		conOK, ansOK := contestant.Scan(), answer.Scan()

		if err := contestant.Err(); err != nil {
			fmt.Fprintf(stderr, "reading the contestant's output: %v\n", err)
			return exitFailure
		}
		if err := answer.Err(); err != nil {
			fmt.Fprintf(stderr, "reading the jury's answer: %v\n", err)
			return exitFailure
		}

		switch {
		case !conOK && !ansOK:
			return exitAccepted
		case conOK != ansOK:
			// Reports the position but never the token itself. CheckResult.Message
			// is discarded today, but surfacing a checker's message is the natural
			// thing to add later, and keeping values out means that can be done
			// without leaking the jury's answer.
			fmt.Fprintf(stderr, "token count mismatch at position %d\n", position)
			return exitRejected
		case contestant.Text() != answer.Text():
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
