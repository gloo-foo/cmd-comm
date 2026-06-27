// Package alias provides unprefixed type aliases for comm command flags.
//
//	import comm "github.com/gloo-foo/cmd-comm/alias"
//	comm.Comm(input, comm.SuppressColumn1)
package alias

import command "github.com/gloo-foo/cmd-comm"

// Comm re-exports the constructor.
var Comm = command.Comm

// CommInput re-exports the second-input type.
type CommInput = command.CommInput

// CommFs re-exports the filesystem-injection option.
var CommFs = command.CommFs

// -1 flag: suppress column 1 (lines only in file1)
const SuppressColumn1 = command.CommSuppressColumn1

// default: keep column 1
const NoSuppressColumn1 = command.CommNoSuppressColumn1

// -2 flag: suppress column 2 (lines only in file2)
const SuppressColumn2 = command.CommSuppressColumn2

// default: keep column 2
const NoSuppressColumn2 = command.CommNoSuppressColumn2

// -3 flag: suppress column 3 (lines in both files)
const SuppressColumn3 = command.CommSuppressColumn3

// default: keep column 3
const NoSuppressColumn3 = command.CommNoSuppressColumn3
