package main

import (
	"net/http"

	inbound_http "github.com/DENFNC/devPractice/internal/adapters/inbound/http"
)

func main() {

	mux := http.NewServeMux()
	server := inbound_http.NewServer(":8080", mux)
	server.Start()
}
