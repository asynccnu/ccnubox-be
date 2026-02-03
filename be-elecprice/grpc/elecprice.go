package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-elecprice/domain"
	"github.com/asynccnu/ccnubox-be/be-elecprice/service"
	v1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	"google.golang.org/grpc"
)

type ElecpriceServiceServer struct {
	v1.UnimplementedElecpriceServiceServer

	ser service.ElecpriceService
}

func NewElecpriceGrpcService(ser service.ElecpriceService) *ElecpriceServiceServer {
	return &ElecpriceServiceServer{ser: ser}
}

func (s *ElecpriceServiceServer) Register(server grpc.ServiceRegistrar) {
	v1.RegisterElecpriceServiceServer(server, s)
}

func (s *ElecpriceServiceServer) GetArchitecture(ctx context.Context, req *v1.GetArchitectureRequest) (*v1.GetArchitectureResponse, error) {
	res, err := s.ser.GetArchitecture(ctx, req.AreaName)
	if err != nil {
		return nil, err
	}

	var resp v1.GetArchitectureResponse
	for _, a := range res.ArchitectureInfoList.ArchitectureInfo {
		resp.ArchitectureList = append(resp.ArchitectureList, &v1.GetArchitectureResponse_Architecture{
			ArchitectureName: a.ArchitectureName,
			ArchitectureID:   a.ArchitectureID,
			BaseFloor:        a.ArchitectureBegin,
			TopFloor:         a.ArchitectureStorys,
		})
	}
	return &resp, nil
}

func (s *ElecpriceServiceServer) GetRoomInfo(ctx context.Context, req *v1.GetRoomInfoRequest) (*v1.GetRoomInfoResponse, error) {
	res, err := s.ser.GetRoomInfo(ctx, req.ArchitectureID, req.Floor)
	if err != nil {
		return nil, err
	}

	var resp = v1.GetRoomInfoResponse{
		RoomList: []*v1.GetRoomInfoResponse_Room{},
	}
	for _, v := range res.Rooms {
		resp.RoomList = append(resp.RoomList, &v1.GetRoomInfoResponse_Room{
			RoomName: v.RoomName,
			AC:       v.AC,
			Light:    v.Light,
			Union:    v.Union,
		})
	}
	return &resp, nil
}

func (s *ElecpriceServiceServer) GetPrice(ctx context.Context, req *v1.GetPriceRequest) (*v1.GetPriceResponse, error) {
	res, err := s.ser.GetPriceByName(ctx, req.RoomName)
	if err != nil {
		return nil, err
	}

	return &v1.GetPriceResponse{
		Ac:    res.AC.ToProto(),
		Light: res.Light.ToProto(),
		Union: res.Union.ToProto(),
	}, nil
}

func (s *ElecpriceServiceServer) SetStandard(ctx context.Context, req *v1.SetStandardRequest) (*v1.SetStandardResponse, error) {
	err := s.ser.SetStandard(ctx, &domain.SetStandardRequest{
		StudentId: req.StudentId,
		Standard: &domain.Standard{
			Limit:    req.Standard.Limit,
			RoomId:   req.Standard.RoomId,
			RoomName: req.Standard.RoomName,
		},
	})

	return &v1.SetStandardResponse{}, err
}

func (s *ElecpriceServiceServer) GetStandardList(ctx context.Context, req *v1.GetStandardListRequest) (*v1.GetStandardListResponse, error) {
	res, err := s.ser.GetStandardList(ctx, &domain.GetStandardListRequest{
		StudentId: req.StudentId,
	})
	if err != nil {
		return nil, err
	}

	var resp v1.GetStandardListResponse
	for _, s := range res.Standard {
		resp.Standards = append(resp.Standards, &v1.Standard{
			Limit:    s.Limit,
			RoomId:   s.RoomId,
			RoomName: s.RoomName,
		})
	}
	return &resp, nil
}

func (s *ElecpriceServiceServer) CancelStandard(ctx context.Context, req *v1.CancelStandardRequest) (*v1.CancelStandardResponse, error) {
	err := s.ser.CancelStandard(ctx, &domain.CancelStandardRequest{
		StudentId: req.StudentId,
		RoomId:    req.RoomId,
	})

	return &v1.CancelStandardResponse{}, err
}

func (s *ElecpriceServiceServer) GetBillingBalance(ctx context.Context, req *v1.GetBillingBalanceRequest) (*v1.GetBillingBalanceResponse, error) {
	res, err := s.ser.GetPriceById(ctx, req.RoomId)
	if err != nil {
		return nil, err
	}

	return &v1.GetBillingBalanceResponse{
		Price: res.ToProto(),
	}, nil
}
