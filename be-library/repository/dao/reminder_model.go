package dao

import "time"

const (
	SubscriptionAuthUnknown         = "UNKNOWN"
	SubscriptionAuthOK              = "OK"
	SubscriptionAuthError           = "AUTH_ERROR"
	SubscriptionAuthUpstreamUnknown = "UPSTREAM_UNKNOWN"

	JobPending    = "pending"
	JobRunning    = "running"
	JobDone       = "done"
	JobCancelled  = "cancelled"
	JobSuppressed = "suppressed"

	OutboxPending    = "pending"
	OutboxSending    = "sending"
	OutboxSent       = "sent"
	OutboxFailed     = "failed"
	OutboxSuppressed = "suppressed"

	SuppressedReasonFeatureDisabled          = "feature disabled"
	SuppressedReasonNotificationTypeDisabled = "notification type disabled"

	AwayStateAway     = "AWAY"
	AwayStateReturned = "RETURNED"
	AwayStateEnded    = "ENDED"
)

type LibraryReminderSubscription struct {
	ID                int64  `gorm:"primaryKey;autoIncrement"`
	StudentID         string `gorm:"column:student_id;type:varchar(64);not null;uniqueIndex"`
	Enabled           bool   `gorm:"not null;default:false;index"`
	FeedRevision      int64  `gorm:"not null;default:0"`
	PreferenceVersion int64  `gorm:"not null;default:0"`
	BaselineCompleted bool   `gorm:"not null;default:false"`
	AuthStatus        string `gorm:"type:varchar(32);not null;default:UNKNOWN"`
	LastFullRefreshAt *time.Time
	LastActiveScanAt  *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (LibraryReminderSubscription) TableName() string { return "library_reminder_subscriptions" }

type LibraryPreferenceSyncCursor struct {
	ConsumerName string `gorm:"primaryKey;type:varchar(64)"`
	LastRevision int64  `gorm:"not null;default:0"`
	UpdatedAt    time.Time
}

func (LibraryPreferenceSyncCursor) TableName() string { return "library_preference_sync_cursor" }

type ReservationSnapshot struct {
	ID                    int64     `gorm:"primaryKey;autoIncrement"`
	StudentID             string    `gorm:"column:student_id;type:varchar(64);not null;uniqueIndex:uidx_reservation,priority:1;index:idx_reservation_window,priority:1"`
	ExternalReservationID string    `gorm:"type:varchar(128);not null;uniqueIndex:uidx_reservation,priority:2"`
	SeatID                string    `gorm:"type:varchar(128)"`
	SeatLabel             string    `gorm:"type:varchar(128)"`
	Location              string    `gorm:"type:varchar(512)"`
	MakeDate              string    `gorm:"type:char(10)"`
	StartAt               time.Time `gorm:"not null;index:idx_reservation_window,priority:2"`
	EndAt                 time.Time `gorm:"not null;index:idx_reservation_window,priority:3"`
	Status                string    `gorm:"type:varchar(32);index:idx_reservation_window,priority:4"`
	RawStatus             string    `gorm:"type:varchar(128)"`
	RawPayload            []byte    `gorm:"type:blob"`
	FirstSeenAt           time.Time `gorm:"not null"`
	LastSeenAt            time.Time `gorm:"not null"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (ReservationSnapshot) TableName() string { return "reservation_snapshots" }

type LibraryUserStateSnapshot struct {
	ID               int64  `gorm:"primaryKey;autoIncrement"`
	StudentID        string `gorm:"column:student_id;type:varchar(64);not null;uniqueIndex"`
	CycleTimeName    string `gorm:"type:varchar(128)"`
	BreachNum        int
	ScoreNum         int
	BlackTime        string `gorm:"type:varchar(255)"`
	BlackMessage     string `gorm:"type:text"`
	BlacklistEpisode int    `gorm:"not null;default:0"`
	LastSeenAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (LibraryUserStateSnapshot) TableName() string { return "library_user_state_snapshots" }

type AwayEpisode struct {
	ID                    int64  `gorm:"primaryKey;autoIncrement"`
	StudentID             string `gorm:"column:student_id;type:varchar(64);not null;uniqueIndex:uidx_away_episode,priority:1;index"`
	ExternalReservationID string `gorm:"type:varchar(128);not null;uniqueIndex:uidx_away_episode,priority:2"`
	EpisodeVersion        int    `gorm:"not null;uniqueIndex:uidx_away_episode,priority:3"`
	AwayStartedAt         time.Time
	LastAwayMinutes       int
	State                 string `gorm:"type:varchar(16);not null"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (AwayEpisode) TableName() string { return "away_episodes" }

type NotificationJob struct {
	ID                    int64  `gorm:"primaryKey;autoIncrement"`
	LogicalKey            string `gorm:"type:varchar(512);not null;uniqueIndex"`
	StudentID             string `gorm:"column:student_id;type:varchar(64);not null;index:idx_job_student_status,priority:1"`
	ExternalReservationID string `gorm:"type:varchar(128)"`
	EpisodeVersion        int
	PreferenceVersion     int64
	Type                  string     `gorm:"type:varchar(64);not null"`
	TargetAt              time.Time  `gorm:"index"`
	ExpiresAt             *time.Time `gorm:"index"`
	RunAt                 time.Time  `gorm:"not null;index:idx_job_due,priority:2"`
	Status                string     `gorm:"type:varchar(16);not null;index:idx_job_due,priority:1;index:idx_job_student_status,priority:2"`
	Version               int64      `gorm:"not null;default:1"`
	Attempts              int        `gorm:"not null;default:0"`
	LastError             string     `gorm:"type:text"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (NotificationJob) TableName() string { return "notification_jobs" }

type NotificationOutbox struct {
	ID                    int64  `gorm:"primaryKey;autoIncrement"`
	DedupeKey             string `gorm:"type:varchar(512);not null;uniqueIndex"`
	StudentID             string `gorm:"column:student_id;type:varchar(64);not null;index:idx_outbox_student_status,priority:1"`
	ExternalReservationID string `gorm:"type:varchar(128);index:idx_outbox_reservation"`
	PreferenceVersion     int64
	Type                  string     `gorm:"type:varchar(64);not null"`
	Payload               []byte     `gorm:"type:blob;not null"`
	Status                string     `gorm:"type:varchar(16);not null;index:idx_outbox_due,priority:1;index:idx_outbox_student_status,priority:2"`
	Attempts              int        `gorm:"not null;default:0"`
	NextAttemptAt         time.Time  `gorm:"not null;index:idx_outbox_due,priority:2"`
	ExpiresAt             *time.Time `gorm:"index"`
	LastError             string     `gorm:"type:text"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SentAt                *time.Time
}

func (NotificationOutbox) TableName() string { return "notification_outbox" }
