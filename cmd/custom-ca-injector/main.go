package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/radudd/custom-ca-inject/pkg/mutate"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
)

func handleMutate(w http.ResponseWriter, r *http.Request) {
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
	//w.WriteHeader(http.StatusOK)
	responsewriters.WriteRawJSON(http.StatusOK, nil, w)
	w.Write(mutated)
}

func main() {
	http.HandleFunc("/mutate", handleMutate)
	log.Fatal(http.ListenAndServeTLS(":8443", "/ssl/tls.crt", "/ssl/tls.key", nil))
}
