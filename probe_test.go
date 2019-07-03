package pelican_test

import (
	"testing"

	"github.com/itchio/pelican"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/headway/state"
	"github.com/stretchr/testify/assert"
)

func testProbeParams(t *testing.T) *pelican.ProbeParams {
	return &pelican.ProbeParams{
		Consumer: &state.Consumer{
			OnMessage: func(level string, message string) {
				if level == "debug" {
					return
				}
				t.Logf("[%s] %s", level, message)
			},
		},
		Strict: true,
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

	assert.Nil(t, info.AssemblyInfo)
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

func Test_WinCDEmuInstaller(t *testing.T) {
	f, err := eos.Open("./testdata/wincdemu/WinCDEmu-4.1.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, info.Arch)

	vp := info.VersionProperties
	assert.EqualValues(t, "Sysprogs OU", vp["CompanyName"])
	assert.EqualValues(t, "WinCDEmu installer", vp["FileDescription"])
	assert.EqualValues(t, "4.1", vp["FileVersion"])
	assert.EqualValues(t, "LGPL", vp["LegalCopyright"])
	assert.EqualValues(t, "WinCDEmu", vp["ProductName"])
	assert.EqualValues(t, "4.1", vp["ProductVersion"])

	assert.NotNil(t, info.AssemblyInfo)
	assert.EqualValues(t, "requireAdministrator", info.AssemblyInfo.RequestedExecutionLevel)
	assert.True(t, info.RequiresElevation())

	assert.EqualValues(t, 1, len(info.DependentAssemblies))
	da := info.DependentAssemblies[0]
	assert.EqualValues(t, "Microsoft.Windows.Common-Controls", da.Name)
	assert.EqualValues(t, "*", da.Language)
	assert.EqualValues(t, "*", da.ProcessorArchitecture)
	assert.EqualValues(t, "6595b64144ccf1df", da.PublicKeyToken)
}

func Test_PidginUninstaller(t *testing.T) {
	f, err := eos.Open("./testdata/pidgin/pidgin-uninst.exe")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, info.Arch)

	vp := info.VersionProperties
	assert.EqualValues(t, "Pidgin Installer", vp["FileDescription"])
	assert.EqualValues(t, "2.10.11", vp["FileVersion"])
	assert.EqualValues(t, "Pidgin", vp["ProductName"])
	assert.EqualValues(t, "2.10.11", vp["ProductVersion"])

	assert.NotNil(t, info.AssemblyInfo)
	assert.EqualValues(t, "highestAvailable", info.AssemblyInfo.RequestedExecutionLevel)
	assert.True(t, info.RequiresElevation())

	assert.EqualValues(t, 1, len(info.DependentAssemblies))
	da := info.DependentAssemblies[0]
	assert.EqualValues(t, "Microsoft.Windows.Common-Controls", da.Name)
	assert.EqualValues(t, "*", da.Language)
	assert.EqualValues(t, "X86", da.ProcessorArchitecture)
	assert.EqualValues(t, "6595b64144ccf1df", da.PublicKeyToken)
}

func Test_Stockboy(t *testing.T) {
	f, err := eos.Open("./testdata/stockboy/stockboy_install_sliced.EXE")
	assert.NoError(t, err)
	defer f.Close()

	info, err := pelican.Probe(f, testProbeParams(t))
	assert.NoError(t, err)
	assert.EqualValues(t, pelican.Arch386, info.Arch)
}
