package domain

import (
	"time"

	"github.com/google/uuid"
)

type ViolationType string

const (
	ViolationProfanity  ViolationType = "profanity"
	ViolationToxicity   ViolationType = "toxicity"
	ViolationNsfwText   ViolationType = "nsfw_text"
	ViolationPolitical  ViolationType = "political"
	ViolationHateSpeech ViolationType = "hate_speech"
)

func (v ViolationType) IsValid() bool {
	switch v {
	case ViolationProfanity, ViolationToxicity, ViolationNsfwText,
		ViolationPolitical, ViolationHateSpeech:
		return true
	}
	return false
}

type ModerationViolation struct {
	ID             uuid.UUID
	ServerID       uuid.UUID
	UserID         uuid.UUID
	MessageID      *uuid.UUID
	MessageContent *string
	ViolationType  ViolationType
	ActionTaken    ModerationAction
	CreatedAt      time.Time
}

func NewModerationViolation(
	serverID, userID uuid.UUID,
	messageID *uuid.UUID,
	messageContent *string,
	violationType ViolationType,
	actionTaken ModerationAction,
) *ModerationViolation {
	return &ModerationViolation{
		ID:             uuid.New(),
		ServerID:       serverID,
		UserID:         userID,
		MessageID:      messageID,
		MessageContent: messageContent,
		ViolationType:  violationType,
		ActionTaken:    actionTaken,
		CreatedAt:      time.Now(),
	}
}

type ModerationRequest struct {
	Text     string
	UserID   uuid.UUID
	ServerID uuid.UUID
}

type ModerationResult struct {
	Allowed    bool
	Violations []Violation
	Fallback   bool
}

type Violation struct {
	Type    ViolationType
	Message string
}

func NewModerationResult(allowed bool, violations ...Violation) ModerationResult {
	return ModerationResult{
		Allowed:    allowed,
		Violations: violations,
	}
}
