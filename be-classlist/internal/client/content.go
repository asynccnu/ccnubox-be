package client

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	b_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
)

func InitContentClient(r *etcd.Registry, cf *conf.Registry, logger log.Logger, env *b_conf.Env) (contentv1.ContentServiceClient, error) {
	conn, err := InitClient(r, cf.Contentsvc, env)
	if err != nil {
		log.NewHelper(logger).WithContext(context.Background()).Errorw("kind", "grpc-client", "reason", "GRPC_CLIENT_INIT_ERROR", "err", err)
		return nil, err
	}

	return contentv1.NewContentServiceClient(conn), nil
}
