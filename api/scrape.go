package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type ApiResponse struct {
	Status  int    `json:"status"`
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
		Page struct {
			Total int `json:"total"`
		} `json:"page"`
		Data []Formasi `json:"data"`
	} `json:"data"`
}

type Formasi struct {
	FormasiID     string `json:"formasi_id"`
	InsNm         string `json:"ins_nm"`
	JpNama        string `json:"jp_nama"`
	FormasiNm     string `json:"formasi_nm"`
	JabatanNm     string `json:"jabatan_nm"`
	LokasiNm      string `json:"lokasi_nm"`
	JumlahFormasi int    `json:"jumlah_formasi"`
	GajiMin       string `json:"gaji_min"`
	GajiMax       string `json:"gaji_max"`
}

func getData(offset int, kodeRefPend, pengadaanKd string) (*ApiResponse, error) {
	url := fmt.Sprintf("https://api-sscasn.bkn.go.id/2024/portal/spf?kode_ref_pend=%s&pengadaan_kd=%s&offset=%d", kodeRefPend, pengadaanKd, offset)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to mimic a browser request
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id-ID;q=0.8,id;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", "api-sscasn.bkn.go.id")
	req.Header.Set("Origin", "https://sscasn.bkn.go.id")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", "https://sscasn.bkn.go.id/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="99", "Google Chrome";v="127", "Chromium";v="127"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body safely
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyStr := string(bodyBytes)
	// Optional: log response preview to debug
	if len(bodyStr) > 100 {
		log.Println("Response preview:", bodyStr[:100])
	} else {
		log.Println("Response preview:", bodyStr)
	}

	var apiResp ApiResponse
	err = json.Unmarshal(bodyBytes, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("gagal decode JSON: %w\nresponse body: %s", err, bodyStr)
	}

	if apiResp.Error {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
	}

	return &apiResp, nil
}

func Form(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Form input kode pendidikan
		form := `
		<!DOCTYPE html>
		<html><head><title>Input Kode Pendidikan</title></head><body>
		<h1>Masukkan Kode Pendidikan</h1>
		<form method="POST" action="/scrape">
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

// Handler scrape + pagination + download CSV
func ScrapeHandler(w http.ResponseWriter, r *http.Request) {
	kode := ""
	page := 1
	isDownload := false

	if r.Method == http.MethodPost {
		kode = r.FormValue("kodePendidikan")
		pageStr := r.FormValue("page")
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
	} else if r.Method == http.MethodGet {
		// Kalau ada query param download CSV
		kode = r.URL.Query().Get("kodePendidikan")
		if r.URL.Query().Get("download") == "csv" {
			isDownload = true
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if kode == "" {
		http.Error(w, "Kode Pendidikan wajib diisi", http.StatusBadRequest)
		return
	}

	if isDownload {
		// Download CSV: ambil semua data dari page 1 sampai last page
		downloadCSV(w, kode)
		return
	}

	// Pagination normal: ambil 10 data per page
	offset := (page - 1) * 10
	dataResp, err := getData(offset, kode, "2")
	if err != nil {
		http.Error(w, "Gagal mengambil data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	total := dataResp.Data.Meta.Total
	totalPages := (total + 9) / 10

	// Tampilkan hasil tabel dengan pagination dan tombol download CSV
	tmpl := `
	<!DOCTYPE html>
	<html><head><title>Hasil Scraping</title>
	<style>
		table, th, td { border: 1px solid black; border-collapse: collapse; padding: 5px; }
		form { display: inline; }
		#downloadLink { position: absolute; top: 10px; right: 10px; }
	</style>
	</head><body>
	<h1>Hasil Scraping untuk Kode Pendidikan: {{.Kode}}</h1>

	<!-- Tombol Download CSV -->
	<a id="downloadLink" href="/scrape?kodePendidikan={{.Kode}}&download=csv" target="_blank">Download CSV</a>

	<table>
		<tr>
			<th>Nama Instansi</th>
			<th>Formasi</th>
			<th>Jabatan</th>
			<th>Unit Kerja</th>
			<th>Jumlah Kebutuhan</th>
			<th>Gaji Min</th>
			<th>Gaji Max</th>
			<th>Link</th>
		</tr>
		{{range .Data}}
		<tr>
			<td>{{.InsNm}}</td>
			<td>{{.JpNama}} {{.FormasiNm}}</td>
			<td>{{.JabatanNm}}</td>
			<td>{{.LokasiNm}}</td>
			<td>{{.JumlahFormasi}}</td>
			<td>{{.GajiMin}}</td>
			<td>{{.GajiMax}}</td>
			<td><a href="https://sscasn.bkn.go.id/detailformasi/{{.FormasiID}}" target="_blank">Detail</a></td>
		</tr>
		{{end}}
	</table>
	<br>
	<div>
		{{if gt .Page 1}}
			<form method="POST" action="/scrape">
				<input type="hidden" name="kodePendidikan" value="{{.Kode}}">
				<input type="hidden" name="page" value="{{.PrevPage}}">
				<button type="submit">Previous</button>
			</form>
		{{end}}
		Page {{.Page}} of {{.TotalPages}}
		{{if lt .Page .TotalPages}}
			<form method="POST" action="/scrape">
				<input type="hidden" name="kodePendidikan" value="{{.Kode}}">
				<input type="hidden" name="page" value="{{.NextPage}}">
				<button type="submit">Next</button>
			</form>
		{{end}}
	</div>
	</body></html>`

	t := template.Must(template.New("result").Parse(tmpl))
	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, map[string]interface{}{
		"Kode":       kode,
		"Data":       dataResp.Data.Data,
		"Page":       page,
		"TotalPages": totalPages,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
	})
}

// Fungsi download CSV, scrape semua data hingga habis
func downloadCSV(w http.ResponseWriter, kode string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="data_formasi.csv"`)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Tulis header CSV
	writer.Write([]string{"Nama Instansi", "Formasi", "Jabatan", "Unit Kerja", "Jumlah Kebutuhan", "Gaji Min", "Gaji Max", "Link"})

	offset := 0
	limit := 10

	for {
		dataResp, err := getData(offset, kode, "2")
		if err != nil {
			http.Error(w, "Gagal mengambil data untuk CSV: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(dataResp.Data.Data) == 0 {
			break
		}

		for _, f := range dataResp.Data.Data {
			row := []string{
				f.InsNm,
				strings.TrimSpace(f.JpNama + " " + f.FormasiNm),
				f.JabatanNm,
				f.LokasiNm,
				strconv.Itoa(f.JumlahFormasi),
				f.GajiMin,
				f.GajiMax,
				"https://sscasn.bkn.go.id/detailformasi/" + f.FormasiID,
			}
			writer.Write(row)
		}

		offset += limit
		if offset >= dataResp.Data.Meta.Total {
			break
		}
	}
}

// Vercel entrypoint
func Handler() {
	// Because the file is /api/scrape.go, automatically this Handler will be triggered for /api/scrape
	http.HandleFunc("/", Form) // form input
	http.HandleFunc("/scrape", ScrapeHandler)

	port := "8080" // default
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Println("Server running di port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
