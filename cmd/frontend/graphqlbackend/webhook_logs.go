package graphqlbackend

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/log"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/auth"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/internal/gqlutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// webhookLogArgs are the arguments common to the two queries that provide
// access to webhook logs: the webhookLogs method on the top level query, and on
// the ExternalService type.
type webhookLogsArgs struct {
	graphqlutil.ConnectionArgs
	After      *string
	OnlyErrors *bool
	Since      *time.Time
	Until      *time.Time
	WebhookID  *graphql.ID
}

// webhookLogsExternalServiceID is used to represent an external service ID,
// which may be a constant defined below to represent all or unmatched external
// services.
type webhookLogsExternalServiceID int64

var (
	webhookLogsAllExternalServices      webhookLogsExternalServiceID = -1
	webhookLogsUnmatchedExternalService webhookLogsExternalServiceID = 0
)

func (id webhookLogsExternalServiceID) toListOpt() *int64 {
	switch id {
	case webhookLogsAllExternalServices:
		return nil
	case webhookLogsUnmatchedExternalService:
		fallthrough
	default:
		i := int64(id)
		return &i
	}
}

// toListOpts transforms the GraphQL webhookLogsArgs into options that can be
// provided to the WebhookLogStore's Count and List methods.
func (args *webhookLogsArgs) toListOpts(externalServiceID webhookLogsExternalServiceID) (database.WebhookLogListOpts, error) {
	opts := database.WebhookLogListOpts{
		ExternalServiceID: externalServiceID.toListOpt(),
		Since:             args.Since,
		Until:             args.Until,
	}

	if args.First != nil {
		opts.Limit = int(*args.First)
	} else {
		opts.Limit = 50
	}

	if args.After != nil {
		var err error
		opts.Cursor, err = strconv.ParseInt(*args.After, 10, 64)
		if err != nil {
			return opts, errors.Wrap(err, "parsing the after cursor")
		}
	}

	if args.OnlyErrors != nil && *args.OnlyErrors {
		opts.OnlyErrors = true
	}

	// Both nil and "-1" webhook IDs should be resolved to nil WebhookID
	// WebhookLogListOpts option
	if args.WebhookID != nil {
		id, err := unmarshalWebhookID(*args.WebhookID)
		if err != nil {
			return opts, errors.Wrap(err, "unmarshalling webhook ID")
		}
		if id > 0 {
			opts.WebhookID = &id
		}
	}

	return opts, nil
}

type globalWebhookLogsArgs struct {
	webhookLogsArgs
	OnlyUnmatched *bool
}

// WebhookLogs is the top level query used to return webhook logs that weren't
// resolved to a specific external service.
func (r *schemaResolver) WebhookLogs(ctx context.Context, args *globalWebhookLogsArgs) (*webhookLogConnectionResolver, error) {
	externalServiceID := webhookLogsAllExternalServices
	if unmatched := args.OnlyUnmatched; unmatched != nil && *unmatched {
		externalServiceID = webhookLogsUnmatchedExternalService
	}

	return newWebhookLogConnectionResolver(ctx, r.db, &args.webhookLogsArgs, externalServiceID)
}

type webhookLogConnectionResolver struct {
	logger            log.Logger
	args              *webhookLogsArgs
	externalServiceID webhookLogsExternalServiceID
	store             database.WebhookLogStore

	once sync.Once
	logs []*types.WebhookLog
	next int64
	err  error
}

func newWebhookLogConnectionResolver(
	ctx context.Context, db database.DB, args *webhookLogsArgs,
	externalServiceID webhookLogsExternalServiceID,
) (*webhookLogConnectionResolver, error) {
	if err := auth.CheckCurrentUserIsSiteAdmin(ctx, db); err != nil {
		return nil, err
	}

	return &webhookLogConnectionResolver{
		logger:            log.Scoped("webhookLogConnectionResolver", ""),
		args:              args,
		externalServiceID: externalServiceID,
		store:             db.WebhookLogs(keyring.Default().WebhookLogKey),
	}, nil
}

