package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTsvToCsv(t *testing.T) {
	input := "a\tb\tc\n1\t2\t3\n"
	out := tsvToCsv([]byte(input))
	// strings.Split on \n produces ["a b c", "1 2 3", ""]
	expected := "a,b,c\n1,2,3\n\n"
	if string(out) != expected {
		t.Errorf("tsvToCsv = %q, want %q", string(out), expected)
	}
}

func TestTsvToCsv_EscapesCommas(t *testing.T) {
	input := "hello, world\tfoo\n"
	out := tsvToCsv([]byte(input))
	expected := "\"hello, world\",foo\n\n"
	if string(out) != expected {
		t.Errorf("tsvToCsv with commas = %q, want %q", string(out), expected)
	}
}

func TestTsvToCsv_EscapesQuotes(t *testing.T) {
	input := "say \"hi\"\tval\n"
	out := tsvToCsv([]byte(input))
	expected := "\"say \"\"hi\"\"\",val\n\n"
	if string(out) != expected {
		t.Errorf("tsvToCsv with quotes = %q, want %q", string(out), expected)
	}
}

func TestTsvToCsv_CRLF(t *testing.T) {
	input := "a\tb\r\n1\t2\r\n"
	out := tsvToCsv([]byte(input))
	expected := "a,b\n1,2\n\n"
	if string(out) != expected {
		t.Errorf("tsvToCsv CRLF = %q, want %q", string(out), expected)
	}
}

func TestWriteCSV(t *testing.T) {
	w := httptest.NewRecorder()
	writeCSV(w, "job1", "trades", []byte("a\tb\n1\t2\n"))

	if w.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q", w.Header().Get("Content-Type"))
	}
	cd := w.Header().Get("Content-Disposition")
	if cd != `attachment; filename="backtest-job1-trades.csv"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	body := w.Body.String()
	if body != "a,b\n1,2\n\n" {
		t.Errorf("body = %q", body)
	}
}

func TestCredModeError(t *testing.T) {
	live := credModeError("live", "binance")
	if live == "" {
		t.Error("live credModeError should not be empty")
	}
	paper := credModeError("paper", "binance")
	if paper == "" {
		t.Error("paper credModeError should not be empty")
	}
}


func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", w.Header().Get("Content-Type"))
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
	if w.Body.String() == "" {
		t.Error("body should not be empty")
	}
}
