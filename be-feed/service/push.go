package service

import (
	"context"
	"sync"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type pushService struct {
	pushClient        jpush.PushClient //用于推送的客户端
	userFeedConfigDAO dao.FeedUserConfigDAO
	feedFailEventDAO  dao.FeedFailEventDAO
	feedTokenDAO      dao.FeedTokenDAO
	l                 logger.Logger
}

type PushService interface {
	GetPushCID(context.Context) (string, error)
	PreparePush(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, error)
	PushPreparedMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, prepared *PreparedPush, cid string) error
	PushMSG(ctx context.Context, pushData *domain.FeedEvent) error
	PushMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, cid string) error
	PushMSGS(ctx context.Context, pushDatas []domain.FeedEvent) []ErrWithData
	PushToAll(ctx context.Context, pushData *domain.FeedEvent) error
	InsertFailFeedEvents(ctx context.Context, failEvents []domain.FeedEvent) error
}

// PreparedPush 保存已通过 Token 和推送开关检查的接收目标。
type PreparedPush struct {
	tokens []string
}

type ErrWithData struct {
	FeedEvent *domain.FeedEvent `json:"feed_event"`
	Err       error             `json:"err"`
}

func NewPushService(
	pushClient jpush.PushClient,
	userFeedConfigDAO dao.FeedUserConfigDAO,
	feedTokenDAO dao.FeedTokenDAO,
	feedFailEventDAO dao.FeedFailEventDAO,
	l logger.Logger,
) PushService {
	return &pushService{
		pushClient:        pushClient,
		userFeedConfigDAO: userFeedConfigDAO,
		feedTokenDAO:      feedTokenDAO,
		feedFailEventDAO:  feedFailEventDAO,
		l:                 l,
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

// 此处返回errors但是不做错误处理,如果还是失败选择放任着条消息丢失
func (s *pushService) InsertFailFeedEvents(ctx context.Context, failEvents []domain.FeedEvent) error {
	err := s.feedFailEventDAO.InsertFeedFailEventList(ctx, convFeedFailEventFromDomainToModel(failEvents))
	if err != nil {
		return errorx.Errorf("service: insert fail feed events failed, count: %d, err: %w", len(failEvents), err)
	}
	return nil
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
	tokens, err := s.feedTokenDAO.GetTokens(ctx, pushData.StudentId)
	if err != nil {
		return nil, errorx.Errorf("service: get tokens failed for push, sid: %s, err: %w", pushData.StudentId, err)
	}
	if len(tokens) == 0 {
		return nil, nil
	}

	// 权限检测
	allowed, err := s.checkIfAllow(ctx, pushData.Type, pushData.StudentId)
	if err != nil {
		return nil, errorx.Errorf("service: check push permission failed, sid: %s, type: %s, err: %w", pushData.StudentId, pushData.Type, err)
	}
	if !allowed {
		return nil, nil
	}
	return &PreparedPush{tokens: tokens}, nil
}

func (s *pushService) PushPreparedMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, prepared *PreparedPush, cid string) error {
	if prepared == nil || len(prepared.tokens) == 0 {
		return nil
	}
	err := s.pushClient.Push(ctx, prepared.tokens, jpush.PushData{
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

// 推送消息给所有人[弃用]:推送成本太高,而且事务难以实现,一致性难
func (s *pushService) PushToAll(ctx context.Context, pushData *domain.FeedEvent) error {
	const batchSize = 50
	var lastId int64 = 0

	for {
		studentIdsAndTokens, newLastId, err := s.feedTokenDAO.GetStudentIdAndTokensByCursor(ctx, lastId, batchSize)
		if err != nil {
			s.l.Error("service: push to all get cursor data error", logger.Int64("lastId", lastId), logger.Error(err))
			break // 游标查询失败属于严重错误，退出循环
		}

		if len(studentIdsAndTokens) == 0 {
			break
		}

		var filteredTokens []string

		for studentId, tokens := range studentIdsAndTokens {
			allowed, err := s.checkIfAllow(ctx, pushData.Type, studentId)
			if err != nil {
				s.l.Error("service: push to all check allow error ignored", logger.String("sid", studentId), logger.Error(err))
				continue
			}

			if allowed {
				filteredTokens = append(filteredTokens, tokens...)
			}
		}

		if len(filteredTokens) > 0 {
			err = s.pushClient.Push(ctx, filteredTokens, jpush.PushData{
				ContentType: pushData.Type,
				Extras:      pushData.ExtendFields,
				MsgContent:  pushData.Content,
				Title:       pushData.Title,
			})

			if err != nil {
				s.l.Error("service: push to all batch jpush error", logger.Int("tokens_count", len(filteredTokens)), logger.Error(err))
			}
		}

		lastId = newLastId
	}

	return nil
}

func (s *pushService) checkIfAllow(ctx context.Context, label string, studentId string) (bool, error) {
	list, err := s.userFeedConfigDAO.FindOrCreateUserFeedConfig(ctx, studentId)
	if err != nil {
		return false, errorx.Errorf("service: find/create config failed in allow check, sid: %s, err: %w", studentId, err)
	}

	pos, exists := configMap[label]
	if !exists {
		return false, nil
	}

	return s.userFeedConfigDAO.GetConfigBit(list.PushConfig, pos), nil
}
