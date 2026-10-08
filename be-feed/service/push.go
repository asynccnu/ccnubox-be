package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

type pushService struct {
	pushClient        jpush.PushClient // 用于推送的客户端
	userFeedConfigDAO dao.FeedUserConfigDAO
	feedTokenDAO      dao.FeedTokenDAO
}

type PushService interface {
	GetPushCID(context.Context) (string, error)
	LoadPushTargets(ctx context.Context, studentIDs []string) (*PushTargets, error)
	PreparePush(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, error)
	PreparePushForDelivery(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, string, error)
	PushPreparedMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, prepared *PreparedPush, cid string) error
	PushMSG(ctx context.Context, pushData *domain.FeedEvent) error
	PushMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, cid string) error
	PushMSGS(ctx context.Context, pushDatas []domain.FeedEvent) []ErrWithData
}

// PreparedPush 保存已通过 Token 和推送开关检查的接收目标。
type PreparedPush struct {
	tokens []string
}

// PushTargets 仅在一个准备分组内共享，配置和 token 均按只读快照使用。
type PushTargets struct {
	users map[string]pushTarget
}

type pushTarget struct {
	config dao.PushConfigSnapshot
	tokens []string
}

func (targets *PushTargets) prepare(pushData *domain.FeedEvent) (*PreparedPush, string, error) {
	if !tool.IsValidStudentID(pushData.StudentId) {
		return nil, "suppressed_no_target_or_disabled", nil
	}
	target, exists := targets.users[pushData.StudentId]
	if !exists {
		// 未预取不等于配置缺失，不能据此使用默认允许规则。
		return nil, "", errorx.Errorf("service: push target not prefetched, sid: %s", pushData.StudentId)
	}
	prepared, reason := preparePushTarget(pushData.Type, target)
	return prepared, reason, nil
}

func preparePushTarget(label string, target pushTarget) (*PreparedPush, string) {
	if !pushAllowed(label, target.config) {
		return nil, suppressionReason(label)
	}
	if len(target.tokens) == 0 {
		return nil, "suppressed_no_target_or_disabled"
	}
	return &PreparedPush{tokens: target.tokens}, ""
}

type ErrWithData struct {
	FeedEvent *domain.FeedEvent `json:"feed_event"`
	Err       error             `json:"err"`
}

func NewPushService(
	pushClient jpush.PushClient,
	userFeedConfigDAO dao.FeedUserConfigDAO,
	feedTokenDAO dao.FeedTokenDAO,
) PushService {
	return &pushService{
		pushClient:        pushClient,
		userFeedConfigDAO: userFeedConfigDAO,
		feedTokenDAO:      feedTokenDAO,
	}
}

func (s *pushService) PushMSGS(ctx context.Context, pushDatas []domain.FeedEvent) []ErrWithData {
	errs := make([]ErrWithData, 0)
	concurrencyLimit := 10
	semaphore := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex // 保护 errs 切片的并发安全

	for _, pushData := range pushDatas {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(data domain.FeedEvent) {
			defer wg.Done()
			defer func() { <-semaphore }()

			err := s.PushMSG(ctx, &data)
			if err != nil {
				mu.Lock()
				errs = append(errs, ErrWithData{
					FeedEvent: &data,
					Err:       err,
				})
				mu.Unlock()
			}
		}(pushData)
	}
	wg.Wait()

	return errs
}

