package pelican

import (
	"fmt"

	"github.com/itchio/pelican/pe"

	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
)

type ProbeParams struct {
	Consumer *state.Consumer
	// Return errors instead of printing warnings when
	// we can't parse some parts of the file
	Strict bool
}

// Probe retrieves information about an PE file
func Probe(file eos.File, params ProbeParams) (*PeInfo, error) {
	consumer := params.Consumer

	stats, err := file.Stat()
	if err != nil {
		return nil, err
	}

	pf, err := pe.NewFile(file, stats.Size())
	if err != nil {
		return nil, err
	}

	info := &PeInfo{
		VersionProperties: make(map[string]string),
	}

	switch pf.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		info.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		info.Arch = "amd64"
	}

	imports, err := pf.ImportedLibraries()
	if err != nil {
		if params.Strict {
			return nil, fmt.Errorf("while parsing imported libraries: %w", err)
		}
		consumer.Warnf("Could not parse imported libraries: %+v", err)
	}
	info.Imports = imports

	sect := pf.Section(".rsrc")
	if sect != nil {
		err = params.parseResources(info, sect)
		if err != nil {
			if params.Strict {
				return nil, fmt.Errorf("while parsing resources: %w", err)
			}
			consumer.Warnf("Could not parse resources: %+v", err)
		}
	}

	return info, nil
}
