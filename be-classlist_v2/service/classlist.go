package service

import (
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1" // 此处改成了api中的,方便其他服务调用.
)

type ClassListService struct {
	pb.UnimplementedClasserServer
}
