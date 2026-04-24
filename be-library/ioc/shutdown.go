package ioc

import "context"

func InitNoopShutdown() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return nil
	}
}
