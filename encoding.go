package main

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

// textEncoding is one encoding offered in File > Encoding.
type textEncoding struct {
	ID    string // stable id, used for menu item IDs
	Label string
	Enc   encoding.Encoding
}

var encodings = []textEncoding{
	{"utf-8", "UTF-8", unicode.UTF8},
	{"utf-8-bom", "UTF-8 with BOM", unicode.UTF8BOM},
	{"utf-16le", "UTF-16 LE", unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)},
	{"utf-16be", "UTF-16 BE", unicode.UTF16(unicode.BigEndian, unicode.UseBOM)},
	{"windows-1252", "Western (Windows-1252)", charmap.Windows1252},
	{"iso-8859-1", "Western (ISO-8859-1)", charmap.ISO8859_1},
	{"iso-8859-15", "Western (ISO-8859-15)", charmap.ISO8859_15},
	{"windows-1250", "Central European (Windows-1250)", charmap.Windows1250},
	{"windows-1251", "Cyrillic (Windows-1251)", charmap.Windows1251},
	{"koi8-r", "Cyrillic (KOI8-R)", charmap.KOI8R},
	{"windows-1253", "Greek (Windows-1253)", charmap.Windows1253},
	{"windows-1254", "Turkish (Windows-1254)", charmap.Windows1254},
	{"windows-1255", "Hebrew (Windows-1255)", charmap.Windows1255},
	{"windows-1256", "Arabic (Windows-1256)", charmap.Windows1256},
	{"windows-874", "Thai (Windows-874 / TIS-620)", charmap.Windows874},
	{"windows-1258", "Vietnamese (Windows-1258)", charmap.Windows1258},
	{"shift_jis", "Japanese (Shift_JIS)", japanese.ShiftJIS},
	{"euc-jp", "Japanese (EUC-JP)", japanese.EUCJP},
	{"gbk", "Chinese Simplified (GBK)", simplifiedchinese.GBK},
	{"gb18030", "Chinese Simplified (GB18030)", simplifiedchinese.GB18030},
	{"big5", "Chinese Traditional (Big5)", traditionalchinese.Big5},
	{"euc-kr", "Korean (EUC-KR)", korean.EUCKR},
}

const defaultEncoding = "utf-8"

func encodingByID(id string) textEncoding {
	for _, e := range encodings {
		if e.ID == id {
			return e
		}
	}
	return encodings[0]
}

// detectEncoding guesses a file's encoding: a byte order mark, else UTF-8
// when the bytes are valid UTF-8, else Windows-1252.
func detectEncoding(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return "utf-8-bom"
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return "utf-16le"
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return "utf-16be"
	case utf8.Valid(b):
		return "utf-8"
	}
	return "windows-1252"
}

// decodeText decodes b with the encoding id, dropping a byte order mark.
func decodeText(b []byte, id string) (string, error) {
	e := encodingByID(id)
	// Decoders of the BOM-aware Unicode encodings remove the BOM; strip it
	// for plain UTF-8 too.
	if id == "utf-8" {
		b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	}
	out, err := e.Enc.NewDecoder().Bytes(b)
	if err != nil {
		return "", fmt.Errorf("decoding as %s: %w", e.Label, err)
	}
	return string(out), nil
}

// encodeText encodes s with the encoding id. Characters the encoding can't
// represent make it fail rather than being silently replaced.
func encodeText(s, id string) ([]byte, error) {
	e := encodingByID(id)
	out, err := e.Enc.NewEncoder().Bytes([]byte(s))
	if err != nil {
		return nil, fmt.Errorf("the text has characters that %s can't represent; choose another encoding (e.g. UTF-8) in File > Encoding: %w", e.Label, err)
	}
	return out, nil
}
