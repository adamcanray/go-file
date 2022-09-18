package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type M map[string]interface{}

func main() {
	http.HandleFunc("/", routeIndexGet)
	http.HandleFunc("/process", routeSubmitPost)
	http.HandleFunc("/list-files", handleListFiles)
	http.HandleFunc("/download", handleDownload)
	http.HandleFunc("/detail", handleDetail)

	fmt.Println("server started at localhost:9000")
	http.ListenAndServe(":9000", nil)
}

func routeIndexGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	var tmpl = template.Must(template.ParseFiles("view.html"))
	var err = tmpl.Execute(w, nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func routeSubmitPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// Method ParseMultipartForm() digunakan untuk mem-parsing form data yang ada data file nya.
	// Argumen 1024 pada method tersebut adalah maxMemory. Pemanggilan method tersebut membuat
	// file yang terupload disimpan sementara pada memory dengan alokasi adalah sesuai dengan maxMemory.
	// Jika ternyata kapasitas yang sudah dialokasikan tersebut tidak cukup, maka file akan disimpan dalam temporary file.
	if err := r.ParseMultipartForm(1024); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// code to get data alias dan file

	// Statement r.FormFile("file") digunakan untuk mengambil file yg di upload, mengembalikan 3 objek:
	// - Objek bertipe multipart.File (yang merupakan turunan dari *os.File)
	// - Informasi header file (bertipe *multipart.FileHeader)
	// - Dan error jika ada
	alias := r.FormValue("alias")

	uploadedFile, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer uploadedFile.Close()

	dir, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Jika inputan alias di-isi, maka nama nilai inputan tersebut dijadikan sebagai nama file.

	filename := handler.Filename
	if alias != "" {
		// Fungsi filepath.Ext digunakan untuk mengambil ekstensi dari sebuah file. Pada kode di atas, handler.Filename yang berisi nama file terupload diambil ekstensinya, lalu digabung dengan alias yang sudah terisi.
		filename = fmt.Sprintf("%s%s", alias, filepath.Ext(handler.Filename))
	}

	// Fungsi filepath.Join berguna untuk pembentukan path.
	fileLocation := filepath.Join(dir, "files", filename)
	// Fungsi os.OpenFile digunakan untuk membuka file. Fungsi ini membutuhkan 3 buah parameter:
	// - Parameter pertama merupakan path atau lokasi dari file yang ingin di buka
	// - Parameter kedua adalah flag mode, apakah read only, write only, atau keduanya, atau lainnya.
	//   - os.O_WRONLY|os.O_CREATE maknanya, file yang dibuka hanya akan bisa di tulis saja (write only konsantanya adalah os.O_WRONLY),
	//   - dan file tersebut akan dibuat jika belum ada (konstantanya os.O_CREATE).
	// - Sedangkan parameter terakhir adalah permission dari file, yang digunakan dalam pembuatan file itu sendiri.
	targetFile, err := os.OpenFile(fileLocation, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()

	// Fungsi io.Copy akan mengisi konten file parameter pertama (targetFile) dengan isi parameter kedua (uploadedFile).
	// File kosong yang telah kita buat tadi akan diisi dengan data file yang tersimpan di memory.
	if _, err := io.Copy(targetFile, uploadedFile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("done"))
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {

	files := []M{}
	// 	Fungsi os.Getwd() mengembalikan informasi absolute path di mana aplikasi di-eksekusi.
	// Path tersebut kemudian di gabung dengan folder bernama files lewat fungsi filepath.Join.
	basePath, _ := os.Getwd()
	// Fungsi filepath.Join akan menggabungkan item-item dengan path separator sesuai dengan
	// sistem operasi di mana program dijalankan. \ untuk Windows dan / untuk Linux/Unix.
	filesLocation := filepath.Join(basePath, "files")

	// Fungsi filepath.Walk berguna untuk membaca isi dari sebuah direktori, apa yang ada di
	// dalamnya (file maupun folder) akan di-loop. Dengan memanfaatkan callback parameter kedua
	// fungsi ini (yang bertipe filepath.WalkFunc), kita bisa mengamil informasi tiap item satu-per satu.
	err := filepath.Walk(filesLocation, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		files = append(files, M{"filename": info.Name(), "path": path})
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res, err := json.Marshal(files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path := r.FormValue("path")
	f, err := os.Open(path)
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Content-Disposition adalah salah satu ekstensi MIME protocol, berguna untuk menginformasikan browser
	// bagaimana dia harus berinteraksi dengan output. Ada banyak jenis value content-disposition,
	// salah satunya adalah attachment. Pada kode di atas, header Content-Disposition: attachment; filename=filename.json
	// menghasilkan output response berupa attachment atau file, yang kemudian akan di-download oleh browser.
	contentDisposition := fmt.Sprintf("attachment; filename=%s", f.Name())
	w.Header().Set("Content-Disposition", contentDisposition)

	// Objek file yang direpresentasikan variabel f, isinya di-copy ke objek response lewat statement io.Copy(w, f).
	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleDetail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Println("asds")

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path := r.FormValue("path")

	response_type := r.URL.Query().Get("response_type")

	if response_type == "base64" {
		// return a base64
		bytes, err := ioutil.ReadFile(strings.Split(path, "?")[0])
		if err != nil {
			log.Fatal(err)
		}

		var base64Encoding string

		// // Determine the content type of the image file
		// mimeType := http.DetectContentType(bytes)

		// // Prepend the appropriate URI scheme header depending
		// // on the MIME type
		// switch mimeType {
		// case "image/jpeg":
		// 	base64Encoding += "data:image/jpeg;base64,"
		// case "image/png":
		// 	base64Encoding += "data:image/png;base64,"
		// }

		// Append the base64 encoded output
		base64Encoding += toBase64(bytes)

		res, err := json.Marshal(base64Encoding)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	} else {
		// return a file
		f, err := os.Open(path)
		if f != nil {
			defer f.Close()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Objek file yang direpresentasikan variabel f, isinya di-copy ke objek response lewat statement io.Copy(w, f).
		if _, err := io.Copy(w, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}

func toBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
