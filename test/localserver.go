package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func runLocalServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("catched request!")
		log.Println("request headers:", r.Header)

		defer r.Body.Close()
		rb, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("read body err:", err)
		}

		log.Println("request body:", string(rb))

		w.Header().Add("X-Custom-Resp-Header-A", "A-value")
		w.Header().Add("X-Custom-Resp-Header-B", "B-value")
		fmt.Fprintf(w, "Hello, World!")
		log.Println()
	})

	sa := targetServerAddr
	if sa == "" {
		sa = DefaultTargetServerAddr
	}
	log.Println("Starting Target HTTP server on: ", sa)
	err := http.ListenAndServe(sa, nil)
	if err != nil {
		log.Fatal("Server error: ", err)
	}
}
