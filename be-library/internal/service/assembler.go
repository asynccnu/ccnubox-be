package service

import (
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
)

type Assembler struct{}

func NewAssembler() *Assembler {
	return &Assembler{}
}

func (a *Assembler) ConvertTimeSlots(src []*biz.TimeSlot) []*pb.TimeSlot {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.TimeSlot, 0, len(src))
	for _, ts := range src {
		result = append(result, &pb.TimeSlot{
			Start:  ts.Start,
			End:    ts.End,
			State:  ts.State,
			Owner:  ts.Owner,
			Occupy: ts.Occupy,
		})
	}
	return result
}

func (a *Assembler) ConvertDiscussionTS(src []*biz.DiscussionTS) []*pb.DiscussionTS {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.DiscussionTS, 0, len(src))
	for _, ts := range src {
		result = append(result, &pb.DiscussionTS{
			Start:  ts.Start,
			End:    ts.End,
			State:  ts.State,
			Title:  ts.Title,
			Owner:  ts.Owner,
			Occupy: ts.Occupy,
		})
	}
	return result
}

func (a *Assembler) ConvertRecords(src []*biz.FutureRecords) []*pb.Record {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.Record, 0, len(src))
	for _, r := range src {
		result = append(result, &pb.Record{
			Id:       r.ID,
			Owner:    r.Owner,
			Start:    r.Start,
			End:      r.End,
			TimeDesc: r.TimeDesc,
			States:   r.States,
			DevName:  r.DevName,
			RoomId:   r.RoomID,
			RoomName: r.RoomName,
			LabName:  r.LabName,
		})
	}
	return result
}

func (a *Assembler) ConvertHistory(src []*biz.HistoryRecords) []*pb.History {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.History, 0, len(src))
	for _, h := range src {
		result = append(result, &pb.History{
			Place:      h.Place,
			Floor:      h.Floor,
			Status:     h.Status,
			Date:       h.Date,
			SubmitTime: h.SubmitTime,
		})
	}
	return result
}

func (a *Assembler) ConvertCreditRecords(src []*biz.CreditRecord) []*pb.CreditRecord {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.CreditRecord, 0, len(src))
	for _, r := range src {
		result = append(result, &pb.CreditRecord{
			Title:    r.Title,
			Subtitle: r.Subtitle,
			Location: r.Location,
		})
	}
	return result
}

func (a *Assembler) ConvertSeats(src []*biz.Seat) []*pb.Seat {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.Seat, 0, len(src))
	for _, seat := range src {
		result = append(result, &pb.Seat{
			LabName:  seat.LabName,
			KindName: seat.RoomName,
			DevId:    seat.DevID,
			DevName:  seat.DevName,
			Ts:       a.ConvertTimeSlots(seat.Ts),
		})
	}
	return result
}

func (a *Assembler) ConvertDiscussions(src []*biz.Discussion) []*pb.Discussion {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.Discussion, 0, len(src))
	for _, d := range src {
		result = append(result, &pb.Discussion{
			LabId:    d.LabID,
			LabName:  d.LabName,
			KindId:   d.KindID,
			KindName: d.KindName,
			DevId:    d.DevID,
			DevName:  d.DevName,
			TS:       a.ConvertDiscussionTS(d.TS),
		})
	}
	return result
}

func (a *Assembler) ConvertGetSeatResponse(data map[string][]*biz.Seat) *pb.GetSeatResponse {
	if len(data) == 0 {
		return &pb.GetSeatResponse{}
	}
	result := &pb.GetSeatResponse{
		RoomSeats: make([]*pb.RoomSeat, 0, len(data)),
	}
	for roomID, seats := range data {
		result.RoomSeats = append(result.RoomSeats, &pb.RoomSeat{
			RoomId: roomID,
			Seats:  a.ConvertSeats(seats),
		})
	}
	return result
}

func (c *Assembler) ConvertMessages(data []*biz.Comment) *pb.GetCommentResp {
	if len(data) == 0 {
		return &pb.GetCommentResp{}
	}

	result := make([]*pb.Comment, 0, len(data))
	for _, r := range data {
		result = append(result, &pb.Comment{
			Id:        int64(r.ID),
			SeatId:    r.SeatID,
			Username:  r.Username,
			Content:   r.Content,
			Rating:    int64(r.Rating),
			CreatedAt: r.CreatedAt.String(),
		})
	}

	return &pb.GetCommentResp{
		Comment: result,
	}
}
