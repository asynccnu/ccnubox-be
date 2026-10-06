package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/tool"
	v1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	smartSeatMaxPlans       = 3
	smartSeatMaxRooms       = 10
	smartSeatTargetSliceMin = 30
	smartSeatMaxSlices      = 8
	smartSeatMaxCandidates  = 12
	smartSeatMaxVerify      = 6
	smartSeatMaxVariants    = 6
	smartSeatConcurrency    = 8
	smartSeatMaxSegments    = 12
	smartSeatTimelineTTL    = 60 * time.Second
	smartSeatGroupPrefix    = "ccnubox:library:smart_seat"
	smartSeatTimelinePrefix = "ccnubox:library:smart_seat:timeline"
)

// seatInterval 空闲区间，闭开区间 [start, end)，单位为当天分钟数。
type seatInterval struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// seatTimeline 座位在指定日期的空闲时间线。
type seatTimeline struct {
	RoomID    string
	SeatID    string
	SeatLabel string
	SeatName  string
	Intervals []seatInterval
}

// smartSeatPick 接力链中的一段：某座位在 [Start, End) 连续空闲。
type smartSeatPick struct {
	RoomID    string
	SeatID    string
	SeatLabel string
	SeatName  string
	Start     int
	End       int
}

// smartSeatSegment 已规范化的预约分段。
type smartSeatSegment struct {
	RoomID      string
	SeatID      string
	SeatLabel   string
	StartMinute int
	EndMinute   int
}

