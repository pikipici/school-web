// Package attachments validates and stores the teacher's PDF/PPT/PPTX files.
//
// The type is decided from the file content, never from what the browser
// claims; the name's extension must agree with it. Macro-enabled
// presentations are refused. Parsing is bounded: nothing is decompressed
// except one small XML part, and every offset read from a file is checked
// before use, so a hostile file cannot make the server allocate or loop
// without limit.
package attachments

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"path"
	"strings"
	"unicode/utf16"
)

// Kind is a validated attachment type.
type Kind int

const (
	PDF Kind = iota + 1
	PPT
	PPTX
)

func (k Kind) String() string {
	switch k {
	case PDF:
		return "pdf"
	case PPT:
		return "ppt"
	case PPTX:
		return "pptx"
	}
	return ""
}

// ContentType is the MIME type sent when the file is downloaded.
func (k Kind) ContentType() string {
	switch k {
	case PDF:
		return "application/pdf"
	case PPT:
		return "application/vnd.ms-powerpoint"
	case PPTX:
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	}
	return ""
}

var (
	// ErrType means the content is not a PDF/PPT/PPTX, or does not match the name.
	ErrType = errors.New("attachments: unsupported or mismatched file type")
	// ErrMacro means the presentation contains macros.
	ErrMacro = errors.New("attachments: macro-enabled presentation")
)

const (
	maxContentTypesXML = 1 << 20
	maxZipEntries      = 20000
	maxCFBSectors      = 1 << 20
	maxDirEntries      = 1 << 16
)

var (
	pdfMagic = []byte("%PDF-")
	zipMagic = []byte("PK\x03\x04")
	cfbMagic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
)

// Detect reads r (size bytes long) and reports its kind. The extension of
// name must be .pdf, .ppt, or .pptx and must match the content.
func Detect(r io.ReaderAt, size int64, name string) (Kind, error) {
	want := kindFromName(name)
	if want == 0 || size < 8 {
		return 0, ErrType
	}
	head := make([]byte, 8)
	if _, err := r.ReadAt(head, 0); err != nil {
		return 0, ErrType
	}

	var got Kind
	var err error
	switch {
	case bytes.HasPrefix(head, pdfMagic) && (head[5] == '1' || head[5] == '2') && head[6] == '.':
		got = PDF
	case bytes.HasPrefix(head, zipMagic):
		got, err = PPTX, checkPPTX(r, size)
	case bytes.Equal(head, cfbMagic):
		got, err = PPT, checkPPT(r, size)
	default:
		return 0, ErrType
	}
	if err != nil {
		return 0, err
	}
	if got != want {
		return 0, ErrType
	}
	return got, nil
}

