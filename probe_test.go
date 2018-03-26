package pelican_test

import (
	"testing"

	"github.com/itchio/pelican"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/stretchr/testify/assert"
)

func testProbeParams(t *testing.T) *pelican.ProbeParams {
	return &pelican.ProbeParams{
		Consumer: &state.Consumer{
			OnMessage: func(level string, message string) {
				t.Logf("[%s] %s", level, message)
			},
		},
	}
}

func Test_NotPeFile(t *testing.T) {
	f, err := eos.Open("./testdata/hello/hello.c")
	assert.NoError(t, err)
	defer f.Close()

	_, err = pelican.Probe(f, testProbeParams(t))
	assert.Error(t, err)
}

func Test_Hello32Mingw(t *testing.T) {
	f, err := eos.Open("./testdata/hello/hello32-mingw.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, info.Arch)
}

func Test_Hello32Msvc(t *testing.T) {
	f, err := eos.Open("./testdata/hello/hello32-msvc.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, info.Arch)
}

func Test_Hello64Mingw(t *testing.T) {
	f, err := eos.Open("./testdata/hello/hello64-mingw.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.ArchAmd64, info.Arch)
}

func Test_Hello64Msvc(t *testing.T) {
	f, err := eos.Open("./testdata/hello/hello64-msvc.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.ArchAmd64, info.Arch)
}

func assertResources(t *testing.T, info *pelican.PeInfo) {
	vp := info.VersionProperties
	assert.EqualValues(t, "itch corp.", vp["CompanyName"])
	assert.EqualValues(t, "Test PE file for pelican", vp["FileDescription"])
	assert.EqualValues(t, "3.14", vp["FileVersion"])
	assert.EqualValues(t, "resourceful", vp["InternalName"])
	assert.EqualValues(t, "(c) 2018 itch corp.", vp["LegalCopyright"])

	// totally a mistake, but leaving this as a reminder that
	// not everything is worth fixing
	assert.EqualValues(t, "butler", vp["ProductName"])

	assert.EqualValues(t, "6.28", vp["ProductVersion"])
}

func Test_Resourceful32Mingw(t *testing.T) {
	f, err := eos.Open("./testdata/resourceful/resourceful32-mingw.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, info.Arch)

	assertResources(t, info)
}

func Test_Resourceful64Mingw(t *testing.T) {
	f, err := eos.Open("./testdata/resourceful/resourceful64-mingw.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.ArchAmd64, info.Arch)

	assertResources(t, info)
}
