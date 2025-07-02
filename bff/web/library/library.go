package library

import (
	libraryv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/library/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type LibraryHandler struct {
	LibraryClient  libraryv1.LibraryClient // 注入 grpc 服务
	Administrators map[string]struct{}
}

func NewLibraryHandler(client libraryv1.LibraryClient, admins map[string]struct{}) *LibraryHandler {
	return &LibraryHandler{
		LibraryClient:  client,
		Administrators: admins,
	}
}

func (h *LibraryHandler) RegisterRoutes(s *gin.RouterGroup, authMiddlerware gin.HandlerFunc) {
	sg := s.Group("/library")
	sg.POST("/get_seat", authMiddlerware, ginx.WrapClaimsAndReq(h.GetSeatInfos))
}

func (h *LibraryHandler) GetSeatInfos(ctx *gin.Context, req GetSeatRequest, uc ijwt.UserClaims) (web.Response, error) {
	res, err := h.LibraryClient.GetSeat(ctx, &libraryv1.GetSeatRequest{
		StuId:  req.StuID,
		RoomId: req.RoomID,
	})
	if err != nil {
		return web.Response{}, errs.GET_SEAT_ERROR(err)
	}

	var respSeats = make([]Seat, 0, len(res.Seat))

	for _, seat := range res.Seat {
		var timeSlots []TimeSlot
		for _, ts := range seat.Ts {
			timeSlots = append(timeSlots, TimeSlot{
				Start: ts.Start,
				End:   ts.End,
				Owner: ts.Owner,
			})
		}

		respSeats = append(respSeats, Seat{
			Name:      seat.Name,
			DevID:     seat.DevId,
			KindName:  seat.KindName,
			TimeSlots: timeSlots,
		})
	}

	resp := GetSeatResponse{
		Seats: respSeats,
	}

	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}
