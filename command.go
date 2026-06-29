package command

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/destel/rill"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
)

// CommInput supplies the second input as raw lines, taking precedence over any
// second positional file path.
type CommInput [][]byte

// lines is one decoded input: the sorted set of lines comm compares.
type lines [][]byte

// Comm compares two sorted line streams and emits three columns:
//   - Column 1 (no indent): lines only in input1.
//   - Column 2 (one tab):   lines only in input2.
//   - Column 3 (two tabs):  lines in both.
//
// Opts:
//   - 1st positional file/Reader: input1 (overrides the upstream stream).
//   - 2nd positional file/Reader: input2.
//   - CommInput: input2 as raw lines (highest precedence for input2).
//   - CommSuppressColumn1 / 2 / 3: hide that column from output.
//   - CommFs: filesystem used to open File positionals (defaults to the OS).
//
// Suppression also collapses leading-tab indentation: when column 1 is
// suppressed, columns 2 and 3 lose their leading tab; when column 2 is also
// suppressed, column 3 loses its second tab. This matches GNU comm.
func Comm(opts ...any) gloo.Command[[]byte, []byte] {
	params := gloo.NewParameters[gloo.File, flags](opts...)
	f := params.Flags
	src := newSources(opts, params.Positional, f.fs.value())
	return gloo.FuncCommand[[]byte, []byte](func(ctx context.Context, in gloo.Stream[[]byte]) gloo.Stream[[]byte] {
		return gloo.GenerateFrom(ctx, in, func(_ context.Context, send func([]byte) bool, sendErr func(error)) {
			run(send, sendErr, src, in, f)
		})
	})
}

// run loads both inputs and emits the merged columns, forwarding any load error.
func run(send func([]byte) bool, sendErr func(error), src sources, in gloo.Stream[[]byte], f flags) {
	input1, input2, err := src.load(in)
	if err != nil {
		sendErr(err)
		return
	}
	columnsOf(send, f).merge(input1, input2)
}

// sources resolves the two comm inputs from opts, positionals, and the upstream
// stream. It is an immutable value built once per Comm call.
type sources struct {
	fs             afero.Fs
	positionals    []any
	explicitInput2 lines
	hasExplicit2   bool
}

// newSources classifies the opts into the resolved input sources.
func newSources(opts, positionals []any, fs afero.Fs) sources {
	explicit, ok := explicitInput2(opts)
	return sources{
		fs:             fs,
		positionals:    positionals,
		explicitInput2: explicit,
		hasExplicit2:   ok,
	}
}

// explicitInput2 returns the first CommInput option, if any.
func explicitInput2(opts []any) (lines, bool) {
	for _, o := range opts {
		if v, ok := o.(CommInput); ok {
			return lines(v), true
		}
	}
	return nil, false
}

// load resolves input1 then input2.
func (s sources) load(in gloo.Stream[[]byte]) (lines, lines, error) {
	input1, err := s.loadInput1(in)
	if err != nil {
		return nil, nil, err
	}
	input2, err := s.loadInput2()
	if err != nil {
		return nil, nil, err
	}
	return input1, input2, nil
}

// loadInput1 reads the first positional, falling back to the upstream stream.
func (s sources) loadInput1(in gloo.Stream[[]byte]) (lines, error) {
	if len(s.positionals) >= 1 {
		return s.readPositional(s.positionals[0])
	}
	got, err := rill.ToSlice(in.Chan())
	return lines(got), err
}

// loadInput2 prefers an explicit CommInput, else the second positional, else
// nothing.
func (s sources) loadInput2() (lines, error) {
	switch {
	case s.hasExplicit2:
		return s.explicitInput2, nil
	case len(s.positionals) >= 2:
		return s.readPositional(s.positionals[1])
	default:
		return nil, nil
	}
}

// readPositional decodes one positional argument into lines. The framework
// guarantees every positional is a gloo.File path or an io.Reader (see
// gloo.NewParameters), so those two cases are exhaustive.
func (s sources) readPositional(positional any) (lines, error) {
	if name, ok := positional.(gloo.File); ok {
		return s.readFile(name)
	}
	return scanLines(positional.(io.Reader))
}

// readFile opens a File positional on the injected filesystem and scans it.
func (s sources) readFile(name gloo.File) (out lines, err error) {
	f, err := s.fs.Open(string(name))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	return scanLines(f)
}

// scanLines reads r into a slice of independently-owned line copies.
func scanLines(r io.Reader) (lines, error) {
	scanner := bufio.NewScanner(r)
	var out lines
	for scanner.Scan() {
		out = append(out, bytes.Clone(scanner.Bytes()))
	}
	return out, scanner.Err()
}

// columns renders the three comm output columns through send, honoring
// suppression and the GNU indentation-collapse rule.
type columns struct {
	send       func([]byte) bool
	col2Prefix []byte
	col3Prefix []byte
	emit1      bool
	emit2      bool
	emit3      bool
}

// columnsOf builds the column renderer from the suppression flags.
func columnsOf(send func([]byte) bool, f flags) columns {
	emit1 := !bool(f.suppress1)
	emit2 := !bool(f.suppress2)
	return columns{
		send:       send,
		emit1:      emit1,
		emit2:      emit2,
		emit3:      !bool(f.suppress3),
		col2Prefix: indent(emit1),
		col3Prefix: append(indent(emit1), indent(emit2)...),
	}
}

// merge walks both sorted inputs, emitting each line in its column.
func (c columns) merge(input1, input2 lines) {
	i, j := 0, 0
	for i < len(input1) && j < len(input2) {
		i, j = c.step(input1, input2, i, j)
	}
	c.drain(input1[i:], c.emit1, nil)
	c.drain(input2[j:], c.emit2, c.col2Prefix)
}

// step compares the lines at i and j, emits the appropriate column, and returns
// the advanced indices.
func (c columns) step(input1, input2 lines, i, j int) (int, int) {
	switch cmp := bytes.Compare(input1[i], input2[j]); {
	case cmp < 0:
		c.put(c.emit1, nil, input1[i])
		return i + 1, j
	case cmp > 0:
		c.put(c.emit2, c.col2Prefix, input2[j])
		return i, j + 1
	default:
		c.put(c.emit3, c.col3Prefix, input1[i])
		return i + 1, j + 1
	}
}

// drain emits any remaining lines of one input under a fixed column.
func (c columns) drain(rest lines, emit bool, prefix []byte) {
	for _, line := range rest {
		c.put(emit, prefix, line)
	}
}

// put emits one prefixed line when its column is enabled.
func (c columns) put(emit bool, prefix, line []byte) {
	if emit {
		c.send(append(bytes.Clone(prefix), line...))
	}
}

// indent returns a single tab when the preceding column is present, else empty.
func indent(present bool) []byte {
	if present {
		return []byte{'\t'}
	}
	return nil
}
