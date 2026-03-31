package pdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

var (
	// Baris daftar isi: hanya titik-titik dan spasi
	reDots = regexp.MustCompile(`(?m)^[.\s]{5,}$`)
	// Baris kosong berlebih
	reExcessNewlines = regexp.MustCompile(`\n{3,}`)
)

// ExtractText reads raw PDF bytes, removes statistical noise, and returns plain text.
func ExtractText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf reader: %w", err)
	}

	pages := make([]string, 0, r.NumPage())
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		pages = append(pages, text)
	}

	// Deteksi header/footer: baris yang muncul di lebih dari 30% halaman
	boilerplate := detectBoilerplate(pages, 0.30)

	var buf bytes.Buffer
	for _, page := range pages {
		for _, line := range strings.Split(page, "\n") {
			normalized := strings.TrimSpace(line)
			if boilerplate[normalized] {
				continue
			}
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}

	return cleanText(buf.String()), nil
}

// detectBoilerplate returns lines that appear in more than `threshold` fraction of pages.
// These are likely headers, footers, or repeated section titles.
func detectBoilerplate(pages []string, threshold float64) map[string]bool {
	if len(pages) == 0 {
		return nil
	}

	// Count how many pages each normalized line appears in
	linePageCount := make(map[string]int)
	for _, page := range pages {
		seen := make(map[string]bool)
		for _, line := range strings.Split(page, "\n") {
			normalized := strings.TrimSpace(line)
			if len(normalized) < 5 { // skip baris sangat pendek
				continue
			}
			if !seen[normalized] {
				linePageCount[normalized]++
				seen[normalized] = true
			}
		}
	}

	minPages := int(float64(len(pages)) * threshold)
	if minPages < 2 {
		minPages = 2
	}

	boilerplate := make(map[string]bool)
	for line, count := range linePageCount {
		if count >= minPages {
			boilerplate[line] = true
		}
	}
	return boilerplate
}

func cleanText(text string) string {
	text = reDots.ReplaceAllString(text, "")
		text = reExcessNewlines.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}
