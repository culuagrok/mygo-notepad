package main

import "testing"

func TestRoundTrip(t *testing.T) {
	cases := map[string]string{
		"utf-8": "héllo ไทย", "utf-8-bom": "héllo ไทย", "utf-16le": "héllo ไทย", "utf-16be": "héllo ไทย",
		"windows-1252": "héllo €", "windows-874": "สวัสดี", "shift_jis": "こんにちは", "gbk": "你好",
	}
	for id, s := range cases {
		b, err := encodeText(s, id)
		if err != nil {
			t.Fatal(id, err)
		}
		if got := detectEncoding(b); (id == "utf-8-bom" || id == "utf-16le" || id == "utf-16be" || id == "utf-8") && got != id {
			t.Errorf("%s detected as %s", id, got)
		}
		d, err := decodeText(b, id)
		if err != nil || d != s {
			t.Errorf("%s: got %q, %v", id, d, err)
		}
	}
	if _, err := encodeText("ไทย", "windows-1252"); err == nil {
		t.Error("expected an error for unencodable text")
	}
	if detectEncoding([]byte{'c', 'a', 'f', 0xE9}) != "windows-1252" {
		t.Error("fallback")
	}
}
