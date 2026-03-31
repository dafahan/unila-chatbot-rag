package pdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

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

// ExtractPages extracts text page-by-page, removes statistical boilerplate,
// and returns one PageText per page. Page numbers are 1-indexed.
func ExtractPages(data []byte) ([]PageText, error) {
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
		text, err := page.GetPlainText(nil)
		if err != nil {
			raw = append(raw, "")
			continue
		}
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
