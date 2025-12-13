package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"bitx-backend/internal/db"
	"bitx-backend/internal/logx"
	"bitx-backend/internal/util"
)

const (
	codeTTL = 15 * time.Minute
)

func Export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var body struct {
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logx.Export(r, "INVALID_JSON")
		http.Error(w, "invalid json", 400)
		return
	}

	if len(body.Payload) == 0 {
		logx.Export(r, "INVALID_PAYLOAD")
		http.Error(w, "empty payload", 400)
		return
	}

	code := util.GenerateTransferCode()
	hash := util.HashCode(code)

	now := time.Now().Unix()
	expires := time.Now().Add(codeTTL).Unix()

	_, err := db.DB.Exec(`
		INSERT INTO transfers (code_hash, payload, expires_at, used, created_at)
		VALUES (?, ?, ?, 0, ?)`,
		hash, string(body.Payload), expires, now,
	)
	if err != nil {
		logx.Export(r, "DB_ERROR")
		http.Error(w, "failed to save", 500)
		return
	}

	util.WriteJSON(w, map[string]interface{}{
		"transfer_code": code,
		"expires_in":    int(codeTTL.Seconds()),
	})

	logx.Export(r, "OK")
}

func Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var body struct {
		TransferCode string `json:"transfer_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logx.Import(r, "FAILED", "invalid_json")
		http.Error(w, "invalid json", 400)
		return
	}

	hash := util.HashCode(body.TransferCode)
	now := time.Now().Unix()

	tx, err := db.DB.Begin()
	if err != nil {
		logx.Import(r, "FAILED", "db_error")
		http.Error(w, "db error", 500)
		return
	}

	var id int
	var payloadStr string

	err = tx.QueryRow(`
		SELECT id, payload FROM transfers
		WHERE code_hash = ?
		  AND used = 0
		  AND expires_at > ?`,
		hash, now,
	).Scan(&id, &payloadStr)

	if err == sql.ErrNoRows {
		logx.Import(r, "FAILED", "invalid_or_expired")
		tx.Rollback()
		http.Error(w, "invalid or expired code", 404)
		return
	}
	if err != nil {
		logx.Import(r, "FAILED", "db_error")
		tx.Rollback()
		http.Error(w, "db error", 500)
		return
	}

	// tandai sebagai used
	if _, err := tx.Exec(`UPDATE transfers SET used = 1 WHERE id = ?`, id); err != nil {
		logx.Import(r, "FAILED", "db_error")
		tx.Rollback()
		http.Error(w, "db error", 500)
		return
	}

	tx.Commit()

	// ===============================
	// 🔐 PAYLOAD NORMALIZATION
	// ===============================
	raw := []byte(payloadStr)

	// 1. harus JSON valid
	if !json.Valid(raw) {
		logx.Import(r, "FAILED", "invalid_payload")
		http.Error(w, "invalid payload format", 500)
		return
	}

	// 2. cek apakah payload adalah string JSON (double encoded)
	var maybeString string
	if err := json.Unmarshal(raw, &maybeString); err == nil {
		// payload ternyata string → decode ulang
		raw = []byte(maybeString)

		if !json.Valid(raw) {
			logx.Import(r, "FAILED", "invalid_nested_payload")
			http.Error(w, "invalid payload format", 500)
			return
		}
	}

	// 3. response konsisten
	util.WriteJSON(w, map[string]interface{}{
		"payload": json.RawMessage(raw),
	})

	logx.Import(r, "OK", "")
}

func Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
