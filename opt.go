package command

import (
	"github.com/spf13/afero"
)

// commSuppress1Flag suppresses column 1 (lines only in file1).
type commSuppress1Flag bool

const (
	CommSuppressColumn1   commSuppress1Flag = true
	CommNoSuppressColumn1 commSuppress1Flag = false
)

// commSuppress2Flag suppresses column 2 (lines only in file2).
type commSuppress2Flag bool

const (
	CommSuppressColumn2   commSuppress2Flag = true
	CommNoSuppressColumn2 commSuppress2Flag = false
)

// commSuppress3Flag suppresses column 3 (lines in both files).
type commSuppress3Flag bool

const (
	CommSuppressColumn3   commSuppress3Flag = true
	CommNoSuppressColumn3 commSuppress3Flag = false
)

// CommFs selects the filesystem comm uses to open File positional arguments,
// so tests can supply an in-memory filesystem: CommFs(afero.NewMemMapFs()).
// When absent, comm falls back to the OS filesystem.
type CommFs afero.Fs

// fsOrOS returns the injected filesystem, defaulting to the OS filesystem.
func fsOrOS(fs CommFs) afero.Fs {
	if fs == nil {
		return afero.NewOsFs()
	}
	return fs
}

// flags is the option set folded from a Comm call's option values.
type flags struct {
	fs               CommFs
	suppress1Enabled commSuppress1Flag
	suppress2Enabled commSuppress2Flag
	suppress3Enabled commSuppress3Flag
}

// fold partitions opts: comm's own option values are folded into the flag set,
// and every other argument is passed through unchanged for the framework's
// positional classifier.
func fold(opts []any) (flags, []any) {
	var f flags
	rest := make([]any, 0, len(opts))
	for _, o := range opts {
		switch v := o.(type) {
		case commSuppress1Flag:
			f.suppress1Enabled = v
		case commSuppress2Flag:
			f.suppress2Enabled = v
		case commSuppress3Flag:
			f.suppress3Enabled = v
		case CommFs:
			f.fs = v
		default:
			rest = append(rest, o)
		}
	}
	return f, rest
}
