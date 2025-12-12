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
	codeTTL       = 15 * time.Minute
	maxPayloadLen = 100 * 1024
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

	if len(body.Payload) == 0 || len(body.Payload) > maxPayloadLen {
		logx.Export(r, "INVALID_PAYLOAD")
		http.Error(w, "invalid payload size", 400)
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
	var payload string

	err = tx.QueryRow(`
		SELECT id, payload FROM transfers
		WHERE code_hash = ?
		  AND used = 0
		  AND expires_at > ?`,
		hash, now,
	).Scan(&id, &payload)

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

	_, err = tx.Exec(`UPDATE transfers SET used = 1 WHERE id = ?`, id)
	if err != nil {
		logx.Import(r, "FAILED", "db_error")
		tx.Rollback()
		http.Error(w, "db error", 500)
		return
	}

	tx.Commit()

	util.WriteJSON(w, json.RawMessage(payload))
	logx.Import(r, "OK", "")
}

func Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
