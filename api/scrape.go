package handler

import (
	"fmt"
	"net/http"
)

func main() {}

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		form := `
		<!DOCTYPE html>
		<html><head><title>Input Kode Pendidikan</title></head><body>
		<h1>Masukkan Kode Pendidikan</h1>
		<form method="GET" action="/api/scrape">
			<label>Kode Pendidikan: <input type="text" name="kodePendidikan" required></label>
			<button type="submit">Cari</button>
		</form>
		</body></html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, form)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
