package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/example/game-creator-sms-login/internal/login"
)

type server struct {
	sms *login.SMSClient
}

type sendRequest struct {
	Phone     string `json:"phone"`
	RequestID string `json:"request_id"`
}

type verifyRequest struct {
	Phone  string              `json:"phone"`
	Code   string              `json:"code"`
	Player login.PlayerContext `json:"player"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	s := &server{sms: login.NewSMSClient(key)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code", s.sendCode)
	mux.HandleFunc("POST /login/verify", s.verifyCode)
	log.Printf("game login listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) sendCode(w http.ResponseWriter, r *http.Request) {
	var in sendRequest
	if !decodeJSON(w, r, &in) || strings.TrimSpace(in.Phone) == "" || strings.TrimSpace(in.RequestID) == "" {
		if in.Phone == "" || in.RequestID == "" {
			http.Error(w, "phone and request_id are required", http.StatusBadRequest)
		}
		return
	}
	if err := s.sms.SendOTP(r.Context(), in.Phone, in.RequestID); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "code_sent"})
}

func (s *server) verifyCode(w http.ResponseWriter, r *http.Request) {
	var in verifyRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Phone == "" || in.Code == "" || in.Player.PlayerID == "" {
		http.Error(w, "phone, code, and player.player_id are required", http.StatusBadRequest)
		return
	}
	decision, err := login.DecideAccess(r.Context(), s.sms, in.Phone, in.Code, in.Player)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	status := http.StatusOK
	if !decision.Allowed {
		status = http.StatusForbidden
	}
	writeJSON(w, status, decision)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return false
	}
	return true
}

func writeAPIError(w http.ResponseWriter, err error) {
	var apiErr *login.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		writeJSON(w, apiErr.HTTPStatus, map[string]any{"ok": false, "error": apiErr})
		return
	}
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
