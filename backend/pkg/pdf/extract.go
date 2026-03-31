package pdf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

var (
	reDots           = regexp.MustCompile(`(?m)^[.\s]{5,}$`)
	reExcessNewlines = regexp.MustCompile(`\n{3,}`)
)

// PageText holds the cleaned text of a single PDF page.
type PageText struct {
	Page int
	Text string
}

// ExtractPages extracts text page-by-page with boilerplate removal.
// If the native Go extractor produces mostly garbage (non-printable chars),
// it falls back to pdftotext (poppler-utils) for the entire document,
// returning everything as a single page.
func ExtractPages(data []byte) ([]PageText, error) {
	pages, err := extractNative(data)
	if err == nil && !isGarbage(pages) {
		return pages, nil
	}

	// Fallback: pdftotext
	return extractPdftotext(data)
}

// extractNative uses ledongthuc/pdf to extract text per page.
func extractNative(data []byte) ([]PageText, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("pdf reader: %w", err)
	}

	raw := make([]string, 0, r.NumPage())
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			raw = append(raw, "")
			continue
		}
		text, _ := page.GetPlainText(nil)
		raw = append(raw, text)
	}

	boilerplate := detectBoilerplate(raw, 0.30)

	result := make([]PageText, 0, len(raw))
	for i, pageText := range raw {
		var buf bytes.Buffer
		for _, line := range strings.Split(pageText, "\n") {
			normalized := strings.TrimSpace(line)
			if boilerplate[normalized] {
				continue
			}
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
		cleaned := cleanText(buf.String())
		if cleaned != "" {
			result = append(result, PageText{Page: i + 1, Text: cleaned})
		}
	}
	return result, nil
}

// isGarbage returns true if extracted pages are mostly non-printable/non-Latin
// characters — a sign of encoding issues in the PDF.
func isGarbage(pages []PageText) bool {
	if len(pages) == 0 {
		return true
	}
	var total, bad int
	for _, p := range pages {
		for _, r := range p.Text {
			total++
			if r > 127 && !unicode.IsLetter(r) && !unicode.IsSpace(r) {
				bad++
			}
		}
	}
	if total == 0 {
		return true
	}
	return float64(bad)/float64(total) > 0.15
}

// extractPdftotext uses the system pdftotext binary (poppler-utils) as fallback.
// All pages are returned as a single PageText since pdftotext doesn't expose
// per-page boundaries easily without -f/-l flags.
func extractPdftotext(data []byte) ([]PageText, error) {
	tmp, err := os.CreateTemp("", "unila-pdf-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("pdftotext tmp: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()

	out, err := exec.Command("pdftotext", "-layout", tmp.Name(), "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext: %w", err)
	}

	// Split on form-feed character (\f) which pdftotext uses as page separator
	rawPages := strings.Split(string(out), "\f")
	boilerplate := detectBoilerplate(rawPages, 0.30)

	result := make([]PageText, 0, len(rawPages))
	for i, pageText := range rawPages {
		var buf bytes.Buffer
		for _, line := range strings.Split(pageText, "\n") {
			normalized := strings.TrimSpace(line)
			if boilerplate[normalized] {
				continue
			}
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
		cleaned := cleanText(buf.String())
		if cleaned != "" {
			result = append(result, PageText{Page: i + 1, Text: cleaned})
		}
	}
	return result, nil
}

// ExtractText is kept for compatibility — returns all pages concatenated.
func ExtractText(data []byte) (string, error) {
	pages, err := ExtractPages(data)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, p := range pages {
		sb.WriteString(p.Text)
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

func detectBoilerplate(pages []string, threshold float64) map[string]bool {
	if len(pages) == 0 {
		return nil
	}
	linePageCount := make(map[string]int)
	for _, page := range pages {
		seen := make(map[string]bool)
		for _, line := range strings.Split(page, "\n") {
			normalized := strings.TrimSpace(line)
			if len(normalized) < 5 {
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
