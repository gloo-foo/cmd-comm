package command_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-comm"
	gloo "github.com/gloo-foo/framework"
	"github.com/gloo-foo/testable"
)

// comm produces three columns from two SORTED inputs:
//   - column 1 (no tab):    lines only in input1
//   - column 2 (one tab):   lines only in input2
//   - column 3 (two tabs):  lines in both
//
// The -1/-2/-3 flags suppress a column and collapse the leading tab of every
// later column. These tests assert the exact column/tab layout, not just counts.

func input2(ss ...string) command.CommInput {
	in := make(command.CommInput, len(ss))
	for i, s := range ss {
		in[i] = []byte(s)
	}
	return in
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestComm_ThreeColumnLayout(t *testing.T) {
	// input1: apple banana cherry grape ; input2: banana cherry kiwi.
	lines, err := testable.TestLines(
		command.Comm(input2("banana", "cherry", "kiwi")),
		"apple\nbanana\ncherry\ngrape\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{
		"apple",      // only in input1, column 1
		"\t\tbanana", // in both, column 3
		"\t\tcherry", // in both, column 3
		"grape",      // only in input1, column 1
		"\tkiwi",     // only in input2, column 2
	})
}

func TestComm_SuppressColumn1(t *testing.T) {
	// -1: column 1 gone; columns 2 and 3 lose their leading tab.
	lines, err := testable.TestLines(
		command.Comm(input2("banana", "cherry", "kiwi"), command.CommSuppressColumn1),
		"apple\nbanana\ncherry\ngrape\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{
		"\tbanana", // column 3 keeps one tab (column 2's slot)
		"\tcherry",
		"kiwi", // column 2 now at the left margin
	})
}

func TestComm_SuppressColumn2(t *testing.T) {
	// -2: column 2 gone; column 3 loses the second tab.
	lines, err := testable.TestLines(
		command.Comm(input2("banana", "cherry", "kiwi"), command.CommSuppressColumn2),
		"apple\nbanana\ncherry\ngrape\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{
		"apple",
		"\tbanana", // column 3 keeps column 1's tab only
		"\tcherry",
		"grape",
	})
}

func TestComm_SuppressColumn3(t *testing.T) {
	// -3: common lines gone; only the unique columns remain.
	lines, err := testable.TestLines(
		command.Comm(input2("banana", "cherry", "kiwi"), command.CommSuppressColumn3),
		"apple\nbanana\ncherry\ngrape\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{
		"apple",
		"grape",
		"\tkiwi",
	})
}

func TestComm_CommonOnly(t *testing.T) {
	// -1 -2: only the common column, with no indentation at all.
	lines, err := testable.TestLines(
		command.Comm(input2("banana", "cherry", "kiwi"),
			command.CommSuppressColumn1, command.CommSuppressColumn2),
		"apple\nbanana\ncherry\ngrape\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"banana", "cherry"})
}

func TestComm_NoSuppressConstantsMatchDefault(t *testing.T) {
	// The No* constants are the disabled forms and must behave like no flag.
	lines, err := testable.TestLines(
		command.Comm(input2("banana"),
			command.CommNoSuppressColumn1,
			command.CommNoSuppressColumn2,
			command.CommNoSuppressColumn3),
		"apple\nbanana\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "\t\tbanana"})
}

func TestComm_Input1TailDrains(t *testing.T) {
	// input1 longer than input2: the leftover column-1 lines must all emit.
	lines, err := testable.TestLines(
		command.Comm(input2("a")),
		"a\nb\nc\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"\t\ta", "b", "c"})
}

func TestComm_Input2TailDrains(t *testing.T) {
	// input2 longer than input1: the leftover column-2 lines must all emit.
	lines, err := testable.TestLines(
		command.Comm(input2("a", "b", "c")),
		"a\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"\t\ta", "\tb", "\tc"})
}

func TestComm_EmptyInputs(t *testing.T) {
	lines, err := testable.TestLines(command.Comm(command.CommInput{}), "")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{})
}

func TestComm_NoSecondInput(t *testing.T) {
	// With neither a CommInput nor a second positional, input2 is empty: every
	// input1 line is unique and lands in column 1.
	lines, err := testable.TestLines(
		command.Comm(strings.NewReader("apple\nbanana\n")),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "banana"})
}

func TestComm_PositionalFilesViaMemFs(t *testing.T) {
	// Both inputs as File positionals, read through an injected filesystem.
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "a.txt", []byte("apple\nbanana\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, "b.txt", []byte("banana\nkiwi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := testable.TestLines(
		command.Comm(gloo.File("a.txt"), gloo.File("b.txt"), command.CommFs(fs)),
		"ignored upstream\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "\t\tbanana", "\tkiwi"})
}

func TestComm_ReaderPositionals(t *testing.T) {
	// Both inputs as io.Reader positionals.
	lines, err := testable.TestLines(
		command.Comm(strings.NewReader("apple\nbanana\n"), strings.NewReader("banana\nkiwi\n")),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "\t\tbanana", "\tkiwi"})
}

func TestComm_FileNotFoundPropagates(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := testable.TestLines(
		command.Comm(gloo.File("missing.txt"), command.CommFs(fs)),
		"",
	)
	if err == nil {
		t.Fatal("expected an error opening a missing file, got nil")
	}
}

func TestComm_Input2FileNotFoundPropagates(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "a.txt", []byte("apple\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := testable.TestLines(
		command.Comm(gloo.File("a.txt"), gloo.File("missing.txt"), command.CommFs(fs)),
		"",
	)
	if err == nil {
		t.Fatal("expected an error opening the missing second file, got nil")
	}
}

func TestComm_ScannerErrorPropagates(t *testing.T) {
	// A reader that fails mid-scan must surface its error, not be swallowed.
	_, err := testable.TestLines(
		command.Comm(errReader{}),
		"",
	)
	if !errors.Is(err, errBoom) {
		t.Fatalf("got %v, want %v", err, errBoom)
	}
}

func TestComm_DefaultFsResolves(t *testing.T) {
	// With no CommFs option, comm opens File positionals on the OS filesystem.
	lines, err := testable.TestLines(
		command.Comm(gloo.File("testdata/file1.txt"), gloo.File("testdata/file2.txt")),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 {
		t.Fatal("expected output from on-disk testdata files")
	}
}

var errBoom = errors.New("boom")

// errReader is an io.Reader that always fails, exercising the scanner error path.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errBoom }