func (s *pushService) LoadPushTargets(ctx context.Context, studentIDs []string) (*PushTargets, error) {
	targets := &PushTargets{users: make(map[string]pushTarget)}
	ids := make([]string, 0, len(studentIDs))
	for _, id := range studentIDs {
		if !tool.IsValidStudentID(id) {
			continue
		}
		if _, exists := targets.users[id]; !exists {
			targets.users[id] = pushTarget{}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return targets, nil
	}
	configs, err := s.userFeedConfigDAO.GetPushConfigs(ctx, ids)
	if err != nil {
		return nil, errorx.Errorf("service: load push configs failed: %w", err)
	}
	tokens, err := s.feedTokenDAO.GetTokensBatch(ctx, ids)
	if err != nil {
		return nil, errorx.Errorf("service: load push tokens failed: %w", err)
	}
	for _, id := range ids {
		targets.users[id] = pushTarget{config: configs[id], tokens: tokens[id]}
	}
	return targets, nil
}

func (s *pushService) GetPushCID(ctx context.Context) (string, error) {
	cid, err := s.pushClient.GetCID(ctx)
	if err != nil {
		return "", errorx.Errorf("service: get jpush cid failed: %w", err)
	}
	return cid, nil
}

// 推送单条消息
func (s *pushService) PushMSG(ctx context.Context, pushData *domain.FeedEvent) error {
	return s.PushMSGWithCID(ctx, pushData, "")
}

func (s *pushService) PushMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, cid string) error {
	prepared, err := s.PreparePush(ctx, pushData)
	if err != nil || prepared == nil {
		return err
	}
	return s.PushPreparedMSGWithCID(ctx, pushData, prepared, cid)
}

func (s *pushService) PreparePush(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, error) {
	prepared, _, err := s.PreparePushForDelivery(ctx, pushData)
	return prepared, err
}

func (s *pushService) PreparePushForDelivery(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, string, error) {
	if !tool.IsValidStudentID(pushData.StudentId) {
		return nil, "suppressed_no_target_or_disabled", nil
	}
	config, err := s.userFeedConfigDAO.GetPushConfig(ctx, pushData.StudentId)
	if err != nil {
		return nil, "", errorx.Errorf("service: get push config failed, sid: %s, err: %w", pushData.StudentId, err)
	}
	if !pushAllowed(pushData.Type, config) {
		return nil, suppressionReason(pushData.Type), nil
	}
	tokens, err := s.feedTokenDAO.GetTokens(ctx, pushData.StudentId)
	if err != nil {
		return nil, "", errorx.Errorf("service: get tokens failed for push, sid: %s, err: %w", pushData.StudentId, err)
	}
	prepared, reason := preparePushTarget(pushData.Type, pushTarget{config: config, tokens: tokens})
	return prepared, reason, nil
}

func suppressionReason(label string) string {
	if strings.EqualFold(label, "library") {
		return "suppressed_by_allow_list"
	}
	return "suppressed_no_target_or_disabled"
}

func pushAllowed(label string, config dao.PushConfigSnapshot) bool {
	pos, exists := configMap[label]
	if !exists || config.Deleted {
		return false
	}
	if !config.Exists {
		if strings.EqualFold(label, "library") {
			return false
		}
		config.Config = model.DefaultPushConfig
	}
	return config.Config&(1<<pos) != 0
}

func (s *pushService) PushPreparedMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, prepared *PreparedPush, cid string) error {
	if prepared == nil || len(prepared.tokens) == 0 {
		return nil
	}
	var expires *time.Time
	if strings.EqualFold(pushData.Type, "library") && strings.EqualFold(strings.TrimSpace(pushData.ExtendFields["notification_type"]), "TEAM_INVITATION") {
		if err := domain.ValidateLibraryTeamInvitation(*pushData); err != nil {
			return jpush.ErrPushExpired
		}
		value, err := strconv.ParseInt(pushData.ExtendFields["expires_at"], 10, 64)
		if err != nil {
			return jpush.ErrPushExpired
		}
		at := time.Unix(value, 0)
		expires = &at
	}
	err := s.pushClient.Push(ctx, prepared.tokens, jpush.PushData{
		ExpiresAt:   expires,
		ContentType: pushData.Type,
		Extras:      pushData.ExtendFields,
		MsgContent:  pushData.Content,
		Title:       pushData.Title,
		Cid:         cid,
	})
	if err != nil {
		return errorx.Errorf("service: jpush client call failed, sid: %s, tokens_count: %d, err: %w", pushData.StudentId, len(prepared.tokens), err)
	}
	return nil
}
