package login

import (
	"context"
	"testing"
)

type verifierStub struct {
	valid bool
}

func (v verifierStub) VerifyOTP(context.Context, string, string) (bool, error) {
	return v.valid, nil
}

func TestDecideAccess(t *testing.T) {
	tests := []struct {
		name       string
		validCode  bool
		moderation []ModerationItem
		allowed    bool
		reason     string
	}{
		{name: "verified creator enters", validCode: true, allowed: true, reason: "verified"},
		{name: "bad code stays out", reason: "code_rejected"},
		{name: "blocked queue item holds login", validCode: true, moderation: []ModerationItem{{ID: "mod-7", Decision: "blocked"}}, reason: "moderation_hold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := PlayerContext{PlayerID: "player-42", ModerationQueue: tt.moderation}
			got, err := DecideAccess(context.Background(), verifierStub{valid: tt.validCode}, "+15551234567", "123456", player)
			if err != nil {
				t.Fatal(err)
			}
			if got.Allowed != tt.allowed || got.Reason != tt.reason {
				t.Fatalf("decision = %#v, want allowed=%v reason=%q", got, tt.allowed, tt.reason)
			}
		})
	}
}
