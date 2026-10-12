package official

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBurgerKingPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost || !strings.Contains(r.FormValue("message"), `"trcode":"BKJ0310"`) {
			t.Errorf("request = %s %s", r.Method, r.FormValue("message"))
		}
		_, _ = w.Write([]byte(`{"body":{"pdfUrl":"https://example.com/a.pdf"}}`))
	}))
	defer srv.Close()
	_, fetch := server(t, nil)
	bk := BurgerKing(fetch)
	bk.Base = srv.URL
	urls, err := bk.pdfURLs(context.Background())
	if err != nil || len(urls) != 1 || urls[0] != "https://example.com/a.pdf" {
		t.Errorf("urls = %v, %v", urls, err)
	}
}
