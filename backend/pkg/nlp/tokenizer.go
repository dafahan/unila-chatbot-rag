package nlp

import (
	"strings"

	"github.com/RadhiFadlillah/go-sastrawi"
)

var (
	stemmer  = sastrawi.NewStemmer(sastrawi.DefaultDictionary())
	stopword = sastrawi.DefaultStopword()
)

// Tokenize splits text into lowercase stemmed tokens, removing stopwords and
// short words. Uses the Sastrawi Indonesian NLP library.
func Tokenize(text string) []string {
	raw := sastrawi.Tokenize(strings.ToLower(text))
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		if stopword.Contains(w) {
			continue
		}
		stemmed := stemmer.Stem(w)
		if len(stemmed) > 2 {
			out = append(out, stemmed)
		}
	}
	return out
}
