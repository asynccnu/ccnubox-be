package grpcx

type Server interface {
	Serve() error
	Close() error
}
