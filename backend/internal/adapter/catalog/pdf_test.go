package catalog

import (
	"bytes"
	"fmt"
	"testing"
)

// 指定位置に文字列を置いた1ページのPDF
func makePDF(texts []string) []byte {
	var content bytes.Buffer
	for i, t := range texts {
		fmt.Fprintf(&content, "BT /F1 12 Tf 50 %d Td (%s) Tj ET\n", 700-i*20, t)
	}
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", content.Len(), content.String()),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func TestReadPDF(t *testing.T) {
	pages, err := ReadPDF(makePDF([]string{"Menu kcal Protein", "Burger 500 20.5", "Fries 300 3.1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("pages = %d", len(pages))
	}
	lines := pages[0].Lines()
	if len(lines) != 3 {
		t.Fatalf("lines = %+v", lines)
	}
	got := []string{}
	for _, w := range lines[1] {
		got = append(got, w.Text)
	}
	if fmt.Sprint(got) != "[Burger 500 20.5]" {
		t.Errorf("line = %v", got)
	}
	if !(lines[1][0].X < lines[1][1].X && lines[1][1].X < lines[1][2].X) {
		t.Errorf("x order = %+v", lines[1])
	}
	if _, err := ReadPDF([]byte("not a pdf")); err == nil {
		t.Error("invalid pdf accepted")
	}
}
