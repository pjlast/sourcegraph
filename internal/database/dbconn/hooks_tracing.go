package dbconn

import (
	"context"
	"fmt"
	"strconv"

	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/qustavo/sqlhooks/v2"

	"github.com/sourcegraph/sourcegraph/internal/trace"
)

type tracingHooks struct{}

var _ sqlhooks.Hooks = &tracingHooks{}
var _ sqlhooks.OnErrorer = &tracingHooks{}

func (h *tracingHooks) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	if BulkInsertion(ctx) {
		query = string(postgresBulkInsertRowsPattern.ReplaceAll([]byte(query), postgresBulkInsertRowsReplacement))
	}

	tr, ctx := trace.New(ctx, "sql", query,
		trace.Tag{Key: "span.kind", Value: "client"},
		trace.Tag{Key: "database.type", Value: "sql"},
	)

	if !BulkInsertion(ctx) {
		tr.LogFields(otlog.Lazy(func(fv otlog.Encoder) {
			emittedChars := 0
			for i, arg := range args {
				k := strconv.Itoa(i + 1)
				v := fmt.Sprintf("%v", arg)
				emittedChars += len(k) + len(v)
				// Limit the amount of characters reported in a span because
				// a Jaeger span may not exceed 65k. Usually, the arguments are
				// not super helpful if it's so many of them anyways.
				if emittedChars > 32768 {
					fv.EmitString("more omitted", strconv.Itoa(len(args)-i))
					break
				}
				fv.EmitString(k, v)
			}
		}))
	} else {
		tr.LogFields(otlog.Bool("bulk_insert", true), otlog.Int("num_args", len(args)))
	}

	return ctx, nil
}

func (h *tracingHooks) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	if tr := trace.TraceFromContext(ctx); tr != nil {
		tr.Finish()
	}

	return ctx, nil
}

func (h *tracingHooks) OnError(ctx context.Context, err error, query string, args ...interface{}) error {
	if tr := trace.TraceFromContext(ctx); tr != nil {
		tr.SetError(err)
		tr.Finish()
	}

	return err
}
