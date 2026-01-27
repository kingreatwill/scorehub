package store

import "time"

type User struct {
	ID              int64
	WeChatOpenID    string
	WeChatNickname  string
	WeChatAvatarURL string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Scorebook struct {
	ID              string
	Name            string
	LocationText    string
	StartTime       time.Time
	UpdatedAt       time.Time
	Status          string
	CreatedByUserID int64
	EndedAt         *time.Time
	InviteCode      string
}

type Member struct {
	ID         string
	ScorebookID string
	UserID     int64
	Role       string
	Nickname   string
	AvatarURL  string
	JoinedAt   time.Time
	UpdatedAt  time.Time
}

type MemberWithScore struct {
	Member
	Score int64
}

type ScoreRecord struct {
	ID           string
	ScorebookID  string
	FromMemberID string
	ToMemberID   string
	Delta        int64
	Note         string
	CreatedAt    time.Time
}

type ScorebookListItem struct {
	ScorebookID   string
	Name          string
	LocationText  string
	StartTime     time.Time
	UpdatedAt     time.Time
	Status        string
	EndedAt       *time.Time
	InviteCode    string
	MyMemberID    string
	MyRole        string
	MemberCount   int64
}

type InviteInfo struct {
	ScorebookID string
	Name        string
	Status      string
	UpdatedAt   time.Time
}

