// Copyright 2018-2021 the LinuxBoot Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cbfs

import (
	"fmt"
	"io"
	"log"
)

func init() {
	if err := RegisterFileReader(&SegReader{Type: TypeSELF, Name: "Payload", New: NewPayloadRecord}); err != nil {
		log.Fatal(err)
	}
}

func NewPayloadRecord(f *File) (ReadWriter, error) {
	p := &PayloadRecord{File: *f}
	return p, nil
}

func (p *PayloadRecord) Read(in io.ReadSeeker) error {
	for {
		var h PayloadHeader
		if err := Read(in, &h); err != nil {
			Debug("PayloadHeader read: %v", err)
			return err
		}
		Debug("Got PayloadHeader %s", h.String())
		p.Segs = append(p.Segs, h)
		if h.Type == SegEntry {
			break
		}
	}
	// Seek to offset (after the header); the remainder is the actual payload.
	offset, err := in.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("Finding location in stream: %v", err)
	}
	bodySize := int64(p.Size) - offset
	Debug("Payload size: %v, body size: %v, offset: %v", p.Size, bodySize, offset)
	if bodySize < 0 {
		// This should not happen. Tolerate a potential error.
		return nil
	}
	// This _may_ happen. E.g. with the test payload here. Silently ignore.
	if bodySize == 0 {
		Debug("Payload empty, nothing to read")
		return nil
	}
	p.FData = make([]byte, bodySize)
	n, err := in.Read(p.FData)
	if err != nil {
		return err
	}
	Debug("Payload read %d bytes", n)
	return nil
}

func (h *PayloadRecord) String() string {
	s := recString(h.File.Name, h.RecordStart, h.Type.String(), h.Size, "none")
	for i, seg := range h.Segs {
		s += "\n"
		s += recString(fmt.Sprintf(" Seg #%d", i), seg.Offset, seg.Type.String(), seg.Size, seg.Compression.String())
	}
	return s
}

func (r *PayloadHeader) String() string {
	return fmt.Sprintf("Type %#x Compression %#x Offset %#x LoadAddress %#x Size %#x MemSize %#x",
		r.Type,
		r.Compression,
		r.Offset,
		r.LoadAddress,
		r.Size,
		r.MemSize)
}

func (r *PayloadRecord) Write(w io.Writer) error {
	if err := Write(w, r.Segs); err != nil {
		return err
	}
	return Write(w, r.FData)
}

func (r *PayloadRecord) GetFile() *File {
	return &r.File
}
