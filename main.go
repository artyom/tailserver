package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/artyom/httpgzip"
)

func main() {
	log.SetFlags(0)
	addr := "localhost:8080"
	var reload int
	flag.StringVar(&addr, "addr", addr, "address to listen")
	flag.IntVar(&reload, "r", reload, "reload page every N `seconds`")
	flag.Parse()
	if err := run(addr, reload, flag.Args()); err != nil {
		log.Fatal(err)
	}
}

func run(addr string, reload int, names []string) error {
	if len(names) == 0 {
		return errors.New("at least one file name required")
	}
	mux := http.NewServeMux()
	mux.Handle("GET /", httpgzip.New(&handler{names: names, reload: reload}))
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return srv.ListenAndServe()
}

type handler struct {
	names  []string
	reload int
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var out []byte
	for i, name := range h.names {
		if i != 0 {
			out = append(out, '\n')
		}
		out = append(out, "==> "...)
		out = append(out, name...)
		out = append(out, " <==\n"...)
		buf, err := tail(name)
		if err != nil {
			code := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				code = http.StatusNotFound
			}
			http.Error(w, err.Error(), code)
			return
		}
		out = append(out, buf...)
	}
	w.Header().Set("Content-Type", "text/plain")
	if h.reload > 0 {
		w.Header().Set("Refresh", strconv.Itoa(h.reload))
	}
	w.Write(out)
}

func tail(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	const tailSize = 1024
	pos, err := f.Seek(-tailSize, io.SeekEnd)
	if err != nil && !errors.Is(err, syscall.EINVAL) {
		return nil, err
	}
	buf := make([]byte, tailSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buf = buf[:n]
	if pos != 0 {
		if i := bytes.IndexByte(buf, '\n'); i != -1 {
			buf = buf[i+1:]
		}
	}
	if len(buf) != 0 && buf[len(buf)-1] != '\n' {
		buf = append(buf, '\n')
	}
	return buf, nil
}
