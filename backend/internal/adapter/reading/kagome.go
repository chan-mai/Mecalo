// 形態素解析による読みの取得
package reading

import (
	"strings"
	"sync"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"

	"github.com/chan-mai/mecalo/backend/internal/port"
)

type Kagome struct {
	once sync.Once
	t    *tokenizer.Tokenizer
	err  error
}

var _ port.TextReader = (*Kagome)(nil)

// 辞書は初回利用時に読込
func New() *Kagome {
	return &Kagome{}
}

func (k *Kagome) init() error {
	k.once.Do(func() {
		k.t, k.err = tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	})
	return k.err
}

// 空白区切りの語ごとに読みへ変換, 読みが不明な語は表記のまま
func (k *Kagome) Reading(text string) string {
	if err := k.init(); err != nil {
		return text
	}
	words := strings.Fields(text)
	out := make([]string, 0, len(words))
	for _, w := range words {
		var b strings.Builder
		for _, tok := range k.t.Tokenize(w) {
			if r, ok := tok.Reading(); ok && r != "*" {
				b.WriteString(r)
			} else {
				b.WriteString(tok.Surface)
			}
		}
		out = append(out, b.String())
	}
	return strings.Join(out, " ")
}
