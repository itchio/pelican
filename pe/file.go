// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pe implements access to PE (Microsoft Windows Portable Executable) files.
package pe

import (
	"debug/dwarf"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Avoid use of post-Go 1.4 io features, to make safe for toolchain bootstrap.
const seekStart = 0

// A File represents an open PE file.
type File struct {
	FileHeader
	OptionalHeader interface{} // of type *OptionalHeader32 or *OptionalHeader64
	Sections       []*Section
	Symbols        []*Symbol    // COFF symbols with auxiliary symbol records removed
	COFFSymbols    []COFFSymbol // all COFF symbols (including auxiliary symbol records)
	StringTable    StringTable

	closer   io.Closer
	readerAt io.ReaderAt
	base     int64
	size     int64
}

var (
	sizeofOptionalHeader32 = uint16(binary.Size(OptionalHeader32{}))
	sizeofOptionalHeader64 = uint16(binary.Size(OptionalHeader64{}))
)

// TODO(brainman): add Load function, as a replacement for NewFile, that does not call removeAuxSymbols (for performance)

// NewFile creates a new File for accessing a PE binary in an underlying reader.
func NewFile(r io.ReaderAt, size int64) (*File, error) {
	f := new(File)
	f.size = size
	f.readerAt = r
	sr := io.NewSectionReader(r, 0, size)

	var dosheader [96]byte
	if _, err := r.ReadAt(dosheader[0:], 0); err != nil {
		return nil, err
	}
	var base int64
	if dosheader[0] == 'M' && dosheader[1] == 'Z' {
		signoff := int64(binary.LittleEndian.Uint32(dosheader[0x3c:]))
		var sign [4]byte
		_, err := r.ReadAt(sign[:], signoff)
		if err != nil {
			return nil, err
		}
		if !(sign[0] == 'P' && sign[1] == 'E' && sign[2] == 0 && sign[3] == 0) {
			return nil, fmt.Errorf("Invalid PE COFF file signature of %v.", sign)
		}
		base = signoff + 4
	} else {
		base = int64(0)
	}
	_, err := sr.Seek(base, seekStart)
	if err != nil {
		return nil, err
	}
	if err = binary.Read(sr, binary.LittleEndian, &f.FileHeader); err != nil {
		return nil, err
	}
	switch f.FileHeader.Machine {
	case IMAGE_FILE_MACHINE_UNKNOWN, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_MACHINE_I386:
	default:
		return nil, fmt.Errorf("Unrecognised COFF file header machine value of 0x%x.", f.FileHeader.Machine)
	}

	// Read string table.
	f.StringTable, err = readStringTable(f, &f.FileHeader, sr)
	if err != nil {
		return nil, err
	}

	// Read symbol table.
	f.COFFSymbols, err = readCOFFSymbols(f, &f.FileHeader, sr)
	if err != nil {
		return nil, err
	}
	f.Symbols, err = removeAuxSymbols(f.COFFSymbols, f.StringTable)
	if err != nil {
		return nil, err
	}

	// Read optional header.
	f.base = base
	sr.Seek(base, seekStart)
	if err := binary.Read(sr, binary.LittleEndian, &f.FileHeader); err != nil {
		return nil, err
	}
	var oh32 OptionalHeader32
	var oh64 OptionalHeader64
	switch f.FileHeader.SizeOfOptionalHeader {
	case sizeofOptionalHeader32:
		if err := binary.Read(sr, binary.LittleEndian, &oh32); err != nil {
			return nil, err
		}
		if oh32.Magic != 0x10b { // PE32
			return nil, fmt.Errorf("pe32 optional header has unexpected Magic of 0x%x", oh32.Magic)
		}
		f.OptionalHeader = &oh32
	case sizeofOptionalHeader64:
		if err := binary.Read(sr, binary.LittleEndian, &oh64); err != nil {
			return nil, err
		}
		if oh64.Magic != 0x20b { // PE32+
			return nil, fmt.Errorf("pe32+ optional header has unexpected Magic of 0x%x", oh64.Magic)
		}
		f.OptionalHeader = &oh64
	}

	// Process sections.
	f.Sections = make([]*Section, f.FileHeader.NumberOfSections)
	for i := 0; i < int(f.FileHeader.NumberOfSections); i++ {
		sh := new(SectionHeader32)
		if err := binary.Read(sr, binary.LittleEndian, sh); err != nil {
			return nil, err
		}
		name, err := sh.fullName(f.StringTable)
		if err != nil {
			return nil, err
		}
		s := new(Section)
		s.SectionHeader = SectionHeader{
			Name:                 name,
			VirtualSize:          sh.VirtualSize,
			VirtualAddress:       sh.VirtualAddress,
			Size:                 sh.SizeOfRawData,
			Offset:               sh.PointerToRawData,
			PointerToRelocations: sh.PointerToRelocations,
			PointerToLineNumbers: sh.PointerToLineNumbers,
			NumberOfRelocations:  sh.NumberOfRelocations,
			NumberOfLineNumbers:  sh.NumberOfLineNumbers,
			Characteristics:      sh.Characteristics,
		}
		r2 := r
		if sh.PointerToRawData == 0 { // .bss must have all 0s
			r2 = zeroReaderAt{}
		}
		s.sr = io.NewSectionReader(r2, int64(s.SectionHeader.Offset), int64(s.SectionHeader.Size))
		s.ReaderAt = s.sr
		f.Sections[i] = s
	}
	for i := range f.Sections {
		var err error
		f.Sections[i].Relocs, err = readRelocs(&f.Sections[i].SectionHeader, sr)
		if err != nil {
			return nil, err
		}
	}

	return f, nil
}

// zeroReaderAt is ReaderAt that reads 0s.
type zeroReaderAt struct{}

// ReadAt writes len(p) 0s into p.
func (w zeroReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// getString extracts a string from symbol string table.
func getString(section []byte, start int) (string, bool) {
	if start < 0 || start >= len(section) {
		return "", false
	}

	for end := start; end < len(section); end++ {
		if section[end] == 0 {
			return string(section[start:end]), true
		}
	}
	return "", false
}

// Section returns the first section with the given name, or nil if no such
// section exists.
func (f *File) Section(name string) *Section {
	for _, s := range f.Sections {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (f *File) DWARF() (*dwarf.Data, error) {
	// There are many other DWARF sections, but these
	// are the ones the debug/dwarf package uses.
	// Don't bother loading others.
	var names = [...]string{"abbrev", "info", "line", "ranges", "str"}
	var dat [len(names)][]byte
	for i, name := range names {
		name = ".debug_" + name
		s := f.Section(name)
		if s == nil {
			continue
		}
		b, err := s.Data()
		if err != nil && uint32(len(b)) < s.Size {
			return nil, err
		}
		if 0 < s.VirtualSize && s.VirtualSize < s.Size {
			b = b[:s.VirtualSize]
		}
		dat[i] = b
	}

	abbrev, info, line, ranges, str := dat[0], dat[1], dat[2], dat[3], dat[4]
	return dwarf.New(abbrev, nil, nil, info, line, nil, ranges, str)
}

type ImageImportDescriptor struct {
	OriginalFirstThunk uint32
	TimeDateStamp      uint32
	ForwarderChain     uint32
	Name               uint32
	FirstThunk         uint32
}

// ImportedSymbols returns the names of all symbols
// referred to by the binary f that are expected to be
// satisfied by other libraries at dynamic load time.
// It does not return weak symbols.
func (f *File) ImportedSymbols() ([]string, error) {
	var dd [16]DataDirectory
	switch oh := f.OptionalHeader.(type) {
	case *OptionalHeader32:
		dd = oh.DataDirectory
	case *OptionalHeader64:
		dd = oh.DataDirectory
	}

	importTableAddress := dd[1]

	pe64 := f.Machine == IMAGE_FILE_MACHINE_AMD64

	iStart := int64(importTableAddress.VirtualAddress)
	iEnd := int64(importTableAddress.VirtualAddress) + int64(importTableAddress.Size)
	var ds *Section
	for _, s := range f.Sections {
		sStart := int64(s.VirtualAddress)
		sEnd := int64(s.VirtualAddress) + int64(s.VirtualSize)

		if sStart <= iStart && iEnd <= sEnd {
			ds = s
			break
		}
	}
	if ds == nil {
		// could not find matching section :(
		return nil, nil
	}

	sectionData, err := ds.Data()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sectionData = sectionData[importTableAddress.VirtualAddress-ds.VirtualAddress:]

	var importDirectories []ImageImportDescriptor
	idBlock := sectionData
	for len(idBlock) > 0 {
		var dt ImageImportDescriptor
		dt.OriginalFirstThunk = binary.LittleEndian.Uint32(idBlock[0:4])
		dt.Name = binary.LittleEndian.Uint32(idBlock[12:16])
		dt.FirstThunk = binary.LittleEndian.Uint32(idBlock[16:20])
		idBlock = idBlock[20:]
		if dt.OriginalFirstThunk == 0 {
			break
		}
		importDirectories = append(importDirectories, dt)
	}

	var allSymbols []string
	for _, dt := range importDirectories {
		dll, _ := getString(sectionData, int(dt.Name-importTableAddress.VirtualAddress))

		// seek to OriginalFirstThunk
		thunkDataBlock := sectionData[dt.OriginalFirstThunk-importTableAddress.VirtualAddress:]

		for len(thunkDataBlock) > 0 {
			if pe64 { // 64bit
				va := binary.LittleEndian.Uint64(thunkDataBlock[0:8])
				thunkDataBlock = thunkDataBlock[8:]
				if va == 0 {
					break
				}
				if va&0x8000000000000000 > 0 { // is Ordinal
					// TODO add dynimport ordinal support.
				} else {
					fn, _ := getString(sectionData, int(uint32(va)-importTableAddress.VirtualAddress+2))
					allSymbols = append(allSymbols, fn+":"+dll)
				}
			} else { // 32bit
				va := binary.LittleEndian.Uint32(thunkDataBlock[0:4])
				thunkDataBlock = thunkDataBlock[4:]
				if va == 0 {
					break
				}
				if va&0x80000000 > 0 { // is Ordinal
					// TODO add dynimport ordinal support.
					//ord := va&0x0000FFFF
				} else {
					fn, _ := getString(sectionData, int(va-importTableAddress.VirtualAddress+2))
					allSymbols = append(allSymbols, fn+":"+dll)
				}
			}
		}
	}

	return allSymbols, nil
}

// ImportedLibraries returns the names of all libraries
// referred to by the binary f that are expected to be
// linked with the binary at dynamic link time.
func (f *File) ImportedLibraries() ([]string, error) {
	var dd [16]DataDirectory
	switch oh := f.OptionalHeader.(type) {
	case *OptionalHeader32:
		dd = oh.DataDirectory
	case *OptionalHeader64:
		dd = oh.DataDirectory
	}

	importTableAddress := dd[1]

	iStart := int64(importTableAddress.VirtualAddress)
	iEnd := int64(importTableAddress.VirtualAddress) + int64(importTableAddress.Size)
	var ds *Section
	for _, s := range f.Sections {
		sStart := int64(s.VirtualAddress)
		sEnd := int64(s.VirtualAddress) + int64(s.VirtualSize)

		if sStart <= iStart && iEnd <= sEnd {
			ds = s
			break
		}
	}
	if ds == nil {
		// could not find matching section :(
		return nil, nil
	}

	sectionData, err := ds.Data()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sectionData = sectionData[importTableAddress.VirtualAddress-ds.VirtualAddress:]

	var importDirectories []ImageImportDescriptor
	idBlock := sectionData
	for len(idBlock) > 0 {
		var dt ImageImportDescriptor
		dt.OriginalFirstThunk = binary.LittleEndian.Uint32(idBlock[0:4])
		dt.Name = binary.LittleEndian.Uint32(idBlock[12:16])
		dt.FirstThunk = binary.LittleEndian.Uint32(idBlock[16:20])
		idBlock = idBlock[20:]
		if dt.OriginalFirstThunk == 0 {
			break
		}
		importDirectories = append(importDirectories, dt)
	}

	var dlls []string
	for _, dt := range importDirectories {
		dll, _ := getString(sectionData, int(dt.Name-importTableAddress.VirtualAddress))
		dlls = append(dlls, dll)
	}

	return dlls, nil
}

// FormatError is unused.
// The type is retained for compatibility.
type FormatError struct {
}

func (e *FormatError) Error() string {
	return "unknown error"
}
