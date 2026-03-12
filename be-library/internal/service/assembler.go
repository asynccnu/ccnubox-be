package service

import (
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
)

type Assembler struct{}

func NewAssembler() *Assembler {
	return &Assembler{}
}

func (a *Assembler) ConvertFreeList(src []*biz.FreeTime) []*pb.FreeTime {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.FreeTime, 0, len(src))
	for _, ts := range src {
		result = append(result, &pb.FreeTime{
			Start: ts.Start,
			End:   ts.End,
		})
	}
	return result
}

func (a *Assembler) ConvertDisableList(src []*biz.DisableTime) []*pb.DisableTime {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.DisableTime, 0, len(src))
	for _, ts := range src {
		result = append(result, &pb.DisableTime{
			Start: ts.Start,
			End:   ts.End,
		})
	}
	return result
}

func (a *Assembler) ConvertRecords(src []*biz.Record) []*pb.Record {
	if len(src) == 0 {
		return nil
	}
	result := make([]*pb.Record, 0, len(src))
	for _, r := range src {
		result = append(result, &pb.Record{
			Id:        r.ID,
			RoomId:    r.RoomID,
			RoomName:  r.RoomName,
			BuildName: r.BuildName,
			FloorName: r.FloorName,
			SeatId:    r.SeatID,
			SeatLabel: r.SeatLabel,
			MakeBegin: r.MakeBegin.Format("2006-01-02 15:04"),
			MakeEnd:   r.MakeEnd.Format("2006-01-02 15:04"),
			MakeDate:  r.MakeDate.Format("2006-01-02"),
			Status:    r.Status,
			Message:   r.Message,
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
			ID:        seat.ID,
			Label:     seat.Label,
			Name:      seat.Name,
			Status:    seat.Status,
			AfterFree: seat.AfterFree,
			FreeList:  a.ConvertFreeList(seat.FreeList),
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
			RoomId:      d.RoomID,
			Name:        d.Name,
			RoomType:    d.RoomType,
			VenueId:     d.VenueID,
			Address:     d.Address,
			DisableList: a.ConvertDisableList(d.DisableList),
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
