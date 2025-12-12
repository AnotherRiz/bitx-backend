package main

import (
	"log"
	"net/http"

	"bitx-backend/internal/api"
	"bitx-backend/internal/db"
	"bitx-backend/internal/logx"
	"bitx-backend/internal/ratelimit"
)

func main() {
	logx.Init("data/bitx.log")
	db.Init("data/transfer.db")

	http.HandleFunc("/export", ratelimit.Export(api.Export))
	http.HandleFunc("/import", api.Import)
	http.HandleFunc("/health", api.Health)

	log.Println("Listening on :8844")
	log.Fatal(http.ListenAndServe(":8844", nil))
}
