package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/radudd/custom-ca-inject/pkg/mutate"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
)

func handleMutate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			//log.Fatal(err)
			responsewriters.InternalError(w, r, fmt.Errorf("Failed to read body: %v", err))
			return
		}

		mutated, err := mutate.Mutate(body)
		if err != nil {
			//log.Fatal(err)
			responsewriters.InternalError(w, r, fmt.Errorf("Failed mutation: %v", err))
			return
		}
		responsewriters.WriteRawJSON(200, mutated, w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/mutate", handleMutate)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "UP"})
	})
	log.Fatal(http.ListenAndServeTLS(":8443", "/ssl/tls.crt", "/ssl/tls.key", nil))
}
