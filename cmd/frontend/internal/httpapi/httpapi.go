package httpapi

import (
	"context"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/graph-gophers/graphql-go"

	sglog "github.com/sourcegraph/log"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/updatecheck"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/handlerutil"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/httpapi/releasecache"
	apirouter "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/httpapi/router"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/httpapi/webhookhandlers"
	frontendsearch "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/search"
	registry "github.com/sourcegraph/sourcegraph/cmd/frontend/registry/api"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/webhooks"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/searchcontexts"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type Handlers struct {
	GitHubSyncWebhook               webhooks.Registerer
	BatchesGitHubWebhook            webhooks.Registerer
	BatchesGitLabWebhook            webhooks.RegistererHandler
	BatchesBitbucketServerWebhook   webhooks.RegistererHandler
	BatchesBitbucketCloudWebhook    webhooks.RegistererHandler
	BatchesChangesFileGetHandler    http.Handler
	BatchesChangesFileExistsHandler http.Handler
	BatchesChangesFileUploadHandler http.Handler
	NewCodeIntelUploadHandler       enterprise.NewCodeIntelUploadHandler
	NewComputeStreamHandler         enterprise.NewComputeStreamHandler
}

// NewHandler returns a new API handler that uses the provided API
// router, which must have been created by httpapi/router.New, or
// creates a new one if nil.
//
// 🚨 SECURITY: The caller MUST wrap the returned handler in middleware that checks authentication
// and sets the actor in the request context.
func NewHandler(
	db database.DB,
	m *mux.Router,
	schema *graphql.Schema,
	rateLimiter graphqlbackend.LimitWatcher,
	handlers *Handlers,
) http.Handler {
	logger := sglog.Scoped("Handler", "frontend HTTP API handler")

	if m == nil {
		m = apirouter.New(nil)
	}
	m.StrictSlash(true)

	handler := jsonMiddleware(&errorHandler{
		Logger: logger,
		// Only display error message to admins when in debug mode, since it
		// may contain sensitive info (like API keys in net/http error
		// messages).
		WriteErrBody: env.InsecureDev,
	})

	// Set handlers for the installed routes.
	m.Get(apirouter.RepoShield).Handler(trace.Route(handler(serveRepoShield(db))))
	m.Get(apirouter.RepoRefresh).Handler(trace.Route(handler(serveRepoRefresh(db))))

	webhookMiddleware := webhooks.NewLogMiddleware(
		db.WebhookLogs(keyring.Default().WebhookLogKey),
	)

	wh := webhooks.WebhookRouter{
		DB: db,
	}
	webhookhandlers.Init(&wh)
	handlers.BatchesGitHubWebhook.Register(&wh)
	handlers.BatchesGitLabWebhook.Register(&wh)
	handlers.BatchesBitbucketServerWebhook.Register(&wh)
	handlers.BatchesBitbucketCloudWebhook.Register(&wh)
	handlers.GitHubSyncWebhook.Register(&wh)

	// 🚨 SECURITY: This handler implements its own secret-based auth
	webhookHandler := webhooks.NewHandler(logger, db, &wh)

	gitHubWebhook := webhooks.GitHubWebhook{WebhookRouter: &wh}

	// New UUID based webhook handler
	m.Get(apirouter.Webhooks).Handler(trace.Route(webhookMiddleware.Logger(webhookHandler)))

	// Old, soon to be deprecated, webhook handlers
	m.Get(apirouter.GitHubWebhooks).Handler(trace.Route(webhookMiddleware.Logger(&gitHubWebhook)))
	m.Get(apirouter.GitLabWebhooks).Handler(trace.Route(webhookMiddleware.Logger(handlers.BatchesGitLabWebhook)))
	m.Get(apirouter.BitbucketServerWebhooks).Handler(trace.Route(webhookMiddleware.Logger(handlers.BatchesBitbucketServerWebhook)))
	m.Get(apirouter.BitbucketCloudWebhooks).Handler(trace.Route(webhookMiddleware.Logger(handlers.BatchesBitbucketCloudWebhook)))

	m.Get(apirouter.BatchesFileGet).Handler(trace.Route(handlers.BatchesChangesFileGetHandler))
	m.Get(apirouter.BatchesFileExists).Handler(trace.Route(handlers.BatchesChangesFileExistsHandler))
	m.Get(apirouter.BatchesFileUpload).Handler(trace.Route(handlers.BatchesChangesFileUploadHandler))
	m.Get(apirouter.LSIFUpload).Handler(trace.Route(handlers.NewCodeIntelUploadHandler(true)))
	m.Get(apirouter.ComputeStream).Handler(trace.Route(handlers.NewComputeStreamHandler()))

	if envvar.SourcegraphDotComMode() {
		m.Path("/updates").Methods("GET", "POST").Name("updatecheck").Handler(trace.Route(http.HandlerFunc(updatecheck.HandlerWithLog(logger))))
	}

	m.Get(apirouter.GraphQL).Handler(trace.Route(handler(serveGraphQL(logger, schema, rateLimiter, false))))

	m.Get(apirouter.SearchStream).Handler(trace.Route(frontendsearch.StreamHandler(db)))

	// Return the minimum src-cli version that's compatible with this instance
	m.Get(apirouter.SrcCli).Handler(trace.Route(newSrcCliVersionHandler(logger)))

	gsClient := gitserver.NewClient(db)
	m.Get(apirouter.GitBlameStream).Handler(trace.Route(handleStreamBlame(logger, db, gsClient)))

	// Set up the src-cli version cache handler (this will effectively be a
	// no-op anywhere other than dot-com).
	m.Get(apirouter.SrcCliVersionCache).Handler(trace.Route(releasecache.NewHandler(logger)))

	m.Get(apirouter.Registry).Handler(trace.Route(handler(registry.HandleRegistry(db))))

	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("API no route: %s %s from %s", r.Method, r.URL, r.Referer())
		http.Error(w, "no route", http.StatusNotFound)
	})

	return m
}