func kindFromName(name string) Kind {
	switch strings.ToLower(path.Ext(strings.ReplaceAll(name, `\`, "/"))) {
	case ".pdf":
		return PDF
	case ".ppt":
		return PPT
	case ".pptx":
		return PPTX
	}
	return 0
}

// checkPPTX confirms an Office Open XML presentation without macros.
func checkPPTX(r io.ReaderAt, size int64) error {
	z, err := zip.NewReader(r, size)
	if err != nil || len(z.File) > maxZipEntries {
		return ErrType
	}
	var types *zip.File
	hasPresentation := false
	for _, f := range z.File {
		lower := strings.ToLower(f.Name)
		switch {
		case lower == "[content_types].xml":
			types = f
		case lower == "ppt/presentation.xml":
			hasPresentation = true
		case strings.HasSuffix(lower, "vbaproject.bin"):
			return ErrMacro
		}
	}
	if types == nil || !hasPresentation || types.UncompressedSize64 > maxContentTypesXML {
		return ErrType
	}

	rc, err := types.Open()
	if err != nil {
		return ErrType
	}
	defer rc.Close()
	xml, err := io.ReadAll(io.LimitReader(rc, maxContentTypesXML+1))
	if err != nil || len(xml) > maxContentTypesXML {
		return ErrType
	}
	lower := strings.ToLower(string(xml))
	if strings.Contains(lower, "macroenabled") || strings.Contains(lower, "vbaproject") {
		return ErrMacro
	}
	if !strings.Contains(lower, "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml") {
		return ErrType
	}
	return nil
}

// checkPPT walks the directory of a Compound File Binary (OLE2) and requires
// a "PowerPoint Document" stream and no VBA storage.
func checkPPT(r io.ReaderAt, size int64) error {
	hdr := make([]byte, 512)
	if size < 512 {
		return ErrType
	}
	if _, err := r.ReadAt(hdr, 0); err != nil {
		return ErrType
	}
	shift := binary.LittleEndian.Uint16(hdr[0x1E:])
	if shift != 9 && shift != 12 {
		return ErrType
	}
	sector := int64(1) << shift
	numSectors := (size - sector) / sector
	if numSectors <= 0 || numSectors > maxCFBSectors {
		return ErrType
	}

	readSector := func(id uint32, buf []byte) error {
		if int64(id) >= numSectors {
			return ErrType
		}
		if _, err := r.ReadAt(buf, (int64(id)+1)*sector); err != nil {
			return ErrType
		}
		return nil
	}

	// FAT sector locations: 109 in the header, the rest in the DIFAT chain.
	numFAT := binary.LittleEndian.Uint32(hdr[0x2C:])
	if numFAT == 0 || int64(numFAT) > numSectors {
		return ErrType
	}
	fatSectors := make([]uint32, 0, numFAT)
	for i := 0; i < 109 && uint32(len(fatSectors)) < numFAT; i++ {
		fatSectors = append(fatSectors, binary.LittleEndian.Uint32(hdr[0x4C+i*4:]))
	}
	difat := binary.LittleEndian.Uint32(hdr[0x44:])
	buf := make([]byte, sector)
	perSector := int(sector / 4)
	for hops := 0; uint32(len(fatSectors)) < numFAT; hops++ {
		if hops > int(numSectors) || readSector(difat, buf) != nil {
			return ErrType
		}
		for i := 0; i < perSector-1 && uint32(len(fatSectors)) < numFAT; i++ {
			fatSectors = append(fatSectors, binary.LittleEndian.Uint32(buf[i*4:]))
		}
		difat = binary.LittleEndian.Uint32(buf[(perSector-1)*4:])
	}

	// Load only as much of the FAT as the file can address.
	fat := make([]uint32, 0, min(int64(len(fatSectors))*int64(perSector), numSectors))
	for _, s := range fatSectors {
		if readSector(s, buf) != nil {
			return ErrType
		}
		for i := 0; i < perSector && int64(len(fat)) < numSectors; i++ {
			fat = append(fat, binary.LittleEndian.Uint32(buf[i*4:]))
		}
	}

	const endOfChain = 0xFFFFFFFE
	hasDocument := false
	seen := make(map[uint32]bool)
	entries := 0
	for s := binary.LittleEndian.Uint32(hdr[0x30:]); s != endOfChain; {
		if seen[s] || int64(s) >= int64(len(fat)) || readSector(s, buf) != nil {
			return ErrType
		}
		seen[s] = true
		for off := int64(0); off+128 <= sector; off += 128 {
			entries++
			if entries > maxDirEntries {
				return ErrType
			}
			e := buf[off : off+128]
			typ := e[0x42]
			if typ == 0 {
				continue
			}
			n := int(binary.LittleEndian.Uint16(e[0x40:]))
			if n < 2 || n > 64 || n%2 != 0 {
				continue
			}
			u := make([]uint16, n/2-1)
			for i := range u {
				u[i] = binary.LittleEndian.Uint16(e[i*2:])
			}
			name := strings.ToLower(string(utf16.Decode(u)))
			switch {
			case typ == 2 && name == "powerpoint document":
				hasDocument = true
			case name == "_vba_project_cur" || name == "vba" || name == "macros" || name == "_vba_project":
				return ErrMacro
			}
		}
		s = fat[s]
	}
	if !hasDocument {
		return ErrType
	}
	return nil
}
