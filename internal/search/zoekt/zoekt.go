package zoekt

import (
	"context"
	"regexp/syntax" //nolint:depguard // zoekt requires this pkg
	"time"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/sourcegraph/zoekt"
	zoektquery "github.com/sourcegraph/zoekt/query"

	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/filter"
	"github.com/sourcegraph/sourcegraph/internal/search/limits"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
	"github.com/sourcegraph/sourcegraph/internal/trace/policy"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

var defaultTimeout = 20 * time.Second

func FileRe(pattern string, queryIsCaseSensitive bool) (zoektquery.Q, error) {
	return parseRe(pattern, true, false, queryIsCaseSensitive)
}

func noOpAnyChar(re *syntax.Regexp) {
	if re.Op == syntax.OpAnyChar {
		re.Op = syntax.OpAnyCharNotNL
	}
	for _, s := range re.Sub {
		noOpAnyChar(s)
	}
}

const regexpFlags = syntax.ClassNL | syntax.PerlX | syntax.UnicodeGroups

func parseRe(pattern string, filenameOnly bool, contentOnly bool, queryIsCaseSensitive bool) (zoektquery.Q, error) {
	// these are the flags used by zoekt, which differ to searcher.
	re, err := syntax.Parse(pattern, regexpFlags)
	if err != nil {
		return nil, err
	}
	noOpAnyChar(re)

	// OptimizeRegexp currently only converts capture groups into non-capture
	// groups (faster for stdlib regexp to execute).
	re = zoektquery.OptimizeRegexp(re, regexpFlags)

	// zoekt decides to use its literal optimization at the query parser
	// level, so we check if our regex can just be a literal.
	if re.Op == syntax.OpLiteral {
		return &zoektquery.Substring{
			Pattern:       string(re.Rune),
			CaseSensitive: queryIsCaseSensitive,
			Content:       contentOnly,
			FileName:      filenameOnly,
		}, nil
	}
	return &zoektquery.Regexp{
		Regexp:        re,
		CaseSensitive: queryIsCaseSensitive,
		Content:       contentOnly,
		FileName:      filenameOnly,
	}, nil
}

func getSpanContext(ctx context.Context) (shouldTrace bool, spanContext map[string]string) {
	if !policy.ShouldTrace(ctx) {
		return false, nil
	}

	spanContext = make(map[string]string)
	if span := opentracing.SpanFromContext(ctx); span != nil {
		if err := ot.GetTracer(ctx).Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(spanContext)); err != nil {
			log15.Warn("Error injecting span context into map: %s", err)
			return true, nil
		}
	}
	return true, spanContext
}

// Options represents the inputs from Sourcegraph that we use to compute
// zoekt.SearchOptions.
type Options struct {
	Selector filter.SelectPath

	// FileMatchLimit is how many results the user wants.
	FileMatchLimit int32

	// NumRepos is the number of repos we are searching over. This number is
	// used as a heuristics to scale the amount of work we will do.
	NumRepos int

	// GlobalSearch is true if we are doing a search were we skip computing
	// NumRepos and instead rely on zoekt.
	GlobalSearch bool

	// Features are feature flags that can affect behaviour of searcher.
	Features search.Features
}

func (o *Options) ToSearch(ctx context.Context) *zoekt.SearchOptions {
	shouldTrace, spanContext := getSpanContext(ctx)
	searchOpts := &zoekt.SearchOptions{
		Trace:        shouldTrace,
		SpanContext:  spanContext,
		MaxWallTime:  defaultTimeout,
		ChunkMatches: true,
	}

	if o.Features.Debug {
		searchOpts.DebugScore = true
	}

	if o.Features.Ranking {
		limit := int(o.FileMatchLimit)

		// Tell each zoekt replica to not send back more than limit results.
		searchOpts.MaxDocDisplayCount = limit

		// These are reasonable default amounts of work to do per shard and
		// replica respectively.
		searchOpts.ShardMaxMatchCount = 10_000
		searchOpts.TotalMaxMatchCount = 100_000

		// If we are searching for large limits, raise the amount of work we
		// are willing to do per shard and zoekt replica respectively.
		if limit > searchOpts.ShardMaxMatchCount {
			searchOpts.ShardMaxMatchCount = limit
		}
		if limit > searchOpts.TotalMaxMatchCount {
			searchOpts.TotalMaxMatchCount = limit
		}

		// This enables our stream based ranking were we wait upto 500ms to
		// collect results before ranking.
		searchOpts.FlushWallTime = 500 * time.Millisecond

		// This enables the use of PageRank scores if they are available.
		searchOpts.UseDocumentRanks = true

		// This damps the impact of document ranks on the final ranking.
		searchOpts.RanksDampingFactor = 0.5

		return searchOpts
	}

	if userProbablyWantsToWaitLonger := o.FileMatchLimit > limits.DefaultMaxSearchResults; userProbablyWantsToWaitLonger {
		searchOpts.MaxWallTime *= time.Duration(3 * float64(o.FileMatchLimit) / float64(limits.DefaultMaxSearchResults))
	}

	if o.Selector.Root() == filter.Repository {
		searchOpts.ShardRepoMaxMatchCount = 1
	} else {
		k := o.resultCountFactor()
		searchOpts.ShardMaxMatchCount = 100 * k
		searchOpts.TotalMaxMatchCount = 100 * k
		// Ask for 2000 more results so we have results to populate
		// RepoStatusLimitHit.
		searchOpts.MaxDocDisplayCount = int(o.FileMatchLimit) + 2000
	}

	return searchOpts
}

func (o *Options) resultCountFactor() (k int) {
	if o.GlobalSearch {
		// for globalSearch, numRepos = 0, but effectively we are searching over all
		// indexed repos, hence k should be 1
		k = 1
	} else {
		// If we're only searching a small number of repositories, return more
		// comprehensive results. This is arbitrary.
		switch {
		case o.NumRepos <= 5:
			k = 100
		case o.NumRepos <= 10:
			k = 10
		case o.NumRepos <= 25:
			k = 8
		case o.NumRepos <= 50:
			k = 5
		case o.NumRepos <= 100:
			k = 3
		case o.NumRepos <= 500:
			k = 2
		default:
			k = 1
		}
	}
	if o.FileMatchLimit > limits.DefaultMaxSearchResults {
		k = int(float64(k) * 3 * float64(o.FileMatchLimit) / float64(limits.DefaultMaxSearchResults))
	}
	return k
}

// repoRevFunc is a function which maps repository names returned from Zoekt
// into the Sourcegraph's resolved repository revisions for the search.
type repoRevFunc func(file *zoekt.FileMatch) (repo types.MinimalRepo, revs []string)
