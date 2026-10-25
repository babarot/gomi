// Command serve serves docs/ on localhost the way GitHub Pages serves
// gomi.dev: a path with nothing there gets docs/404.html with status 404,
// and /install is hack/install, which the Pages workflow copies in when it
// deploys. Nothing is cached, so an edit shows on the next reload.
//
// Run it from the root of the repository: mise run serve.
package main

import (
	"errors"
	"flag"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
)

func main() {
	addr := flag.String("addr", "localhost:4664", "the address to listen on; 4664 is gomi on a phone keypad")
	flag.Parse()

	docs := os.DirFS("docs")
	if _, err := fs.Stat(docs, "index.html"); err != nil {
		log.Fatalf("docs/index.html: %v; run this from the root of the repository", err)
	}
	files := http.FileServerFS(docs)

	mux := http.NewServeMux()
	mux.HandleFunc("/install", func(w http.ResponseWriter, r *http.Request) {
		noCache(w)
		// text/plain rather than the text/html Pages sends, so that a
		// browser shows the script; curl | bash runs it either way
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeFile(w, r, "hack/install")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		noCache(w)
		if !exists(docs, r.URL.Path) {
			notFound(w, docs)
			return
		}
		files.ServeHTTP(w, r)
	})

	// take the port before saying where it serves, so that a port in use
	// is the only thing it says
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("%v; another server may be running there, or pass -addr HOST:PORT", err)
	}
	log.Printf("serving docs/ at http://%s", ln.Addr())
	log.Fatal(http.Serve(ln, mux))
}

func noCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

// exists reports whether Pages would serve something at p: a file, or a
// directory with an index.html
func exists(docs fs.FS, p string) bool {
	name := strings.TrimPrefix(path.Clean("/"+p), "/")
	if name == "" {
		name = "."
	}
	fi, err := fs.Stat(docs, name)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		_, err = fs.Stat(docs, path.Join(name, "index.html"))
		return err == nil
	}
	return true
}

func notFound(w http.ResponseWriter, docs fs.FS) {
	page, err := fs.ReadFile(docs, "404.html")
	if errors.Is(err, fs.ErrNotExist) {
		http.NotFound(w, nil)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if _, err := w.Write(page); err != nil {
		log.Printf("404.html: %v", err)
	}
}
