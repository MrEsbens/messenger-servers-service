package domain

import (
	"time"

	"github.com/google/uuid"
)

var SystemModeratorID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type ServerMember struct {
	ID         uuid.UUID
	ServerID   uuid.UUID
	UserID     uuid.UUID
	JoinedAt   time.Time
	IsMuted    bool
	IsBanned   bool
	MutedUntil *time.Time
}

func NewServerMember(serverID, userID uuid.UUID) (*ServerMember, error) {
	if serverID == uuid.Nil {
		return nil, ErrMemberNotFound
	}
	if userID == uuid.Nil {
		return nil, ErrMemberNotFound
	}

	return &ServerMember{
		ID:       uuid.New(),
		ServerID: serverID,
		UserID:   userID,
		JoinedAt: time.Now(),
		IsMuted:  false,
		IsBanned: false,
	}, nil
}

func (m *ServerMember) Ban() {
	m.IsBanned = true
	m.IsMuted = false
	m.MutedUntil = nil
}

func (m *ServerMember) Unban() {
	m.IsBanned = false
}

func (m *ServerMember) Mute(duration *time.Duration) {
	m.IsMuted = true
	if duration != nil {
		until := time.Now().Add(*duration)
		m.MutedUntil = &until
	}
}

func (m *ServerMember) Unmute() {
	m.IsMuted = false
	m.MutedUntil = nil
}

func (m *ServerMember) IsMuteExpired() bool {
	if !m.IsMuted || m.MutedUntil == nil {
		return true
	}
	return time.Now().After(*m.MutedUntil)
}

func (m *ServerMember) CanSendMessages() bool {
	if m.IsBanned {
		return false
	}
	if m.IsMuted && !m.IsMuteExpired() {
		return false
	}
	return true
}
