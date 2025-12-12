package logx

import (
	"log"
	"net/http"
	"os"
)

func Init(logFile string) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(file)
	}
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func Export(r *http.Request, status string) {
	log.Printf("EXPORT ip=%s status=%s", clientIP(r), status)
}

func Import(r *http.Request, status, reason string) {
	if reason != "" {
		log.Printf("IMPORT ip=%s status=%s reason=%s", clientIP(r), status, reason)
	} else {
		log.Printf("IMPORT ip=%s status=%s", clientIP(r), status)
	}
}
