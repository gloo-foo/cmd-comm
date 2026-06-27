package alias_test

import (
	"slices"
	"testing"

	comm "github.com/gloo-foo/cmd-comm/alias"
	"github.com/gloo-foo/testable"
)

// The alias package re-exports the constructor and flag constants under
// unprefixed names. A mis-wired re-export (say, SuppressColumn1 bound to the
// disabled constant, or Comm bound to the wrong function) compiles cleanly, so
// only behavior can prove the wiring. Each test exercises one re-export and
// asserts the exact GNU comm column/tab layout it must produce.
//
// input1 = apple banana cherry grape (from the stream); input2 = banana cherry
// kiwi (via the re-exported CommInput).

const commInput1 = "apple\nbanana\ncherry\ngrape\n"

func input2() comm.CommInput {
	return comm.CommInput{[]byte("banana"), []byte("cherry"), []byte("kiwi")}
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAlias_DefaultThreeColumns(t *testing.T) {
	lines, err := testable.TestLines(comm.Comm(input2()), commInput1)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "\t\tbanana", "\t\tcherry", "grape", "\tkiwi"})
}

func TestAlias_SuppressColumn1(t *testing.T) {
	lines, err := testable.TestLines(comm.Comm(input2(), comm.SuppressColumn1), commInput1)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"\tbanana", "\tcherry", "kiwi"})
}

func TestAlias_SuppressColumn2(t *testing.T) {
	lines, err := testable.TestLines(comm.Comm(input2(), comm.SuppressColumn2), commInput1)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "\tbanana", "\tcherry", "grape"})
}

func TestAlias_SuppressColumn3(t *testing.T) {
	lines, err := testable.TestLines(comm.Comm(input2(), comm.SuppressColumn3), commInput1)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "grape", "\tkiwi"})
}

func TestAlias_NoSuppressConstantsMatchDefault(t *testing.T) {
	// The No* constants are the disabled forms: they must behave exactly like
	// passing no suppression flag at all.
	lines, err := testable.TestLines(
		comm.Comm(input2(),
			comm.NoSuppressColumn1,
			comm.NoSuppressColumn2,
			comm.NoSuppressColumn3),
		commInput1,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"apple", "\t\tbanana", "\t\tcherry", "grape", "\tkiwi"})
}
