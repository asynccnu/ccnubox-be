package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-library/tool"
	commontool "github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const invitationLimiterCapacity = 10000

type invitationRequestWindow struct {
	started time.Time
	count   int
}

type invitationState struct {
	mu       sync.Mutex
	windows  map[string]invitationRequestWindow
	inFlight chan struct{}
	// 仅记录本进程实际追平时间，不能用数据库非零游标替代初始化就绪。
	preferenceCaughtUp atomic.Int64
}

func newInvitationState(concurrency int) *invitationState {
	return &invitationState{windows: map[string]invitationRequestWindow{}, inFlight: make(chan struct{}, concurrency)}
}

func (s *invitationState) allow(operator string, now time.Time, limit int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, window := range s.windows {
		if !now.Before(window.started.Add(time.Minute)) {
			delete(s.windows, key)
		}
	}
	window, exists := s.windows[operator]
	if !exists {
		if len(s.windows) >= invitationLimiterCapacity {
			return false
		}
		window.started = now
	}
	if window.count >= limit {
		return false
	}
	window.count++
	s.windows[operator] = window
	return true
}

func (s *ReminderService) invitationAvailable() bool {
	return s.Enabled() && s.notificationEnabled(NotificationTeamInvitation) && !s.config.IsDryRun() && s.config.TeamInvitation.ContractVerified
}

func (s *ReminderService) invitationPreferencesReady() bool {
	last := s.invitation.preferenceCaughtUp.Load()
	now := s.now()
	return last > 0 && !now.Before(time.Unix(0, last)) && now.Sub(time.Unix(0, last)) <= s.config.TeamInvitation.PreferenceMaxLag
}

