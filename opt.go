package command

import "github.com/spf13/afero"

// commSuppress1Flag suppresses column 1 (lines only in file1).
type commSuppress1Flag bool

const (
	CommSuppressColumn1   commSuppress1Flag = true
	CommNoSuppressColumn1 commSuppress1Flag = false
)

func (f commSuppress1Flag) Configure(flags *flags) { flags.suppress1 = f }

// commSuppress2Flag suppresses column 2 (lines only in file2).
type commSuppress2Flag bool

const (
	CommSuppressColumn2   commSuppress2Flag = true
	CommNoSuppressColumn2 commSuppress2Flag = false
)

func (f commSuppress2Flag) Configure(flags *flags) { flags.suppress2 = f }

// commSuppress3Flag suppresses column 3 (lines in both files).
type commSuppress3Flag bool

const (
	CommSuppressColumn3   commSuppress3Flag = true
	CommNoSuppressColumn3 commSuppress3Flag = false
)

func (f commSuppress3Flag) Configure(flags *flags) { flags.suppress3 = f }

// commFs injects the filesystem used to open File positionals, so tests can
// supply an in-memory filesystem. The zero value falls back to the OS.
type commFs struct{ afero.Fs }

// CommFs selects the filesystem comm uses to open File positional arguments.
func CommFs(fs afero.Fs) commFs { return commFs{fs} }

func (f commFs) Configure(flags *flags) { flags.fs = f }

// value returns the configured filesystem, defaulting to the OS filesystem.
func (f commFs) value() afero.Fs {
	if f.Fs == nil {
		return afero.NewOsFs()
	}
	return f.Fs
}

type flags struct {
	suppress1 commSuppress1Flag
	suppress2 commSuppress2Flag
	suppress3 commSuppress3Flag
	fs        commFs
}
