package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&Comment{},
		&LibraryReminderSubscription{},
		&LibraryPreferenceSyncCursor{},
		&ReservationSnapshot{},
		&LibraryTeamSnapshot{},
		&LibraryTeamInvitation{},
		&LibraryUserStateSnapshot{},
		&AwayEpisode{},
		&NotificationJob{},
		&NotificationOutbox{},
	)
}
