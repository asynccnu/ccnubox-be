package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/repository/dao"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"gorm.io/gorm"
)

const teamScanPageSize = 200

// ScanTeams 与座位预约筛选和座位基线完全独立。
func (s *ReminderService) ScanTeams(ctx context.Context) error {
	if !s.Enabled() || !s.notificationEnabled(NotificationTeamSuccess) {
		return nil
	}
	cutoff := s.now().Add(-s.config.TeamScanMinInterval)
	var afterID int64
	var afterAttempt *time.Time
	batch := &subscriptionBatchError{Groups: map[string]int{}}
	for {
		rows, err := s.dao.TeamSubscriptions(ctx, cutoff, afterAttempt, afterID, teamScanPageSize)
		if err != nil {
			// 查询失败或取消时也保留此前已扫描批次的统计。
			batch.Cause = err
			return batch
		}
		if len(rows) == 0 {
			break
		}
		afterID = rows[len(rows)-1].ID
		afterAttempt = rows[len(rows)-1].LastTeamScanAttemptAt
		if err := s.forEachSubscription(ctx, rows, s.scanTeamUser); err != nil {
			var page *subscriptionBatchError
			if !errors.As(err, &page) {
				batch.Cause = err
				return batch
			}
			batch.Total += page.Total
			batch.Launched += page.Launched
			batch.Succeeded += page.Succeeded
			batch.Failed += page.Failed
			batch.Canceled += page.Canceled
			for kind, count := range page.Groups {
				batch.Groups[kind] += count
			}
			// 样例上限属于整轮扫描，不能随分页数量增长。
			remaining := batchErrorSampleLimit - len(batch.Samples)
			batch.Samples = append(batch.Samples, page.Samples[:min(remaining, len(page.Samples))]...)
			if batch.Cause == nil {
				batch.Cause = page.Cause
			}
		} else {
			batch.Total += len(rows)
			batch.Launched += len(rows)
			batch.Succeeded += len(rows)
		}
		if err := ctx.Err(); err != nil {
			batch.Cause = err
			return batch
		}
		if len(rows) < teamScanPageSize {
			break
		}
	}
	if batch.Cause != nil {
		return batch
	}
	return nil
}

func (s *ReminderService) scanTeamUser(ctx context.Context, sub dao.LibraryReminderSubscription) (err error) {
	attempted := false
	defer func() {
		if !attempted || s.metrics == nil {
			return
		}
		result := "success"
		if err != nil {
			result = "error"
		}
		s.metrics.TeamScanUsersTotal.WithLabelValues(result).Inc()
	}()
	return s.runUserOperation(ctx, sub, "team_scan", s.config.TeamScanMinInterval, func(ctx context.Context, sub dao.LibraryReminderSubscription) error {
		if !s.Enabled() || !s.notificationEnabled(NotificationTeamSuccess) {
			return nil
		}
		if !attempted {
			ok, err := s.dao.MarkTeamAttempt(ctx, sub.StudentID, sub.PreferenceVersion, s.now())
			if err != nil {
				return stageFailure(ctx, err, "save_team_attempt", "database")
			}
			if !ok {
				return nil
			}
			attempted = true
		}
		return s.scanTeamUserAttempt(ctx, sub)
	})
}

func (s *ReminderService) scanTeamUserAttempt(ctx context.Context, sub dao.LibraryReminderSubscription) error {
	current, err := s.subscriptionStillCurrent(ctx, sub)
	if err != nil || !current {
		return err
	}
	token, err := s.discussionToken(ctx, sub.StudentID)
	if err != nil {
		return stageFailure(ctx, err, "get_discussion_token", "token_rpc")
	}
	current, err = s.subscriptionStillCurrent(ctx, sub)
	if err != nil || !current {
		return err
	}
	team, err := s.crawler.GetCurrentTeam(ctx, token)
	if err != nil {
		return stageFailure(ctx, err, "get_current_team", upstreamFailureKind(err))
	}
	now := s.now()
	err = s.dao.Transaction(ctx, func(tx *dao.ReminderDAO) error {
		locked, err := tx.SubscriptionForUpdate(ctx, sub.StudentID)
		if err != nil {
			return err
		}
		if !locked.Enabled || locked.PreferenceVersion != sub.PreferenceVersion || !s.notificationEnabled(NotificationTeamSuccess) {
			return nil
		}
		txService := *s
		txService.dao = tx
		return txService.reconcileTeam(ctx, *locked, team, now)
	})
	if err != nil {
		return stageFailure(ctx, err, "save_team_state", "database")
	}
	return nil
}

func (s *ReminderService) discussionToken(ctx context.Context, studentID string) (string, error) {
	callCtx, cancel := s.remoteCallContext(ctx)
	defer cancel()
	resp, err := s.user.GetLibraryDiscussionToken(callCtx, &userv1.GetLibraryTokenRequest{StudentId: studentID})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.GetToken() == "" {
		return "", errors.New("user service returned an empty discussion token")
	}
	return resp.GetToken(), nil
}

func (s *ReminderService) reconcileTeam(ctx context.Context, sub dao.LibraryReminderSubscription, team *crawler.ReminderTeam, now time.Time) error {
	if team != nil {
		row, err := s.dao.Team(ctx, sub.StudentID, team.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = &dao.LibraryTeamSnapshot{StudentID: sub.StudentID, TeamID: team.ID, FirstSeenAt: now}
		} else if err != nil {
			return err
		}
		row.Status, row.OnDate, row.Total, row.ExpirationTime, row.LastSeenAt = team.Status, team.OnDate, team.Total, team.ExpirationTime, now
		if team.Status == 1 && row.SuccessDisposition == "" {
			row.SuccessObservedAt = &now
			if !sub.TeamBaselineCompleted && s.config.ShouldBaselineOnEnable() {
				row.SuccessDisposition = "baseline"
			} else {
				key := boundedDedupeKey(fmt.Sprintf("library:team_success:%s:%s", sub.StudentID, team.ID))
				payload := notificationPayload{NotificationType: NotificationTeamSuccess, TeamID: team.ID, OnDate: team.OnDate, TargetAt: now.Unix()}
				if err := s.enqueuePayload(ctx, sub, key, NotificationTeamSuccess, payload); err != nil {
					return err
				}
				row.SuccessDisposition = "enqueued"
			}
		}
		if err := s.dao.SaveTeam(ctx, row); err != nil {
			return err
		}
	}
	return s.dao.MarkTeamScan(ctx, sub.StudentID, now)
}
