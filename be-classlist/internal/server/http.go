package server

//作为rpc下游微服务,http暂时弃用
//// NewHTTPServer new an HTTP server.
//func NewHTTPServer(c *conf.Server, greeter *service.ClasserService, logger log.Logger) *http.Server {
//	var opts = []http.ServerOption{
//		http.Middleware(
//			recovery.Recovery(),
//			metrics.QPSMiddleware(),
//			metrics.DelayMiddleware(),
//			validate.Validator(),
//		),
//		http.ResponseEncoder(encoder.RespEncoder), // Notice: 将响应格式化
//	}
//	if c.Http.Network != "" {
//		opts = append(opts, http.Network(c.Http.Network))
//	}
//	if c.Http.Addr != "" {
//		opts = append(opts, http.Address(c.Http.Addr))
//	}
//	if c.Http.Timeout != nil {
//		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
//	}
//	srv := http.NewServer(opts...)
//	srv.Handle("/metrics", promhttp.Handler())
//	v1.RegisterClasserHTTPServer(srv, greeter)
//	return srv
//}
