package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rwestlund/gotex"
)

var m sync.Mutex

func main() {

	handleRequests()
}

func compileLatex(w http.ResponseWriter, r *http.Request, document string) {

	var pdf, err = gotex.Render(document, gotex.Options{
		Command:   "/usr/bin/pdflatex",
		Runs:      2,
		Texinputs: ""})

	if err != nil {
		log.Println("render failed ", err)
	} else {
		if err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(pdf)
		return
	}

}

func generatePdfFromLatex(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("r.Body", string(body))

	values, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	docuementContent := values.Get("content")
	compileLatex(w, r, docuementContent)
}

/*func handleArgument(w http.ResponseWriter, r *http.Request) {

	m.Lock()

	dt := time.Now().String()

	for i := 0; i < 100000; i++ {
		log.Println(dt)
	}

	m.Unlock()

	return

}*/

func handleArgument(w http.ResponseWriter, r *http.Request) {

	return

}

func handleRequests() {

	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", generatePdfFromLatex).Methods("POST")
	myRouter.HandleFunc("/file", receiveFile)
	myRouter.HandleFunc("/writeFile", controlLaunch)
	myRouter.HandleFunc("/test", handleArgument)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	fmt.Printf("Starting server on port %s \n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), myRouter))
}

func receiveFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // maxMemory 32MB
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, h, err := r.FormFile("upload")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("Succesfully received %s \n", h.Filename)

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return
	}

	mimeType := http.DetectContentType(buf.Bytes())

	log.Printf("This file is type %s\n", mimeType)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(buf.Bytes())
	return

}

func checkIfZip(file *multipart.File) bool {

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, *file); err != nil {
		return false
	}

	mimeType := http.DetectContentType(buf.Bytes())
	return mimeType == "application/zip"

}

func controlLaunch(w http.ResponseWriter, r *http.Request) {
	m.Lock()

	writeFile(w, r)

	m.Unlock()
}

func writeFile(w http.ResponseWriter, r *http.Request) {

	dir, err := os.Getwd()

	err = os.RemoveAll(dir + "/tmp")
	err = os.Mkdir("tmp", 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = r.ParseMultipartForm(64 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, h, err := r.FormFile("upload")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	/*if !checkIfZip(&file) {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("This file is not a zip")
		return
	}*/

	log.Printf("Succesfully received %s \n", h.Filename)

	dt := time.Now()

	hasher := md5.New()
	io.WriteString(hasher, dt.String())
	sum := hex.EncodeToString(hasher.Sum(nil))

	filename := sum + "_" + h.Filename

	tmpfile, err := os.Create("./tmp/" + filename)
	defer tmpfile.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(tmpfile, file)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unzipSource("tmp/"+filename, "./tmp/"+sum)

	content, err := ioutil.ReadFile("./tmp/" + sum + "/main.tex")
	if err != nil {
		fmt.Println("Err")
	}

	log.Println(dir)

	var pdf, errLatex = gotex.Render(string(content), gotex.Options{
		Command:   "/usr/bin/pdflatex",
		Runs:      2,
		Texinputs: dir + "/tmp/" + sum})

	if errLatex != nil {
		log.Println("render failed ", errLatex)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Print()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(pdf)
	return
}
