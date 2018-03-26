package pelican

import (
	"debug/pe"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/eos"
)

type Arch string

const (
	Arch386   = "386"
	ArchAmd64 = "amd64"
)

type PeInfo struct {
	Arch Arch
}

type ProbeParams struct {
	// nothing yet
}

// Probe retrieves information about an PE file
func Probe(file eos.File, params *ProbeParams) (*PeInfo, error) {
	pf, err := pe.NewFile(file)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &PeInfo{}

	switch pf.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		res.Arch = Arch386
	case pe.IMAGE_FILE_MACHINE_AMD64:
		res.Arch = ArchAmd64
	}
	return res, nil
}
