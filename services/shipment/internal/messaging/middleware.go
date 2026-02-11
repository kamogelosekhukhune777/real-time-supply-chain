package messaging

import (
	"context"

	"google.golang.org/protobuf/proto"
)

type Handler[T proto.Message] func(ctx context.Context, evt T) error

type Middleware[T proto.Message] func(Handler[T]) Handler[T]

func wrapMiddleware[T proto.Message](mw []Middleware[T], handler Handler[T]) Handler[T] {
	for i := len(mw) - 1; i >= 0; i-- {
		mwFunc := mw[i]
		if mwFunc != nil {
			handler = mwFunc(handler)
		}
	}

	return handler
}
