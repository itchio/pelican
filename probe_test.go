package pelican_test

import (
	"testing"

	"github.com/itchio/pelican"
	"github.com/itchio/wharf/eos"
	"github.com/stretchr/testify/assert"
)

func Test_NotPeFile(t *testing.T) {
	f, err := eos.Open("./testdata/hello.c")
	assert.NoError(t, err)
	defer f.Close()

	_, err = pelican.Probe(f, nil)
	assert.Error(t, err)
}

func Test_Hello32Mingw(t *testing.T) {
	f, err := eos.Open("./testdata/hello32-mingw.exe")
	assert.NoError(t, err)
	defer f.Close()

	res, err := pelican.Probe(f, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, res.Arch)
}

func Test_Hello32Msvc(t *testing.T) {
	f, err := eos.Open("./testdata/hello32-msvc.exe")
	assert.NoError(t, err)
	defer f.Close()

	res, err := pelican.Probe(f, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, res.Arch)
}

func Test_Hello64Mingw(t *testing.T) {
	f, err := eos.Open("./testdata/hello64-mingw.exe")
	assert.NoError(t, err)
	defer f.Close()

	res, err := pelican.Probe(f, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.ArchAmd64, res.Arch)
}

func Test_Hello64Msvc(t *testing.T) {
	f, err := eos.Open("./testdata/hello64-msvc.exe")
	assert.NoError(t, err)
	defer f.Close()

	res, err := pelican.Probe(f, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.ArchAmd64, res.Arch)
}
