package reader_test

import (
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestParseFileType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    reader.FileType
		wantErr error
	}{
		{name: "pdf without dot", input: "pdf", want: reader.FileTypePDF},
		{name: "pdf with dot", input: ".pdf", want: reader.FileTypePDF},
		{name: "PDF uppercase", input: "PDF", want: reader.FileTypePDF},
		{name: "docx without dot", input: "docx", want: reader.FileTypeDOCX},
		{name: "docx with dot", input: ".docx", want: reader.FileTypeDOCX},
		{name: "DOCX mixed case", input: ".DocX", want: reader.FileTypeDOCX},
		{name: "md without dot", input: "md", want: reader.FileTypeMD},
		{name: "md with dot", input: ".md", want: reader.FileTypeMD},
		{name: "txt without dot", input: "txt", want: reader.FileTypeTXT},
		{name: "TXT uppercase with dot", input: ".TXT", want: reader.FileTypeTXT},
		{name: "csv without dot", input: "csv", want: reader.FileTypeCSV},
		{name: "csv with dot", input: ".csv", want: reader.FileTypeCSV},
		{name: "xlsx without dot", input: "xlsx", want: reader.FileTypeXLSX},
		{name: "XLSX uppercase", input: ".XLSX", want: reader.FileTypeXLSX},
		{name: "unsupported xls (legacy excel)", input: "xls", wantErr: reader.ErrUnsupportedFormat},
		{name: "unsupported odt", input: "odt", wantErr: reader.ErrUnsupportedFormat},
		{name: "unsupported empty", input: "", wantErr: reader.ErrUnsupportedFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reader.ParseFileType(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseFileType(%q) err = %v, want errors.Is %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFileType(%q) unexpected err: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseFileType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFileType_String(t *testing.T) {
	tests := []struct {
		ft   reader.FileType
		want string
	}{
		{reader.FileTypePDF, "pdf"},
		{reader.FileTypeDOCX, "docx"},
		{reader.FileTypeMD, "md"},
		{reader.FileTypeTXT, "txt"},
		{reader.FileTypeCSV, "csv"},
		{reader.FileTypeXLSX, "xlsx"},
	}
	for _, tt := range tests {
		t.Run(string(tt.ft), func(t *testing.T) {
			if got := tt.ft.String(); got != tt.want {
				t.Errorf("FileType(%q).String() = %q, want %q", tt.ft, got, tt.want)
			}
		})
	}
}

func TestSentinelsAreDistinct(t *testing.T) {
	sentinels := map[string]error{
		"ErrEmptyContent":      reader.ErrEmptyContent,
		"ErrUnsupportedFormat": reader.ErrUnsupportedFormat,
		"ErrCorruptedFile":     reader.ErrCorruptedFile,
		"ErrFileTooLarge":      reader.ErrFileTooLarge,
	}
	for name, e := range sentinels {
		if e == nil {
			t.Errorf("%s is nil", name)
		}
	}
	for n1, e1 := range sentinels {
		for n2, e2 := range sentinels {
			if n1 != n2 && errors.Is(e1, e2) {
				t.Errorf("%s should not match %s via errors.Is", n1, n2)
			}
		}
	}
}
