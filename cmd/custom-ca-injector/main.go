package main

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/radudd/custom-ca-inject/pkg/mutate"
)

func handleMutate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	mutated, err := mutate.Mutate(body)
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(mutated)
}

func main() {
	http.HandleFunc("/mutate", handleMutate)
	log.Fatal(http.ListenAndServeTLS(":8443", "/ssl/tls.crt", "/ssl/tls.key", nil))
}
