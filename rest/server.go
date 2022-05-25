package main

import (
	"encoding/json"
	"github.com/hcfonseca/testing-grpc/pb"
	"log"
	"net/http"
)

func handle(w http.ResponseWriter, req *http.Request) {
	bytes, err := json.Marshal(&pb.Payload{Message: "Ola Mundo"})
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func main() {
	server := &http.Server{Addr: "localhost:8080", Handler: http.HandlerFunc(handle)}
	log.Fatal(server.ListenAndServeTLS("./cert/localhost.crt", "./cert/localhost.decrypted.key"))
	//log.Fatal(server.ListenAndServe())
}
