package store

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/log/logtest"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/shared/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/uploads/shared"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/precise"
)

func TestUpdatePackageReferences(t *testing.T) {
	logger := logtest.Scoped(t)
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	store := New(db, &observation.TestContext)

	// for foreign key relation
	insertUploads(t, db, types.Upload{ID: 42})

	if err := store.UpdatePackageReferences(context.Background(), 42, []precise.PackageReference{
		{Package: precise.Package{Scheme: "s0", Name: "n0", Version: "v0"}},
		{Package: precise.Package{Scheme: "s1", Name: "n1", Version: "v1"}},
		{Package: precise.Package{Scheme: "s2", Name: "n2", Version: "v2"}},
		{Package: precise.Package{Scheme: "s3", Name: "n3", Version: "v3"}},
		{Package: precise.Package{Scheme: "s4", Name: "n4", Version: "v4"}},
		{Package: precise.Package{Scheme: "s5", Name: "n5", Version: "v5"}},
		{Package: precise.Package{Scheme: "s6", Name: "n6", Version: "v6"}},
		{Package: precise.Package{Scheme: "s7", Name: "n7", Version: "v7"}},
		{Package: precise.Package{Scheme: "s8", Name: "n8", Version: "v8"}},
		{Package: precise.Package{Scheme: "s9", Name: "n9", Version: "v9"}},
	}); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(db.QueryContext(context.Background(), "SELECT COUNT(*) FROM lsif_references"))
	if err != nil {
		t.Fatalf("unexpected error checking reference count: %s", err)
	}
	if count != 10 {
		t.Errorf("unexpected reference count. want=%d have=%d", 10, count)
	}
}

func TestReferencesForUpload(t *testing.T) {
	logger := logtest.Scoped(t)
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	store := New(db, &observation.TestContext)

	insertUploads(t, db,
		types.Upload{ID: 1, Commit: makeCommit(2), Root: "sub1/"},
		types.Upload{ID: 2, Commit: makeCommit(3), Root: "sub2/"},
		types.Upload{ID: 3, Commit: makeCommit(4), Root: "sub3/"},
		types.Upload{ID: 4, Commit: makeCommit(3), Root: "sub4/"},
		types.Upload{ID: 5, Commit: makeCommit(2), Root: "sub5/"},
	)

	insertPackageReferences(t, store, []shared.PackageReference{
		{Package: shared.Package{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "1.1.0"}},
		{Package: shared.Package{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "2.1.0"}},
		{Package: shared.Package{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "3.1.0"}},
		{Package: shared.Package{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "4.1.0"}},
		{Package: shared.Package{DumpID: 3, Scheme: "gomod", Name: "leftpad", Version: "5.1.0"}},
	})

	scanner, err := store.ReferencesForUpload(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error getting filters: %s", err)
	}

	filters, err := consumeScanner(scanner)
	if err != nil {
		t.Fatalf("unexpected error from scanner: %s", err)
	}

	expected := []shared.PackageReference{
		{Package: shared.Package{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "2.1.0"}},
		{Package: shared.Package{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "3.1.0"}},
		{Package: shared.Package{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "4.1.0"}},
	}
	if diff := cmp.Diff(expected, filters); diff != "" {
		t.Errorf("unexpected filters (-want +got):\n%s", diff)
	}
}