// smartSeatGroupSegment 存储在 Redis 中的分组分段。
type smartSeatGroupSegment struct {
	SeatID    string `json:"seat_id"`
	SeatLabel string `json:"seat_label"`
	RoomID    string `json:"room_id"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

// smartSeatGroup 智能选座整组预约信息。
type smartSeatGroup struct {
	Date      string                  `json:"date"`
	Segments  []smartSeatGroupSegment `json:"segments"`
	CreatedAt int64                   `json:"created_at"`
}

// smartSeatPlanner 智能选座规划器。
type smartSeatPlanner struct {
	crawler *crawler.Crawler
	rdb     *redis.Client
	l       logger.Logger
}

// normalizeSmartSeatRoomIDs 校验并规范化房间列表：去重、去空白、限制数量上限。
func normalizeSmartSeatRoomIDs(roomIDs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(roomIDs))
	result := make([]string, 0, len(roomIDs))
	for _, roomID := range roomIDs {
		roomID = strings.TrimSpace(roomID)
		if roomID == "" {
			continue
		}
		if _, ok := seen[roomID]; ok {
			continue
		}
		seen[roomID] = struct{}{}
		result = append(result, roomID)
	}
	if len(result) == 0 {
		return nil, errorx.New("room_ids must not be empty")
	}
	if len(result) > smartSeatMaxRooms {
		return nil, errorx.Errorf("too many rooms: %d, max %d", len(result), smartSeatMaxRooms)
	}
	return result, nil
}

// GetSmartSeatPlans 生成智能选座接力方案。
func (s *seatService) GetSmartSeatPlans(ctx context.Context, req *v1.GetSmartSeatPlansRequest) (*v1.GetSmartSeatPlansResponse, error) {
	if req == nil || len(req.RoomIds) == 0 || req.Date == "" || req.Start == "" || req.End == "" {
		return nil, ErrGetSeat(errorx.New("room_ids, date, start and end are required"))
	}
	roomIDs, err := normalizeSmartSeatRoomIDs(req.RoomIds)
	if err != nil {
		return nil, ErrGetSeat(err)
	}
	if err := validateSeatDate(req.Date); err != nil {
		return nil, ErrGetSeat(err)
	}
	startMinute, err := crawler.ParseMinute(req.Start)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("invalid start time: %w", err))
	}
	endMinute, err := crawler.ParseMinute(req.End)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("invalid end time: %w", err))
	}
	if endMinute <= startMinute {
		return nil, ErrGetSeat(errorx.New("end time must be after start time"))
	}

	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}

	planner := &smartSeatPlanner{crawler: s.crawler, rdb: s.rdb, l: s.l}
	chains, err := planner.buildPlans(ctx, token, req.Date, roomIDs, startMinute, endMinute)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("build smart seat plans failed, stuId: %s, err: %w", req.StuId, err))
	}
	if len(chains) == 0 {
		return nil, ErrNoAvailableSeat(errorx.Errorf("no smart seat plan, stuId: %s, date: %s, period: %s-%s", req.StuId, req.Date, req.Start, req.End))
	}

	plans := make([]*v1.SmartSeatPlan, 0, len(chains))
	for _, chain := range chains {
		segments := make([]*v1.SmartSeatSegment, 0, len(chain))
		for _, pick := range chain {
			segments = append(segments, &v1.SmartSeatSegment{
				SeatId:    pick.SeatID,
				SeatLabel: pick.SeatLabel,
				SeatName:  pick.SeatName,
				RoomId:    pick.RoomID,
				Start:     crawler.FormatMinute(pick.Start),
				End:       crawler.FormatMinute(pick.End),
			})
		}
		plans = append(plans, &v1.SmartSeatPlan{
			Segments:     segments,
			SegmentCount: int32(len(segments)),
		})
	}
	return &v1.GetSmartSeatPlansResponse{Plans: plans}, nil
}

// buildPlans 组装 ≤3 个多样化的接力方案：优先全程单座，不足时用子窗口拼接补齐。
func (p *smartSeatPlanner) buildPlans(ctx context.Context, token, date string, roomIDs []string, start, end int) ([][]smartSeatPick, error) {
	roomIndex := make(map[string]int, len(roomIDs))
	for i, roomID := range roomIDs {
		roomIndex[roomID] = i
	}

	chains := p.singleSeatPlans(ctx, token, date, roomIDs, start, end)
	if len(chains) < smartSeatMaxPlans {
		composed, err := p.composedPlans(ctx, token, date, roomIDs, start, end, chains)
		if err != nil {
			p.logWarn("smart seat: compose plans failed, err: %v", err)
		} else {
			chains = append(chains, composed...)
		}
	}

	chains = dedupeSmartSeatChains(chains)
	sortSmartSeatChains(chains, roomIndex)
	if len(chains) > smartSeatMaxPlans {
		chains = chains[:smartSeatMaxPlans]
	}
	return chains, nil
}

// singleSeatPlans 查询整段空闲的单座方案，并用精确时间线二次校验。
func (p *smartSeatPlanner) singleSeatPlans(ctx context.Context, token, date string, roomIDs []string, start, end int) [][]smartSeatPick {
	type candidate struct {
		roomID string
		seat   *crawler.Seat
	}
	var candidates []candidate
	for _, roomID := range roomIDs {
		seats, err := p.crawler.GetSeatInfosWithWindow(ctx, token, roomID, date, start, end, end-start)
		if err != nil {
			p.logWarn("smart seat: query whole window failed, room: %s, err: %v", roomID, err)
			continue
		}
		sort.SliceStable(seats, func(i, j int) bool {
			if seats[i] == nil || seats[j] == nil {
				return false
			}
			return seats[i].Label < seats[j].Label
		})
		for _, seat := range seats {
			if seat == nil || seat.ID == "" {
				continue
			}
			candidates = append(candidates, candidate{roomID: roomID, seat: seat})
			if len(candidates) >= smartSeatMaxVerify {
				break
			}
		}
		if len(candidates) >= smartSeatMaxVerify {
			break
		}
	}

	var chains [][]smartSeatPick
	for _, item := range candidates {
		intervals, err := p.seatTimeline(ctx, token, item.seat.ID, date)
		if err != nil {
			p.logWarn("smart seat: verify single seat %s failed, err: %v", item.seat.ID, err)
			continue
		}
		if !intervalsCover(intervals, start, end) {
			continue
		}
		chains = append(chains, []smartSeatPick{{
			RoomID:    item.roomID,
			SeatID:    item.seat.ID,
			SeatLabel: item.seat.Label,
			SeatName:  item.seat.Name,
			Start:     start,
			End:       end,
		}})
		if len(chains) >= smartSeatMaxPlans {
			break
		}
	}
	return chains
}

// composedPlans 按子窗口收集候选座位，再用精确时间线贪心拼接接力链。
// base 为已生成的整段单座方案，其中的座位不再参与拼接。
func (p *smartSeatPlanner) composedPlans(ctx context.Context, token, date string, roomIDs []string, start, end int, base [][]smartSeatPick) ([][]smartSeatPick, error) {
	slices := splitSmartSeatWindow(start, end)
	if len(slices) < 2 {
		return nil, nil
	}

	type candidate struct {
		roomID  string
		seat    *crawler.Seat
		covered int
	}
	var mu sync.Mutex
	candidateMap := make(map[string]*candidate)
	sem := make(chan struct{}, smartSeatConcurrency)
	var wg sync.WaitGroup
	for _, roomID := range roomIDs {
		for _, slice := range slices {
			roomID, slice := roomID, slice
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					return
				}
				seats, err := p.crawler.GetSeatInfosWithWindow(ctx, token, roomID, date, slice.Start, slice.End, slice.End-slice.Start)
				if err != nil {
					p.logWarn("smart seat: query slice failed, room: %s, err: %v", roomID, err)
					return
				}
				mu.Lock()
				defer mu.Unlock()
				for _, seat := range seats {
					if seat == nil || seat.ID == "" {
						continue
					}
					key := roomID + "|" + seat.ID
					item, ok := candidateMap[key]
					if !ok {
						item = &candidate{roomID: roomID, seat: seat}
						candidateMap[key] = item
					}
					item.covered++
				}
			}()
		}
	}
	wg.Wait()

	// 已在单座方案中出现过的座位无需重复参与拼接
	used := make(map[string]struct{})
	for _, chain := range base {
		for _, pick := range chain {
			used[pick.RoomID+"|"+pick.SeatID] = struct{}{}
		}
	}

	candidates := make([]*candidate, 0, len(candidateMap))
	for key, item := range candidateMap {
		if _, ok := used[key]; ok {
			continue
		}
		candidates = append(candidates, item)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].covered != candidates[j].covered {
			return candidates[i].covered > candidates[j].covered
		}
		if candidates[i].roomID != candidates[j].roomID {
			return candidates[i].roomID < candidates[j].roomID
		}
		return candidates[i].seat.Label < candidates[j].seat.Label
	})
	if len(candidates) > smartSeatMaxCandidates {
		candidates = candidates[:smartSeatMaxCandidates]
	}

	var mu2 sync.Mutex
	timelines := make([]seatTimeline, 0, len(candidates))
	var wg2 sync.WaitGroup
	for _, item := range candidates {
		item := item
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			intervals, err := p.seatTimeline(ctx, token, item.seat.ID, date)
			if err != nil {
				p.logWarn("smart seat: get timeline failed, seat: %s, err: %v", item.seat.ID, err)
				return
			}
			intervals = clipIntervals(intervals, start, end)
			if len(intervals) == 0 {
				return
			}
			mu2.Lock()
			defer mu2.Unlock()
			timelines = append(timelines, seatTimeline{
				RoomID:    item.roomID,
				SeatID:    item.seat.ID,
				SeatLabel: item.seat.Label,
				SeatName:  item.seat.Name,
				Intervals: intervals,
			})
		}()
	}
	wg2.Wait()

	return buildSmartSeatChains(timelines, start, end), nil
}

// seatTimeline 获取座位时间线（带短 TTL 缓存，缓存异常时静默降级）。
func (p *smartSeatPlanner) seatTimeline(ctx context.Context, token, seatID, date string) ([]seatInterval, error) {
	cacheKey := fmt.Sprintf("%s:%s:%s", smartSeatTimelinePrefix, seatID, date)
	if p.rdb != nil {
		if raw, err := p.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
			var intervals []seatInterval
			if err := json.Unmarshal(raw, &intervals); err == nil {
				return intervals, nil
			}
		}
	}
	freeList, err := p.crawler.GetFreeList(ctx, token, seatID, date)
	if err != nil {
		return nil, err
	}
	intervals := parseSeatIntervals(freeList)
	if p.rdb != nil {
		if data, err := json.Marshal(intervals); err == nil {
			p.rdb.Set(ctx, cacheKey, data, smartSeatTimelineTTL)
		}
	}
	return intervals, nil
}

func (p *smartSeatPlanner) logWarn(format string, args ...any) {
	if p.l != nil {
		p.l.Warnf(format, args...)
	}
}

// splitSmartSeatWindow 将窗口切分为 2~8 个子窗口，每个约 30 分钟。
func splitSmartSeatWindow(start, end int) []seatInterval {
	duration := end - start
	if duration <= 0 {
		return nil
	}
	slices := duration / smartSeatTargetSliceMin
	if duration%smartSeatTargetSliceMin != 0 {
		slices++
	}
	if slices > smartSeatMaxSlices {
		slices = smartSeatMaxSlices
	}
	if slices < 2 {
		return nil
	}
	length := duration / slices
	result := make([]seatInterval, 0, slices)
	for i := 0; i < slices; i++ {
		sliceStart := start + i*length
		sliceEnd := sliceStart + length
		if i == slices-1 {
			sliceEnd = end
		}
		result = append(result, seatInterval{Start: sliceStart, End: sliceEnd})
	}
	return result
}

// parseSeatIntervals 解析 getTimeLine 返回的空闲时间段（HH:MM-HH:MM）。
func parseSeatIntervals(freeList []*crawler.FreeTime) []seatInterval {
	intervals := make([]seatInterval, 0, len(freeList))
	for _, item := range freeList {
		if item == nil {
			continue
		}
		start, err := crawler.ParseMinute(item.Start)
		if err != nil {
			continue
		}
		end, err := crawler.ParseMinute(item.End)
		if err != nil || end <= start {
			continue
		}
		intervals = append(intervals, seatInterval{Start: start, End: end})
	}
	sort.SliceStable(intervals, func(i, j int) bool { return intervals[i].Start < intervals[j].Start })
	return intervals
}

// clipIntervals 将空闲区间裁剪到指定窗口内。
func clipIntervals(intervals []seatInterval, start, end int) []seatInterval {
	result := make([]seatInterval, 0, len(intervals))
	for _, interval := range intervals {
		clipped := seatInterval{Start: maxInt(interval.Start, start), End: minInt(interval.End, end)}
		if clipped.End > clipped.Start {
			result = append(result, clipped)
		}
	}
	return result
}

func intervalsCover(intervals []seatInterval, start, end int) bool {
	for _, interval := range intervals {
		if interval.Start <= start && interval.End >= end {
			return true
		}
	}
	return false
}

func timelineCoversPoint(item *seatTimeline, minute int) bool {
	for _, interval := range item.Intervals {
		if interval.Start <= minute && interval.End > minute {
			return true
		}
	}
	return false
}

// buildSmartSeatChains 生成若干条覆盖 [start,end] 的接力链（贪心最小段数 + 首座变体）。
func buildSmartSeatChains(timelines []seatTimeline, start, end int) [][]smartSeatPick {
	seen := make(map[string]struct{})
	var chains [][]smartSeatPick
	add := func(chain []smartSeatPick) {
		if len(chain) == 0 {
			return
		}
		key := smartSeatChainKey(chain)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		chains = append(chains, chain)
	}

	if chain, ok := greedySmartSeatChain(timelines, start, end, nil); ok {
		add(chain)
	}
	for i := range timelines {
		if len(chains) >= smartSeatMaxVariants {
			break
		}
		item := &timelines[i]
		if !timelineCoversPoint(item, start) {
			continue
		}
		if chain, ok := greedySmartSeatChain(timelines, start, end, item); ok {
			add(chain)
		}
	}
	return chains
}

// greedySmartSeatChain 从 start 出发每次选择能延伸最远的空闲区间，直到覆盖到 end。
func greedySmartSeatChain(timelines []seatTimeline, start, end int, first *seatTimeline) ([]smartSeatPick, bool) {
	var picks []smartSeatPick
	current := start
	if first != nil {
		interval, ok := bestInterval(*first, current, end)
		if !ok {
			return nil, false
		}
		picks = append(picks, pickFromTimeline(first, interval, current, end))
		current = interval.End
	}
	for current < end {
		if len(picks) >= smartSeatMaxSegments {
			return nil, false
		}
		var best *seatTimeline
		var bestIv seatInterval
		found := false
		for i := range timelines {
			item := &timelines[i]
			for _, interval := range item.Intervals {
				if interval.Start <= current && interval.End > current {
					if !found || interval.End > bestIv.End {
						best = item
						bestIv = interval
						found = true
					}
				}
			}
		}
		if !found {
			return nil, false
		}
		picks = append(picks, pickFromTimeline(best, bestIv, current, end))
		current = bestIv.End
	}
	if len(picks) == 0 {
		return nil, false
	}
	return picks, true
}

func bestInterval(item seatTimeline, current, end int) (seatInterval, bool) {
	var best seatInterval
	found := false
	for _, interval := range item.Intervals {
		if interval.Start <= current && interval.End > current {
			if !found || interval.End > best.End {
				best = interval
				found = true
			}
		}
	}
	if !found {
		return seatInterval{}, false
	}
	return best, true
}

func pickFromTimeline(item *seatTimeline, interval seatInterval, current, end int) smartSeatPick {
	pickEnd := minInt(interval.End, end)
	return smartSeatPick{
		RoomID:    item.RoomID,
		SeatID:    item.SeatID,
		SeatLabel: item.SeatLabel,
		SeatName:  item.SeatName,
		Start:     current,
		End:       pickEnd,
	}
}

func smartSeatChainKey(chain []smartSeatPick) string {
	key := ""
	for _, pick := range chain {
		key += fmt.Sprintf("%s|%s|%d|%d;", pick.RoomID, pick.SeatID, pick.Start, pick.End)
	}
	return key
}

func dedupeSmartSeatChains(chains [][]smartSeatPick) [][]smartSeatPick {
	seen := make(map[string]struct{}, len(chains))
	result := make([][]smartSeatPick, 0, len(chains))
	for _, chain := range chains {
		if len(chain) == 0 {
			continue
		}
		key := smartSeatChainKey(chain)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, chain)
	}
	return result
}

func sortSmartSeatChains(chains [][]smartSeatPick, roomIndex map[string]int) {
	sort.SliceStable(chains, func(i, j int) bool {
		if len(chains[i]) != len(chains[j]) {
			return len(chains[i]) < len(chains[j])
		}
		ri, rj := roomRank(chains[i], roomIndex), roomRank(chains[j], roomIndex)
		if ri != rj {
			return ri < rj
		}
		return smartSeatChainKey(chains[i]) < smartSeatChainKey(chains[j])
	})
}

func roomRank(chain []smartSeatPick, roomIndex map[string]int) int {
	if len(chain) == 0 {
		return 0
	}
	if rank, ok := roomIndex[chain[0].RoomID]; ok {
		return rank
	}
	return len(roomIndex)
}

// normalizeSmartSeatSegments 校验并规范化请求中的分段，要求构成连续接力的时间链。
func normalizeSmartSeatSegments(segments []*v1.SmartSeatSegment) ([]smartSeatSegment, error) {
	if len(segments) == 0 {
		return nil, errorx.New("segments must not be empty")
	}
	result := make([]smartSeatSegment, 0, len(segments))
	for _, segment := range segments {
		if segment == nil || segment.SeatId == "" {
			return nil, errorx.New("seat_id is required for every segment")
		}
		start, err := crawler.ParseMinute(segment.Start)
		if err != nil {
			return nil, errorx.Errorf("invalid segment start %q: %w", segment.Start, err)
		}
		end, err := crawler.ParseMinute(segment.End)
		if err != nil {
			return nil, errorx.Errorf("invalid segment end %q: %w", segment.End, err)
		}
		if end <= start {
			return nil, errorx.Errorf("segment end must be after start: %s-%s", segment.Start, segment.End)
		}
		result = append(result, smartSeatSegment{
			RoomID:      segment.RoomId,
			SeatID:      segment.SeatId,
			SeatLabel:   segment.SeatLabel,
			StartMinute: start,
			EndMinute:   end,
		})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].StartMinute < result[j].StartMinute })
	for i := 1; i < len(result); i++ {
		if result[i].StartMinute != result[i-1].EndMinute {
			return nil, errorx.New("segments must form a continuous chain")
		}
	}
	if len(result) > smartSeatMaxSegments {
		return nil, errorx.Errorf("too many segments: %d", len(result))
	}
	return result, nil
}

// ReserveSmartSeatPlan 预约整组接力方案：全有或全无，失败回滚已预约分段。
func (s *seatService) ReserveSmartSeatPlan(ctx context.Context, req *v1.ReserveSmartSeatPlanRequest) (*v1.ReserveSmartSeatPlanResponse, error) {
	if req == nil || req.Date == "" || len(req.Segments) == 0 {
		return nil, ErrGetSeat(errorx.New("date and segments are required"))
	}
	if err := validateSeatDate(req.Date); err != nil {
		return nil, ErrGetSeat(err)
	}
	segments, err := normalizeSmartSeatSegments(req.Segments)
	if err != nil {
		return nil, ErrGetSeat(err)
	}
	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}

	planner := &smartSeatPlanner{crawler: s.crawler, rdb: s.rdb, l: s.l}
	// 预约前逐段预校验，尽量避免预约到一半失败。
	for _, segment := range segments {
		intervals, err := planner.seatTimeline(ctx, token, segment.SeatID, req.Date)
		if err != nil {
			return nil, ErrGetSeat(errorx.Errorf("verify segment failed, seat: %s, err: %w", segment.SeatID, err))
		}
		if !intervalsCover(intervals, segment.StartMinute, segment.EndMinute) {
			return nil, ErrNoAvailableSeat(errorx.Errorf("seat %s is no longer free for %s-%s", segment.SeatID, crawler.FormatMinute(segment.StartMinute), crawler.FormatMinute(segment.EndMinute)))
		}
	}

	booked := make([]smartSeatSegment, 0, len(segments))
	message := ""
	for i, segment := range segments {
		msg, err := s.crawler.ReserveSeat(ctx, token, segment.SeatID, req.Date, crawler.FormatMinute(segment.StartMinute), crawler.FormatMinute(segment.EndMinute))
		if err != nil {
			// 失败的分段可能已在上游实际成单（如超时或响应解析失败），因此连同当前分段一并回滚。
			residue, rollbackErr := s.rollbackSmartSeat(ctx, token, req.Date, segments[:i+1])
			if len(residue) > 0 {
				if saveErr := s.saveSmartSeatGroup(ctx, req.StuId, req.Date, residue); saveErr != nil {
					s.logWarn("smart seat: save residue group failed, stuId: %s, date: %s, err: %v", req.StuId, req.Date, saveErr)
				}
			}
			if rollbackErr != nil {
				return nil, ErrGetSeat(errorx.Errorf("reserve segment %d failed: %w; rollback incomplete: %v", i+1, err, rollbackErr))
			}
			return nil, ErrGetSeat(errorx.Errorf("reserve segment %d failed: %w", i+1, err))
		}
		if msg != "" {
			message = msg
		}
		booked = append(booked, segment)
	}

	if err := s.saveSmartSeatGroup(ctx, req.StuId, req.Date, booked); err != nil {
		s.logWarn("smart seat: save group failed, stuId: %s, date: %s, err: %v", req.StuId, req.Date, err)
	}
	if message == "" {
		message = fmt.Sprintf("预约成功，共 %d 个座位", len(booked))
	}
	return &v1.ReserveSmartSeatPlanResponse{Message: message}, nil
}

// CancelSmartSeatPlan 取消整组智能选座预约。
func (s *seatService) CancelSmartSeatPlan(ctx context.Context, req *v1.CancelSmartSeatPlanRequest) (*v1.CancelSmartSeatPlanResponse, error) {
	if req == nil || req.Date == "" {
		return nil, ErrGetSeat(errorx.New("date is required"))
	}
	if err := validateSeatDate(req.Date); err != nil {
		return nil, ErrGetSeat(err)
	}
	group, ok, err := s.loadSmartSeatGroup(ctx, req.StuId, req.Date)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("load smart seat group failed, stuId: %s, err: %w", req.StuId, err))
	}
	if !ok {
		return nil, ErrGetSeat(errorx.Errorf("smart seat group not found, stuId: %s, date: %s", req.StuId, req.Date))
	}

	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	records, err := s.seatRecordsForDate(ctx, token, req.Date)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("get seat records failed, stuId: %s, err: %w", req.StuId, err))
	}

	segments := make([]smartSeatSegment, 0, len(group.Segments))
	for _, item := range group.Segments {
		start, err := crawler.ParseMinute(item.Start)
		if err != nil {
			return nil, ErrGetSeat(errorx.Errorf("invalid stored segment start %q: %w", item.Start, err))
		}
		end, err := crawler.ParseMinute(item.End)
		if err != nil {
			return nil, ErrGetSeat(errorx.Errorf("invalid stored segment end %q: %w", item.End, err))
		}
		segments = append(segments, smartSeatSegment{
			RoomID:      item.RoomID,
			SeatID:      item.SeatID,
			SeatLabel:   item.SeatLabel,
			StartMinute: start,
			EndMinute:   end,
		})
	}

	matched, missing := matchSmartSeatRecords(records, req.Date, segments)
	// 记录拉取可能不完整（如 lastMake 只返回部分记录、非当日记录不在历史中），
	// 存在未匹配分段时直接中止并保留分组，避免静默丢失取消能力。
	if len(missing) > 0 {
		return nil, ErrGetSeat(errorx.Errorf("cancel smart seat aborted: %d segment(s) not found (seats: %s), stuId: %s", len(missing), smartSeatSegmentLabels(missing), req.StuId))
	}
	canceled := 0
	var firstErr error
	for _, record := range matched {
		if _, err := s.crawler.CancelReserve(ctx, token, record.ID); err != nil {
			if firstErr == nil {
				firstErr = errorx.Errorf("cancel record %s failed: %w", record.ID, err)
			}
			continue
		}
		canceled++
	}
	if firstErr != nil {
		return nil, ErrGetSeat(errorx.Errorf("cancel smart seat partially failed (canceled %d), stuId: %s, err: %w", canceled, req.StuId, firstErr))
	}
	if err := s.deleteSmartSeatGroup(ctx, req.StuId, req.Date); err != nil {
		s.logWarn("smart seat: delete group failed, stuId: %s, date: %s, err: %v", req.StuId, req.Date, err)
	}
	return &v1.CancelSmartSeatPlanResponse{Message: fmt.Sprintf("已取消 %d 个座位", canceled)}, nil
}

// rollbackSmartSeat 回滚已预约的分段；返回取消失败的残留分段（由预约记录反构），供后续整组取消重试。
func (s *seatService) rollbackSmartSeat(ctx context.Context, token, date string, segments []smartSeatSegment) ([]smartSeatSegment, error) {
	if len(segments) == 0 {
		return nil, nil
	}
	records, err := s.seatRecordsForDate(ctx, token, date)
	if err != nil {
		return nil, err
	}
	matched, _ := matchSmartSeatRecords(records, date, segments)
	var residue []smartSeatSegment
	var firstErr error
	for _, record := range matched {
		if _, err := s.crawler.CancelReserve(ctx, token, record.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			residue = append(residue, segmentFromRecord(record))
		}
	}
	return residue, firstErr
}

// segmentFromRecord 由预约记录反构分段，避免把未真正成单的分段当作残留保存。
func segmentFromRecord(record *crawler.Record) smartSeatSegment {
	loc := tool.GetLocation()
	begin := record.MakeBegin.In(loc)
	end := record.MakeEnd.In(loc)
	return smartSeatSegment{
		RoomID:      record.RoomID,
		SeatID:      record.SeatID,
		SeatLabel:   record.SeatLabel,
		StartMinute: tool.ParseTimeToMinute(begin),
		EndMinute:   tool.ParseTimeToMinute(end),
	}
}

// seatRecordsForDate 查询指定日期的预约记录。
func (s *seatService) seatRecordsForDate(ctx context.Context, token, date string) ([]*crawler.Record, error) {
	loc := tool.GetLocation()
	today := time.Now().In(loc).Format("2006-01-02")
	if date == today {
		return s.crawler.GetTodayRecord(ctx, token)
	}
	records, err := s.crawler.GetHistory(ctx, token)
	if err != nil {
		return nil, err
	}
	result := make([]*crawler.Record, 0, len(records))
	for _, record := range records {
		if record != nil && record.MakeDate.In(loc).Format("2006-01-02") == date {
			result = append(result, record)
		}
	}
	return result, nil
}

// matchSmartSeatRecords 按座位与时间匹配预约记录；返回未找到的记录分段。
func matchSmartSeatRecords(records []*crawler.Record, date string, segments []smartSeatSegment) ([]*crawler.Record, []smartSeatSegment) {
	var matched []*crawler.Record
	var missing []smartSeatSegment
	used := make(map[int]struct{})
	for _, segment := range segments {
		found := false
		for i, record := range records {
			if record == nil {
				continue
			}
			if _, ok := used[i]; ok {
				continue
			}
			if record.SeatID != segment.SeatID {
				continue
			}
			if record.MakeDate.Format("2006-01-02") != date {
				continue
			}
			if record.MakeBegin.Format("15:04") != crawler.FormatMinute(segment.StartMinute) {
				continue
			}
			if record.MakeEnd.Format("15:04") != crawler.FormatMinute(segment.EndMinute) {
				continue
			}
			used[i] = struct{}{}
			matched = append(matched, record)
			found = true
			break
		}
		if !found {
			missing = append(missing, segment)
		}
	}
	return matched, missing
}

// smartSeatSegmentLabels 拼接分段座位标签，便于错误信息定位。
func smartSeatSegmentLabels(segments []smartSeatSegment) string {
	labels := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment.SeatLabel != "" {
			labels = append(labels, segment.SeatLabel)
			continue
		}
		labels = append(labels, segment.SeatID)
	}
	return strings.Join(labels, ", ")
}

func (s *seatService) saveSmartSeatGroup(ctx context.Context, stuID, date string, segments []smartSeatSegment) error {
	if s.rdb == nil {
		return errorx.New("redis client is not configured")
	}
	group := smartSeatGroup{
		Date:      date,
		CreatedAt: time.Now().Unix(),
		Segments:  make([]smartSeatGroupSegment, 0, len(segments)),
	}
	for _, segment := range segments {
		group.Segments = append(group.Segments, smartSeatGroupSegment{
			SeatID:    segment.SeatID,
			SeatLabel: segment.SeatLabel,
			RoomID:    segment.RoomID,
			Start:     crawler.FormatMinute(segment.StartMinute),
			End:       crawler.FormatMinute(segment.EndMinute),
		})
	}
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, smartSeatGroupKey(stuID, date), data, smartSeatGroupTTL(date, time.Now())).Err()
}

func (s *seatService) loadSmartSeatGroup(ctx context.Context, stuID, date string) (smartSeatGroup, bool, error) {
	if s.rdb == nil {
		return smartSeatGroup{}, false, nil
	}
	raw, err := s.rdb.Get(ctx, smartSeatGroupKey(stuID, date)).Bytes()
	if err == redis.Nil {
		return smartSeatGroup{}, false, nil
	}
	if err != nil {
		return smartSeatGroup{}, false, err
	}
	var group smartSeatGroup
	if err := json.Unmarshal(raw, &group); err != nil {
		return smartSeatGroup{}, false, err
	}
	return group, true, nil
}

func (s *seatService) deleteSmartSeatGroup(ctx context.Context, stuID, date string) error {
	if s.rdb == nil {
		return nil
	}
	return s.rdb.Del(ctx, smartSeatGroupKey(stuID, date)).Err()
}

func smartSeatGroupKey(stuID, date string) string {
	return fmt.Sprintf("%s:%s:%s", smartSeatGroupPrefix, stuID, date)
}

// smartSeatGroupTTL 分组信息保留到所选日期次日 + 1 小时。
func smartSeatGroupTTL(date string, now time.Time) time.Duration {
	loc := tool.GetLocation()
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return 48 * time.Hour
	}
	expireAt := day.AddDate(0, 0, 1).Add(time.Hour)
	ttl := expireAt.Sub(now.In(loc))
	if ttl < time.Hour {
		ttl = time.Hour
	}
	return ttl
}

func (s *seatService) logWarn(format string, args ...any) {
	if s.l != nil {
		s.l.Warnf(format, args...)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