// NewInternalHandler returns a new API handler for internal endpoints that uses
// the provided API router, which must have been created by httpapi/router.NewInternal.
//
// 🚨 SECURITY: This handler should not be served on a publicly exposed port. 🚨
// This handler is not guaranteed to provide the same authorization checks as
// public API handlers.
func NewInternalHandler(
	m *mux.Router,
	db database.DB,
	schema *graphql.Schema,
	newCodeIntelUploadHandler enterprise.NewCodeIntelUploadHandler,
	rankingService enterprise.RankingService,
	newComputeStreamHandler enterprise.NewComputeStreamHandler,
	rateLimitWatcher graphqlbackend.LimitWatcher,
) http.Handler {
	logger := sglog.Scoped("InternalHandler", "frontend internal HTTP API handler")
	if m == nil {
		m = apirouter.New(nil)
	}
	m.StrictSlash(true)

	handler := jsonMiddleware(&errorHandler{
		Logger: logger,
		// Internal endpoints can expose sensitive errors
		WriteErrBody: true,
	})

	m.Get(apirouter.ExternalServiceConfigs).Handler(trace.Route(handler(serveExternalServiceConfigs(db))))

	// zoekt-indexserver endpoints
	gsClient := gitserver.NewClient(db)
	indexer := &searchIndexerServer{
		db:              db,
		logger:          logger.Scoped("searchIndexerServer", "zoekt-indexserver endpoints"),
		gitserverClient: gsClient,
		ListIndexable:   backend.NewRepos(logger, db, gsClient).ListIndexable,
		RepoStore:       db.Repos(),
		SearchContextsRepoRevs: func(ctx context.Context, repoIDs []api.RepoID) (map[api.RepoID][]string, error) {
			return searchcontexts.RepoRevs(ctx, db, repoIDs)
		},
		Indexers:               search.Indexers(),
		Ranking:                rankingService,
		MinLastChangedDisabled: os.Getenv("SRC_SEARCH_INDEXER_EFFICIENT_POLLING_DISABLED") != "",
	}
	m.Get(apirouter.SearchConfiguration).Handler(trace.Route(handler(indexer.serveConfiguration)))
	m.Get(apirouter.ReposIndex).Handler(trace.Route(handler(indexer.serveList)))
	m.Get(apirouter.RepoRank).Handler(trace.Route(handler(indexer.serveRepoRank)))
	m.Get(apirouter.DocumentRanks).Handler(trace.Route(handler(indexer.serveDocumentRanks)))
	m.Get(apirouter.UpdateIndexStatus).Handler(trace.Route(handler(indexer.handleIndexStatusUpdate)))

	m.Get(apirouter.ExternalURL).Handler(trace.Route(handler(serveExternalURL)))
	m.Get(apirouter.SendEmail).Handler(trace.Route(handler(serveSendEmail)))
	gitService := &gitServiceHandler{
		Gitserver: gsClient,
	}
	m.Get(apirouter.GitInfoRefs).Handler(trace.Route(handler(gitService.serveInfoRefs())))
	m.Get(apirouter.GitUploadPack).Handler(trace.Route(handler(gitService.serveGitUploadPack())))
	m.Get(apirouter.Telemetry).Handler(trace.Route(telemetryHandler(db)))
	m.Get(apirouter.GraphQL).Handler(trace.Route(handler(serveGraphQL(logger, schema, rateLimitWatcher, true))))
	m.Get(apirouter.Configuration).Handler(trace.Route(handler(serveConfiguration)))
	m.Path("/ping").Methods("GET").Name("ping").HandlerFunc(handlePing)
	m.Get(apirouter.StreamingSearch).Handler(trace.Route(frontendsearch.StreamHandler(db)))
	m.Get(apirouter.ComputeStream).Handler(trace.Route(newComputeStreamHandler()))

	m.Get(apirouter.LSIFUpload).Handler(trace.Route(newCodeIntelUploadHandler(false)))

	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("API no route: %s %s from %s", r.Method, r.URL, r.Referer())
		http.Error(w, "no route", http.StatusNotFound)
	})

	return m
}

