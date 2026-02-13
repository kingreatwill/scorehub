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
	BookID        string
	BookType      string
	Name          string
	Status        string
	ShareDisabled bool
	UpdatedAt     time.Time
}

type BirthdayContact struct {
	ID            string
	UserID        int64
	Name          string
	Gender        string
	Phone         string
	Relation      string
	Note          string
	AvatarURL     string
	SolarBirthday *time.Time
	LunarBirthday string
	PrimaryType   string
	PrimaryMonth  int
	PrimaryDay    int
	PrimaryYear   int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BirthdayContactWithDays struct {
	BirthdayContact
	NextBirthday time.Time
	DaysLeft     int
}

type BirthdayContactInput struct {
	Name          string
	Gender        string
	Phone         string
	Relation      string
	Note          string
	AvatarURL     string
	SolarBirthday *time.Time
	LunarBirthday string
	PrimaryType   string
	PrimaryMonth  int
	PrimaryDay    int
	PrimaryYear   int
}

type BirthdayContactUpdate struct {
	Name          *string
	Gender        *string
	Phone         *string
	Relation      *string
	Note          *string
	AvatarURL     *string
	SolarBirthday *time.Time
	SolarSetNull  bool
	LunarBirthday *string
	PrimaryType   *string
	PrimaryMonth  *int
	PrimaryDay    *int
	PrimaryYear   *int
}

type DepositAccount struct {
	ID        string
	UserID    int64
	Bank      string
	Branch    string
	AccountNo string
	Holder    string
	AvatarURL string
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type DepositAttachment struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Name string `json:"name,omitempty"`
}

type DepositRecord struct {
	ID          string
	UserID      int64
	AccountID   string
	Currency    string
	Amount      float64
	AmountUpper string
	TermValue   int
	TermUnit    string
	Rate        float64
	StartDate   time.Time
	EndDate     time.Time
	Interest    float64
	ReceiptNo   string
	Status      string
	WithdrawnAt *time.Time
	Tags        []string
	Attachments []DepositAttachment
	Note        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type DepositAccountInput struct {
	Bank      string
	Branch    string
	AccountNo string
	Holder    string
	AvatarURL string
	Note      string
}

type DepositAccountUpdate struct {
	Bank      *string
	Branch    *string
	AccountNo *string
	Holder    *string
	AvatarURL *string
	Note      *string
}

type DepositRecordInput struct {
	AccountID   string
	Currency    string
	Amount      float64
	AmountUpper string
	TermValue   int
	TermUnit    string
	Rate        float64
	StartDate   time.Time
	EndDate     time.Time
	Interest    float64
	ReceiptNo   string
	Status      string
	WithdrawnAt *time.Time
	Tags        []string
	Attachments []DepositAttachment
	Note        string
}

type DepositRecordUpdate struct {
	Currency         *string
	Amount           *float64
	AmountUpper      *string
	TermValue        *int
	TermUnit         *string
	Rate             *float64
	StartDate        *time.Time
	EndDate          *time.Time
	Interest         *float64
	ReceiptNo        *string
	Status           *string
	WithdrawnAt      *time.Time
	WithdrawnSetNull bool
	Tags             *[]string
	Attachments      *[]DepositAttachment
	Note             *string
}

type DepositCurrencyStat struct {
	Currency string
	Amount   float64
}

type DepositAccountStat struct {
	AccountID string
	Currency  string
	Amount    float64
}

type DepositStats struct {
	Totals        []DepositCurrencyStat
	AnnualYields  []DepositCurrencyStat
	AccountTotals []DepositAccountStat
}

type DepositTagCount struct {
	Tag   string
	Count int
}
