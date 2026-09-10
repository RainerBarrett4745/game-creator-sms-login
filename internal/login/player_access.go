package login

import "context"

type PlayerAsset struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Moderated bool  `json:"moderated"`
}

type LiveEvent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ModerationItem struct {
	ID       string `json:"id"`
	Decision string `json:"decision"`
}

type PlayerContext struct {
	PlayerID       string           `json:"player_id"`
	Assets         []PlayerAsset    `json:"assets"`
	LiveEvents     []LiveEvent      `json:"live_events"`
	ModerationQueue []ModerationItem `json:"moderation_queue"`
}

type AccessDecision struct {
	Allowed  bool   `json:"allowed"`
	PlayerID string `json:"player_id"`
	Reason   string `json:"reason"`
}

type OTPVerifier interface {
	VerifyOTP(context.Context, string, string) (bool, error)
}

func DecideAccess(ctx context.Context, verifier OTPVerifier, phone, code string, player PlayerContext) (AccessDecision, error) {
	valid, err := verifier.VerifyOTP(ctx, phone, code)
	if err != nil {
		return AccessDecision{}, err
	}
	if !valid {
		return AccessDecision{PlayerID: player.PlayerID, Reason: "code_rejected"}, nil
	}
	for _, item := range player.ModerationQueue {
		if item.Decision == "blocked" {
			return AccessDecision{PlayerID: player.PlayerID, Reason: "moderation_hold"}, nil
		}
	}
	return AccessDecision{Allowed: true, PlayerID: player.PlayerID, Reason: "verified"}, nil
}
