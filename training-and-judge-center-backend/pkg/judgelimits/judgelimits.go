// Package judgelimits holds the size limits that have to agree across binaries.
// They cannot live next to the code that enforces them: the default checker
// ships as its own image built from cmd/compare, a main package nothing can
// import, so a limit left there could only be kept in step by hand.
package judgelimits

// MaxOutputBytes is what a run may write before OUTPUT_LIMIT_EXCEEDED. The
// in-container write limit sits one 512-byte block above it, so going over is
// visible in the file size whatever the runtime did with the signal.
const MaxOutputBytes = 8 << 20

// MaxTokenBytes bounds a single token in the default checker. It must stay
// STRICTLY above every file that checker reads — the contestant's output and
// the jury's answer — because bufio rejects a token of exactly its buffer cap.
const MaxTokenBytes = 16 << 20
