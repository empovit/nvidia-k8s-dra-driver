/*
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
*/

// Adapted from https://github.com/rai-project/ldcache

package ldcache

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

const ldcachePath = "/etc/ld.so.cache"

const (
	magicString1 = "ld.so-1.7.0"
	magicString2 = "glibc-ld.so.cache"
	magicVersion = "1.1"
)

const (
	flagTypeMask = 0x00ff
	flagTypeELF  = 0x0001

	flagArchMask    = 0xff00
	flagArchI386    = 0x0000
	flagArchX8664   = 0x0300
	flagArchX32     = 0x0800
	flagArchPpc64le = 0x0500

	// flagArch_ARM_LIBHF is the flag value for 32-bit ARM libs using hard-float.
	flagArch_ARM_LIBHF = 0x0900
	// flagArch_AARCH64_LIB64 is the flag value for 64-bit ARM libs.
	flagArch_AARCH64_LIB64 = 0x0a00
)

var errInvalidCache = errors.New("invalid ld.so.cache file")

type header1 struct {
	Magic [len(magicString1) + 1]byte // include null delimiter
	NLibs uint32
}

type entry1 struct {
	Flags      int32
	Key, Value uint32
}

type header2 struct {
	Magic     [len(magicString2)]byte
	Version   [len(magicVersion)]byte
	NLibs     uint32
	TableSize uint32
	_         [3]uint32 // unused
	_         uint64    // force 8 byte alignment
}

type entry2 struct {
	Flags      int32
	Key, Value uint32
	OSVersion  uint32
	HWCap      uint64
}

// LDCache represents the interface for performing lookups into the LDCache
//
//go:generate moq -rm -out ldcache_mock.go . LDCache
type LDCache interface {
	List() ([]string, []string)
}

type ldcache struct {
	*bytes.Reader

	data, libs []byte
	header     header2
	entries    []entry2

	root   string
	logger logger.Interface
}

// New creates a new LDCache with the specified logger and root.
func New(logger logger.Interface, root string) (LDCache, error) {
	path := filepath.Join(root, ldcachePath)

	logger.Debugf("Opening ld.conf at %v", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	d, err := syscall.Mmap(int(f.Fd()), 0, int(fi.Size()),
		syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}

	cache := &ldcache{
		data:   d,
		Reader: bytes.NewReader(d),
		root:   root,
		logger: logger,
	}
	return cache, cache.parse()
}

func (c *ldcache) Close() error {
	return syscall.Munmap(c.data)
}

func (c *ldcache) Magic() string {
	return string(c.header.Magic[:])
}

func (c *ldcache) Version() string {
	return string(c.header.Version[:])
}

func strn(b []byte, n int) string {
	return string(b[:n])
}

func (c *ldcache) parse() error {
	var header header1

	// Check for the old format (< glibc-2.2)
	if c.Len() <= int(unsafe.Sizeof(header)) {
		return errInvalidCache
	}
	if strn(c.data, len(magicString1)) == magicString1 {
		if err := binary.Read(c, binary.LittleEndian, &header); err != nil {
			return err
		}
		n := int64(header.NLibs) * int64(unsafe.Sizeof(entry1{}))
		offset, err := c.Seek(n, 1) // skip old entries
		if err != nil {
			return err
		}
		n = (-offset) & int64(unsafe.Alignof(c.header)-1)
		_, err = c.Seek(n, 1) // skip padding
		if err != nil {
			return err
		}
	}

	c.libs = c.data[c.Size()-int64(c.Len()):] // kv offsets start here
	if err := binary.Read(c, binary.LittleEndian, &c.header); err != nil {
		return err
	}
	if c.Magic() != magicString2 || c.Version() != magicVersion {
		return errInvalidCache
	}
	c.entries = make([]entry2, c.header.NLibs)
	if err := binary.Read(c, binary.LittleEndian, &c.entries); err != nil {
		return err
	}
	return nil
}

type entry struct {
	libname string
	bits    int
	value   string
}

// getEntries returns the entires of the ldcache in a go-friendly struct.
func (c *ldcache) getEntries() []entry {
	var entries []entry
	for _, e := range c.entries {
		bits := 0
		if ((e.Flags & flagTypeMask) & flagTypeELF) == 0 {
			continue
		}
		switch e.Flags & flagArchMask {
		case flagArchX8664:
			fallthrough
		case flagArch_AARCH64_LIB64:
			fallthrough
		case flagArchPpc64le:
			bits = 64
		case flagArchX32:
			fallthrough
		case flagArch_ARM_LIBHF:
			fallthrough
		case flagArchI386:
			bits = 32
		default:
			continue
		}
		if e.Key > uint32(len(c.libs)) || e.Value > uint32(len(c.libs)) {
			continue
		}
		lib := bytesToString(c.libs[e.Key:])
		if lib == "" {
			c.logger.Debugf("Skipping invalid lib")
			continue
		}
		value := bytesToString(c.libs[e.Value:])
		if value == "" {
			c.logger.Debugf("Skipping invalid value for lib %v", lib)
			continue
		}
		e := entry{
			libname: lib,
			bits:    bits,
			value:   value,
		}
		entries = append(entries, e)
	}
	return entries
}

// List creates a list of libraries in the ldcache.
// The 32-bit and 64-bit libraries are returned separately.
func (c *ldcache) List() ([]string, []string) {
	paths := make(map[int][]string)
	processed := make(map[string]bool)

	for _, e := range c.getEntries() {
		path := filepath.Join(c.root, e.value)
		if processed[path] {
			continue
		}
		paths[e.bits] = append(paths[e.bits], path)
		processed[path] = true
	}

	return paths[32], paths[64]
}

// bytesToString converts a byte slice to a string.
// This assumes that the byte slice is null-terminated
func bytesToString(value []byte) string {
	n := bytes.IndexByte(value, 0)
	if n < 0 {
		return ""
	}

	return strn(value, n)
}
