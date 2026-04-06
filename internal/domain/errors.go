package domain

import "errors"

var (
	ErrServerNotFound      = errors.New("server not found")
	ErrServerDeleted       = errors.New("server is deleted")
	ErrInvalidServerName   = errors.New("invalid server name")
	ErrServerOwnerRequired = errors.New("server owner is required")
	ErrNotServerOwner      = errors.New("only server owner can perform this action")
)

var (
	ErrConfigNotFound           = errors.New("server config not found")
	ErrModerationConfigNotFound = errors.New("moderation config not found")
	ErrInvalidModerationAction  = errors.New("invalid moderation action")
)

var (
	ErrMemberNotFound  = errors.New("server member not found")
	ErrAlreadyMember   = errors.New("user is already a member")
	ErrNotMember       = errors.New("user is not a member of this server")
	ErrMemberIsBanned  = errors.New("user is banned from this server")
	ErrMemberIsMuted   = errors.New("user is muted in this server")
	ErrCannotBanOwner  = errors.New("cannot ban server owner")
	ErrCannotKickOwner = errors.New("cannot kick server owner")
)

var (
	ErrRoleNotFound            = errors.New("server role not found")
	ErrRoleAlreadyExists       = errors.New("role with this name already exists")
	ErrCannotDeleteDefaultRole = errors.New("cannot delete default role")
	ErrInvalidRolePosition     = errors.New("invalid role position")
)

var (
	ErrModerationServiceUnavailable = errors.New("moderation service unavailable")
	ErrModerationTimeout            = errors.New("moderation check timeout")
	ErrMessageBlocked               = errors.New("message blocked by moderation")
)

var (
	ErrChatNotFound       = errors.New("chat not found")
	ErrChatAlreadyExists  = errors.New("chat already exists in this server")
	ErrMaxChannelsReached = errors.New("maximum number of channels reached")
)

var (
	ErrMaxMembersReached = errors.New("maximum number of members reached")
)