// NotifyTeamInvitation 只等待学校校验和本地事务，不等待 Feed/Kafka/JPush。
func (s *ReminderService) NotifyTeamInvitation(parent context.Context, operator, teamID string, recipients []string) (resultErr error) {
	result := "accepted"
	defer func() {
		if s.metrics == nil {
			return
		}
		switch status.Code(resultErr) {
		case codes.InvalidArgument:
			result = "invalid"
		case codes.PermissionDenied:
			result = "forbidden"
		case codes.FailedPrecondition:
			result = "conflict"
		case codes.ResourceExhausted:
			result = "rate_limited"
		case codes.Unavailable:
			result = "unavailable"
		case codes.Internal:
			result = "db_error"
		}
		s.metrics.InvitationRequestsTotal.WithLabelValues(result).Inc()
	}()
	if !commontool.IsValidStudentID(operator) {
		return status.Error(codes.InvalidArgument, "invalid operator")
	}
	if !s.invitationAvailable() {
		return status.Error(codes.Unavailable, "team invitation unavailable")
	}
	cfg := s.config.TeamInvitation
	if !s.invitation.allow(operator, s.now(), cfg.RequestsPerMinute) {
		return status.Error(codes.ResourceExhausted, "invitation request limit reached")
	}
	recipients, err := commontool.NormalizeTeamInvitation(operator, teamID, recipients, cfg.MaxRecipients)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid invitation parameters")
	}
	ctx, cancel := context.WithTimeout(parent, cfg.RequestTimeout)
	defer cancel()
	select {
	case s.invitation.inFlight <- struct{}{}:
		defer func() { <-s.invitation.inFlight }()
	default:
		return status.Error(codes.ResourceExhausted, "invitation concurrency limit reached")
	}
	repo := s.dao.InvitationDAO()
	existing, err := repo.Invitations(ctx, teamID, recipients, false)
	if err != nil {
		return status.Error(codes.Internal, "invitation lookup failed")
	}
	pending, err := pendingInvitationRecipients(operator, recipients, existing)
	if err != nil || len(pending) == 0 {
		result = "duplicate"
		return err
	}
	if !s.invitationPreferencesReady() {
		return status.Error(codes.Unavailable, "invitation preferences not ready")
	}
	token, err := s.discussionToken(ctx, operator)
	if err != nil {
		return status.Error(codes.Unavailable, "school authentication unavailable")
	}
	team, err := s.crawler.GetCurrentTeamForInvitation(ctx, token)
	if err != nil {
		return status.Error(codes.Unavailable, "school invitation validation unavailable")
	}
	if team == nil || team.ID != teamID {
		return status.Error(codes.FailedPrecondition, "team is no longer current")
	}
	if team.OperatorStudentID != operator || !team.IsMasterUser {
		return status.Error(codes.PermissionDenied, "team operator mismatch")
	}
	observed := s.now().Truncate(time.Second)
	if team.Status != 0 || team.ExpirationTime == nil || !team.ExpirationTime.After(s.now()) {
		return status.Error(codes.FailedPrecondition, "team is not inviting")
	}
	for _, memberStatus := range team.Members {
		if !slices.Contains(cfg.KnownMemberStatuses, memberStatus) {
			return status.Error(codes.Unavailable, "unknown school member status")
		}
	}
	for _, recipient := range pending {
		memberStatus, exists := team.Members[recipient]
		if !exists || !slices.Contains(cfg.PendingMemberStatuses, memberStatus) {
			return status.Error(codes.FailedPrecondition, "recipient is not pending confirmation")
		}
	}
	expires := observed.Add(cfg.MaxDeliveryAge)
	if team.ExpirationTime.Before(expires) {
		expires = *team.ExpirationTime
	}
	expires = expires.Truncate(time.Second)
	// 固定首次校验时间和截止；事务重试不能延长投递时效。
	for attempt := 0; attempt < 3; attempt++ {
		err = repo.Transaction(ctx, func(tx *dao.ReminderDAO) error {
			subs := make(map[string]*dao.LibraryReminderSubscription, len(recipients))
			// 先按学号加订阅行锁，再查邀请事实，序列化同一收件人的日额度。
			for _, recipient := range recipients {
				sub, err := tx.SubscriptionForUpdate(ctx, recipient)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				subs[recipient] = sub
			}
			rows, err := tx.Invitations(ctx, teamID, recipients, true)
			if err != nil {
				return err
			}
			fresh, err := pendingInvitationRecipients(operator, recipients, rows)
			if err != nil || len(fresh) == 0 {
				return err
			}
			if !s.invitationAvailable() || !s.invitationPreferencesReady() {
				return status.Error(codes.Unavailable, "team invitation unavailable")
			}
			if !expires.After(s.now()) || !expires.After(observed) {
				return status.Error(codes.FailedPrecondition, "invitation expired")
			}
			now := s.now()
			local := now.In(tool.GetLocation())
			day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
			for _, recipient := range fresh {
				key := invitationDedupeKey(recipient, teamID)
				row := &dao.LibraryTeamInvitation{TeamID: teamID, RecipientStudentID: recipient, OperatorStudentID: operator, DedupeKey: key, ObservedAt: observed, ExpiresAt: expires, Disposition: "suppressed", CreatedAt: now, UpdatedAt: now}
				sub := subs[recipient]
				var outbox *dao.NotificationOutbox
				if sub == nil || !sub.Enabled {
					row.SuppressedReason = "subscription_disabled"
				} else {
					count, err := tx.InvitationDailyCount(ctx, recipient, day, day.AddDate(0, 0, 1))
					if err != nil {
						return err
					}
					if count >= int64(cfg.RecipientDailyLimit) {
						row.SuppressedReason = "recipient_limit"
					} else {
						row.Disposition = "enqueued"
						payload := notificationPayload{NotificationType: NotificationTeamInvitation, TeamID: teamID, OnDate: team.OnDate, TargetAt: observed.Unix(), InvitationExpiresAt: expires.Unix()}
						raw, err := json.Marshal(payload)
						if err != nil {
							return err
						}
						outbox = &dao.NotificationOutbox{DedupeKey: key, StudentID: recipient, PreferenceVersion: sub.PreferenceVersion, Type: NotificationTeamInvitation, Payload: raw, Status: dao.OutboxPending, NextAttemptAt: now, ExpiresAt: &expires}
					}
				}
				if err := tx.CreateInvitation(ctx, row, outbox); err != nil {
					return err
				}
			}
			return nil
		})
		if err == nil {
			return nil
		}
		if status.Code(err) != codes.Unknown {
			return err
		}
		if !retryInvitationTransaction(err) || attempt == 2 {
			break
		}
		select {
		case <-ctx.Done():
			return status.Error(codes.Unavailable, "invitation request timed out")
		case <-time.After(time.Duration(attempt+1) * 20 * time.Millisecond):
		}
	}
	return status.Error(codes.Internal, "invitation transaction failed")
}

func pendingInvitationRecipients(operator string, recipients []string, rows []dao.LibraryTeamInvitation) ([]string, error) {
	existing := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.OperatorStudentID != operator {
			return nil, status.Error(codes.PermissionDenied, "invitation operator mismatch")
		}
		existing[row.RecipientStudentID] = true
	}
	var pending []string
	for _, recipient := range recipients {
		if !existing[recipient] {
			pending = append(pending, recipient)
		}
	}
	return pending, nil
}

func invitationDedupeKey(recipient, teamID string) string {
	// JSON 元组无拼接歧义，学号含分隔符也不会碰撞。
	raw, _ := json.Marshal([2]string{recipient, teamID})
	sum := sha256.Sum256(raw)
	return boundedDedupeKey("library:team_invitation:" + hex.EncodeToString(sum[:]))
}

func retryInvitationTransaction(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &mysqlErr) && (mysqlErr.Number == 1062 || mysqlErr.Number == 1213 || mysqlErr.Number == 1205))
}
