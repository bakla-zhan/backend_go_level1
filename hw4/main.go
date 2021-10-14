package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

type Employee struct {
	Name   string  `json:"name" xml:"name"`
	Age    int     `json:"age" xml:"age"`
	Salary float32 `json:"salary" xml:"salary"`
}

type BaseHandler struct{}

type UploadHandler struct {
	HostAddr  string
	UploadDir string
}

type FileListHandler struct {
	DirToServe http.Dir
}

func (h *BaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := r.FormValue("name")
		fmt.Fprintf(w, "Parsed query-param with key \"name\": %s", name)
	case http.MethodPost:
		defer r.Body.Close()

		var employee Employee

		contentType := r.Header.Get("Content-Type")

		switch contentType {
		case "application/json":
			err := json.NewDecoder(r.Body).Decode(&employee)
			if err != nil {
				http.Error(w, "Unable to unmarshal JSON", http.StatusBadRequest)
				return
			}
		case "application/xml":
			err := xml.NewDecoder(r.Body).Decode(&employee)
			if err != nil {
				http.Error(w, "Unable to unmarshal XML", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "Unknown content type", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "Got a new employee!\nName: %s\nAge: %dy.o.\nSalary %0.2f\n",
			employee.Name,
			employee.Age,
			employee.Salary,
		)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
}

func (u *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Unable to read file", http.StatusBadRequest)
		return
	}

	filePath := path.Join(u.UploadDir, header.Filename)

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}
	fileLink := fmt.Sprint(u.HostAddr, "/", header.Filename)

	req, err := http.NewRequest(http.MethodHead, fileLink, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to check file", http.StatusInternalServerError)
		return
	}

	cli := &http.Client{}

	resp, err := cli.Do(req)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to check file", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Println(err)
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, fileLink)
}

// относится к заданиям 1 и 2
func (f *FileListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		flusher, ok := w.(http.Flusher)
		if !ok {
			panic("expected http.ResponseWriter to be an http.Flusher")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")

		dir, err := f.DirToServe.Open(".")
		if err != nil {
			log.Println(err)
			http.Error(w, "Unable to open directory", http.StatusInternalServerError)
			return
		}
		defer dir.Close()

		files, err := dir.Readdir(-1)
		if err != nil {
			log.Println(err)
			http.Error(w, "Unable to open files", http.StatusInternalServerError)
			return
		}

		if ext := r.FormValue("ext"); ext != "" {
			for idx, file := range files {
				if filepath.Ext(file.Name()) == fmt.Sprint(".", ext) {
					fmt.Fprintf(w, "file %d:\nname: %s\nextension: %s\nsize: %d\n\n", idx+1, file.Name(), filepath.Ext(file.Name()), file.Size())
					flusher.Flush()
				}
			}
			return
		}
		for idx, file := range files {
			fmt.Fprintf(w, "file %d:\nname: %s\nextension: %s\nsize: %d\n\n", idx+1, file.Name(), filepath.Ext(file.Name()), file.Size())
			flusher.Flush()
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
} // относится к заданиям 1 и 2

func main() {
	baseHandler := &BaseHandler{}
	http.Handle("/", baseHandler)

	uploadHandler := &UploadHandler{
		HostAddr:  "http://localhost:8081",
		UploadDir: "upload",
	}
	http.Handle("/upload", uploadHandler)

	dirToServe := http.Dir(uploadHandler.UploadDir)

	fileListHandler := &FileListHandler{
		DirToServe: dirToServe,
	}
	http.Handle("/filelist", fileListHandler)

	fs := &http.Server{
		Addr:         ":8081",
		Handler:      http.FileServer(dirToServe),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("File server started on port", fs.Addr)
		log.Fatalln(fs.ListenAndServe())
	}()

	log.Println("HTTP server started on port", srv.Addr)
	log.Fatalln(srv.ListenAndServe())
}
