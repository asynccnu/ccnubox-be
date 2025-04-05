package service

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"sync"
)

type pushService struct {
	pushClient        jpush.PushClient //用于推送的客户端
	userFeedConfigDAO dao.UserFeedConfigDAO
	feedFailEventDAO  dao.FeedFailEventDAO
	feedTokenDAO      dao.UserFeedTokenDAO
	l                 logger.Logger
}

type PushService interface {
	PushMSG(ctx context.Context, pushData *domain.FeedEvent) error
	PushMSGS(ctx context.Context, pushDatas []domain.FeedEvent) []ErrWithData
	PushToAll(ctx context.Context, pushData *domain.FeedEvent) error
	InsertFailFeedEvents(ctx context.Context, failEvents []domain.FeedEvent) error
}

type ErrWithData struct {
	FeedEvent *domain.FeedEvent `json:"feed_event"`
	Err       error             `json:"err"`
}

func NewPushService(pushClient jpush.PushClient,

	userFeedConfigDAO dao.UserFeedConfigDAO,
	feedTokenDAO dao.UserFeedTokenDAO,
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
	for _, pushData := range pushDatas {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(data *domain.FeedEvent) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放槽位
			err := s.PushMSG(ctx, data)
			if err != nil {
				errs = append(errs, ErrWithData{
					FeedEvent: data,
					Err:       err,
				})
			}
		}(&pushData)
	}

	return errs

}

// 此处返回errors但是不做错误处理,如果还是失败选择放任着条消息丢失
func (s *pushService) InsertFailFeedEvents(ctx context.Context, failEvents []domain.FeedEvent) error {
	// 插入 FeedEvent 并获取插入后的 ID
	return s.feedFailEventDAO.InsertFeedFailEventList(ctx, convFeedFailEventFromDomainToModel(failEvents))
}

// 推送单条消息
func (s *pushService) PushMSG(ctx context.Context, pushData *domain.FeedEvent) error {
	tokens, err := s.feedTokenDAO.GetTokens(ctx, pushData.StudentId)
	if err != nil {
		return err
	}

	err = s.pushClient.Push(tokens, jpush.PushData{
		ContentType: pushData.Type,
		Extras:      pushData.ExtendFields,
		MsgContent:  pushData.Content,
		Title:       pushData.Title,
	})

	if err != nil {
		return err
	}

	return nil
}

// 推送消息给所有人[弃用]:推送成本太高,而且事务难以实现,一致性难
func (s *pushService) PushToAll(ctx context.Context, pushData *domain.FeedEvent) error {
	const batchSize = 50 // 每批次处理的用户数(为什么一次只推送50条呢?主要是怕推送限流有点严重)
	var lastId int64 = 0 // 游标初始值

	for {

		// 获取一批 studentIds 和 tokens
		studentIdsAndTokens, newLastId, err := s.feedTokenDAO.GetStudentIdAndTokensByCursor(ctx, lastId, batchSize)
		if err != nil {
			s.l.Error("获取用户studentId和tokens错误", append(logger.FormatLog("dao", err))...)
		}

		// 如果没有更多数据，结束循环
		if len(studentIdsAndTokens) == 0 {
			break
		}

		var filteredTokens []string

		// 遍历每个学生的 tokens
		for studentId, tokens := range studentIdsAndTokens {
			// 权限检测
			allowed, err := s.checkIfAllow(ctx, pushData.Type, studentId)
			if err != nil {
				s.l.Error("检查权限出错", append(logger.FormatLog("dao", err))...)
				// 日志记录错误，但不终止流程
				continue
			}

			if !allowed {
				// 如果不允许，跳过当前 studentId
				continue
			}

			// 收集 tokens
			filteredTokens = append(filteredTokens, tokens...)
		}

		// 如果没有需要推送的 tokens，跳过本批次
		if len(filteredTokens) == 0 {
			lastId = newLastId
			continue
		}

		// 批量推送
		err = s.pushClient.Push(filteredTokens, jpush.PushData{
			ContentType: pushData.Type,
			Extras:      pushData.ExtendFields,
			MsgContent:  pushData.Content,
			Title:       pushData.Title,
		})

		if err != nil {
			s.l.Error("批量推送出错", append(logger.FormatLog("push", err))...)
		}

		// 更新游标为最新值
		lastId = newLastId
	}

	return nil
}

func (s *pushService) checkIfAllow(ctx context.Context, label string, studentId string) (bool, error) {
	// 提前获取用户的配置
	list, err := s.userFeedConfigDAO.FindOrCreateUserFeedConfig(ctx, studentId)
	if err != nil {
		return false, err
	}

	// 根据 label 获取对应的位位置
	pos, exists := configMap[label]
	if !exists {
		return false, nil
	}
	ok := s.userFeedConfigDAO.GetConfigBit(list.PushConfig, pos)

	// 根据位位置检查对应的配置是否允许
	return ok, nil
}