var schemaDecoder = schema.NewDecoder()

func init() {
	schemaDecoder.IgnoreUnknownKeys(true)

	// Register a converter for unix timestamp strings -> time.Time values
	// (needed for Appdash PageLoadEvent type).
	schemaDecoder.RegisterConverter(time.Time{}, func(s string) reflect.Value {
		ms, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return reflect.ValueOf(err)
		}
		return reflect.ValueOf(time.Unix(0, ms*int64(time.Millisecond)))
	})
}

type errorHandler struct {
	// Logger is required
	Logger sglog.Logger

	WriteErrBody bool
}

func (h *errorHandler) Handle(w http.ResponseWriter, r *http.Request, status int, err error) {
	logger := trace.Logger(r.Context(), h.Logger)

	trace.SetRequestErrorCause(r.Context(), err)

	// Handle custom errors
	var e *handlerutil.URLMovedError
	if errors.As(err, &e) {
		err := handlerutil.RedirectToNewRepoName(w, r, e.NewRepo)
		if err != nil {
			logger.Error("error redirecting to new URI",
				sglog.Error(err),
				sglog.String("new_url", string(e.NewRepo)))
		}
		return
	}

	// Never cache error responses.
	w.Header().Set("cache-control", "no-cache, max-age=0")

	errBody := err.Error()

	var displayErrBody string
	if h.WriteErrBody {
		displayErrBody = errBody
	}
	http.Error(w, displayErrBody, status)

	if status < 200 || status >= 500 {
		logger.Error("API HTTP handler error response",
			sglog.String("method", r.Method),
			sglog.String("request_uri", r.URL.RequestURI()),
			sglog.Int("status_code", status),
			sglog.Error(err))
	}
}

func jsonMiddleware(errorHandler *errorHandler) func(func(http.ResponseWriter, *http.Request) error) http.Handler {
	return func(h func(http.ResponseWriter, *http.Request) error) http.Handler {
		return handlerutil.HandlerWithErrorReturn{
			Handler: func(w http.ResponseWriter, r *http.Request) error {
				w.Header().Set("Content-Type", "application/json")
				return h(w, r)
			},
			Error: errorHandler.Handle,
		}
	}
}
