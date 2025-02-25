package graphqlbackend

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/externallink"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/highlight"
)

// FileContentFunc is a closure that returns the contents of a file and is used by the VirtualFileResolver.
type FileContentFunc func(ctx context.Context) (string, error)

func NewVirtualFileResolver(stat fs.FileInfo, fileContent FileContentFunc) *VirtualFileResolver {
	return &VirtualFileResolver{
		stat:        stat,
		fileContent: fileContent,
	}
}

type VirtualFileResolver struct {
	fileContent FileContentFunc
	// stat is this tree entry's file info. Its Name method must return the full path relative to
	// the root, not the basename.
	stat fs.FileInfo
}

func (r *VirtualFileResolver) Path() string      { return r.stat.Name() }
func (r *VirtualFileResolver) Name() string      { return path.Base(r.stat.Name()) }
func (r *VirtualFileResolver) IsDirectory() bool { return r.stat.Mode().IsDir() }

func (r *VirtualFileResolver) ToGitBlob() (*GitTreeEntryResolver, bool)    { return nil, false }
func (r *VirtualFileResolver) ToVirtualFile() (*VirtualFileResolver, bool) { return r, true }
func (r *VirtualFileResolver) ToBatchSpecWorkspaceFile() (BatchWorkspaceFileResolver, bool) {
	return nil, false
}

func (r *VirtualFileResolver) URL(ctx context.Context) (string, error) {
	// Todo: allow viewing arbitrary files in the webapp.
	return "", nil
}

func (r *VirtualFileResolver) CanonicalURL() string {
	// Todo: allow viewing arbitrary files in the webapp.
	return ""
}

func (r *VirtualFileResolver) ExternalURLs(ctx context.Context) ([]*externallink.Resolver, error) {
	// Todo: allow viewing arbitrary files in the webapp.
	return []*externallink.Resolver{}, nil
}

func (r *VirtualFileResolver) ByteSize(ctx context.Context) (int32, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len([]byte(content))), nil
}

func (r *VirtualFileResolver) Content(ctx context.Context) (string, error) {
	return r.fileContent(ctx)
}

func (r *VirtualFileResolver) RichHTML(ctx context.Context) (string, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return "", err
	}
	return richHTML(content, path.Ext(r.Path()))
}

func (r *VirtualFileResolver) Binary(ctx context.Context) (bool, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return false, err
	}
	return highlight.IsBinary([]byte(content)), nil
}

var highlightHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "virtual_fileserver_highlight_req",
	Help: "This measures the time for highlighting requests",
})

func (r *VirtualFileResolver) Highlight(ctx context.Context, args *HighlightArgs) (*HighlightedFileResolver, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return nil, err
	}
	timer := prometheus.NewTimer(highlightHistogram)
	defer timer.ObserveDuration()
	return highlightContent(ctx, args, content, r.Path(), highlight.Metadata{
		// TODO: Use `CanonicalURL` here for where to retrieve the file content, once we have a backend to retrieve such files.
		Revision: fmt.Sprintf("Preview file diff %s", r.stat.Name()),
	})
}
