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
	BookType        string
	CreatedByUserID int64
	EndedAt         *time.Time
	InviteCode      string
	ShareDisabled   bool
}

type Member struct {
	ID          string
	ScorebookID string
	UserID      int64
	Role        string
	Nickname    string
	AvatarURL   string
	JoinedAt    time.Time
	UpdatedAt   time.Time
}

type MemberWithScore struct {
	Member
	Score float64
}

type ScoreRecord struct {
	ID           string
	ScorebookID  string
	FromMemberID string
	ToMemberID   string
	Delta        float64
	Note         string
	CreatedAt    time.Time
}

type ScorebookListItem struct {
	ScorebookID  string
	Name         string
	LocationText string
	StartTime    time.Time
	UpdatedAt    time.Time
	Status       string
	BookType     string
	EndedAt      *time.Time
	InviteCode   string
	MyMemberID   string
	MyRole       string
	MemberCount  int64
}

type LedgerMember struct {
	ID        string
	LedgerID  string
	UserID    *int64
	Role      string
	Nickname  string
	AvatarURL string
	Remark    string
	Score     float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LedgerRecord struct {
	ID           string
	LedgerID     string
	FromMemberID string
	ToMemberID   string
	MemberID     string
	Type         string
	Amount       float64
	Note         string
	CreatedAt    time.Time
}

type LedgerListItem struct {
	LedgerID    string
	Name        string
	StartTime   time.Time
	UpdatedAt   time.Time
	Status      string
	EndedAt     *time.Time
	MemberCount int64
	RecordCount int64
}

type InviteInfo struct {
	BookID   string
	BookType string
	Name     string
	Status   string
	ShareDisabled bool
	UpdatedAt time.Time
}
