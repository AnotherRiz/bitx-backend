package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

func GenerateTransferCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	parts := []string{"BITX"}

	for i := 0; i < 4; i++ {
		b := make([]byte, 4)
		rand.Read(b)

		var part strings.Builder
		for _, v := range b {
			part.WriteByte(chars[int(v)%len(chars)])
		}
		parts = append(parts, part.String())
	}

	return strings.Join(parts, "-")
}

func HashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

func WriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