func (r *webhookLogConnectionResolver) Nodes(ctx context.Context) ([]*webhookLogResolver, error) {
	logs, _, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make([]*webhookLogResolver, len(logs))
	db := database.NewDBWith(r.logger, r.store)
	for i, log := range logs {
		nodes[i] = &webhookLogResolver{
			db:  db,
			log: log,
		}
	}

	return nodes, nil
}

func (r *webhookLogConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	opts, err := r.args.toListOpts(r.externalServiceID)
	if err != nil {
		return 0, err
	}

	count, err := r.store.Count(ctx, opts)
	return int32(count), err
}

func (r *webhookLogConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	_, next, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}

	if next == 0 {
		return graphqlutil.HasNextPage(false), nil
	}
	return graphqlutil.NextPageCursor(fmt.Sprint(next)), nil
}

func (r *webhookLogConnectionResolver) compute(ctx context.Context) ([]*types.WebhookLog, int64, error) {
	r.once.Do(func() {
		r.err = func() error {
			opts, err := r.args.toListOpts(r.externalServiceID)
			if err != nil {
				return err
			}

			r.logs, r.next, err = r.store.List(ctx, opts)
			return err
		}()
	})

	return r.logs, r.next, r.err
}

type webhookLogResolver struct {
	db  database.DB
	log *types.WebhookLog
}

func marshalWebhookLogID(id int64) graphql.ID {
	return relay.MarshalID("WebhookLog", id)
}

func unmarshalWebhookLogID(id graphql.ID) (logID int64, err error) {
	err = relay.UnmarshalSpec(id, &logID)
	return
}

func webhookLogByID(ctx context.Context, db database.DB, gqlID graphql.ID) (*webhookLogResolver, error) {
	if err := auth.CheckCurrentUserIsSiteAdmin(ctx, db); err != nil {
		return nil, err
	}

	id, err := unmarshalWebhookLogID(gqlID)
	if err != nil {
		return nil, err
	}

	log, err := db.WebhookLogs(keyring.Default().WebhookLogKey).GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &webhookLogResolver{db: db, log: log}, nil
}

func (r *webhookLogResolver) ID() graphql.ID {
	return marshalWebhookLogID(r.log.ID)
}

func (r *webhookLogResolver) ReceivedAt() gqlutil.DateTime {
	return gqlutil.DateTime{Time: r.log.ReceivedAt}
}

func (r *webhookLogResolver) ExternalService(ctx context.Context) (*externalServiceResolver, error) {
	if r.log.ExternalServiceID == nil {
		return nil, nil
	}

	return externalServiceByID(ctx, r.db, MarshalExternalServiceID(*r.log.ExternalServiceID))
}

func (r *webhookLogResolver) StatusCode() int32 {
	return int32(r.log.StatusCode)
}

func (r *webhookLogResolver) Request(ctx context.Context) (*webhookLogRequestResolver, error) {
	message, err := r.log.Request.Decrypt(ctx)
	if err != nil {
		return nil, err
	}

	return &webhookLogRequestResolver{webhookLogMessageResolver{message: &message}}, nil
}

func (r *webhookLogResolver) Response(ctx context.Context) (*webhookLogMessageResolver, error) {
	message, err := r.log.Response.Decrypt(ctx)
	if err != nil {
		return nil, err
	}

	return &webhookLogMessageResolver{message: &message}, nil
}

type webhookLogMessageResolver struct {
	message *types.WebhookLogMessage
}

func (r *webhookLogMessageResolver) Headers() ([]*HttpHeaders, error) {
	return newHttpHeaders(r.message.Header)
}

func (r *webhookLogMessageResolver) Body() string {
	return string(r.message.Body)
}

type webhookLogRequestResolver struct {
	webhookLogMessageResolver
}

func (r *webhookLogRequestResolver) Method() string {
	return r.message.Method
}

func (r *webhookLogRequestResolver) URL() string {
	return r.message.URL
}

func (r *webhookLogRequestResolver) Version() string {
	return r.message.Version
}

func marshalWebhookID(id int32) graphql.ID {
	return relay.MarshalID("Webhook", id)
}

func unmarshalWebhookID(id graphql.ID) (hookID int32, err error) {
	err = relay.UnmarshalSpec(id, &hookID)
	return
}
