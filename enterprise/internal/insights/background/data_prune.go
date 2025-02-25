package background

import (
	"context"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/log"

	edb "github.com/sourcegraph/sourcegraph/enterprise/internal/database"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/store"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// NewInsightsDataPrunerJob will periodically delete recorded data series that have been marked `deleted`.
func NewInsightsDataPrunerJob(ctx context.Context, postgres database.DB, insightsdb edb.InsightsDB) goroutine.BackgroundRoutine {
	interval := time.Minute * 60
	logger := log.Scoped("InsightsDataPrunerJob", "")

	return goroutine.NewPeriodicGoroutine(ctx, interval,
		goroutine.NewHandlerWithErrorMessage("insights_data_prune", func(ctx context.Context) (err error) {
			return performPurge(ctx, postgres, insightsdb, logger, time.Now().Add(interval))
		}))
}

func performPurge(ctx context.Context, postgres database.DB, insightsdb edb.InsightsDB, logger log.Logger, deletedBefore time.Time) (err error) {
	insightStore := store.NewInsightStore(insightsdb)
	timeseriesStore := store.New(insightsdb, store.NewInsightPermissionStore(postgres))

	// select the series that need to be pruned. These will be the series that are currently flagged
	// as "soft deleted" for a given amount of time.
	seriesIds, err := insightStore.GetSoftDeletedSeries(ctx, deletedBefore)
	if err != nil {
		return errors.Wrap(err, "GetSoftDeletedSeries")
	}

	for _, id := range seriesIds {
		// We will always delete the series definition last, such that any possible partial state
		// the series definition will always be referencable and therefore can be re-attempted. This operation
		// isn't across the same database currently, so there isn't a single transaction across all the
		// tables.
		logger.Info("pruning insight series", log.String("seriesId", id))
		err := deleteQueuedRecords(ctx, postgres, id)
		if err != nil {
			return errors.Wrap(err, "deleteQueuedRecords")
		}
		err = func() (err error) {
			// scope the transaction to an anonymous function so we can defer Done
			tx, err := timeseriesStore.Transact(ctx)
			if err != nil {
				return err
			}
			defer func() { err = tx.Done(err) }()

			err = tx.Delete(ctx, id)
			if err != nil {
				return errors.Wrap(err, "Delete")
			}

			insightStoreTx := insightStore.With(tx)
			return insightStoreTx.HardDeleteSeries(ctx, id)
		}()
		if err != nil {
			return err
		}
	}

	return err
}

func deleteQueuedRecords(ctx context.Context, postgres database.DB, seriesId string) error {
	queueStore := basestore.NewWithHandle(postgres.Handle())
	return queueStore.Exec(ctx, sqlf.Sprintf(deleteQueuedForSeries, seriesId))
}

const deleteQueuedForSeries = `
delete from insights_query_runner_jobs where series_id = %s;
`
