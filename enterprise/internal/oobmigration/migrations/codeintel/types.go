package codeintel

type ID string

// DocumentData represents a single document within an index. The data here can answer
// definitions, references, and hover queries if the results are all contained in the
// same document.
type DocumentData struct {
	Ranges             map[ID]RangeData
	HoverResults       map[ID]string // hover text normalized to markdown string
	Monikers           map[ID]MonikerData
	PackageInformation map[ID]PackageInformationData
	Diagnostics        []DiagnosticData
}

// RangeData represents a range vertex within an index. It contains the same relevant
// edge data, which can be subsequently queried in the containing document. The data
// that was reachable via a result set has been collapsed into this object during
// conversion.
type RangeData struct {
	StartLine              int  // 0-indexed, inclusive
	StartCharacter         int  // 0-indexed, inclusive
	EndLine                int  // 0-indexed, inclusive
	EndCharacter           int  // 0-indexed, inclusive
	DefinitionResultID     ID   // possibly empty
	ReferenceResultID      ID   // possibly empty
	ImplementationResultID ID   // possibly empty
	HoverResultID          ID   // possibly empty
	MonikerIDs             []ID // possibly empty
}

// MonikerData represent a unique name (eventually) attached to a range.
type MonikerData struct {
	Kind                 string // local, import, export, implementation
	Scheme               string // name of the package manager type
	Identifier           string // unique identifier
	PackageInformationID ID     // possibly empty
}

// PackageInformationData indicates a globally unique namespace for a moniker.
type PackageInformationData struct {
	// Name of the package manager.
	Manager string

	// Name of the package that contains the moniker.
	Name string

	// Version of the package.
	Version string
}

// DiagnosticData carries diagnostic information attached to a range within its
// containing document.
type DiagnosticData struct {
	Severity       int
	Code           string
	Message        string
	Source         string
	StartLine      int // 0-indexed, inclusive
	StartCharacter int // 0-indexed, inclusive
	EndLine        int // 0-indexed, inclusive
	EndCharacter   int // 0-indexed, inclusive
}

// ResultChunkData represents a row of the resultChunk table. Each row is a subset
// of definition and reference result data in the index. Results are inserted into
// chunks based on the hash of their identifier, thus every chunk has a roughly
// proportional amount of data.
type ResultChunkData struct {
	// DocumentPaths is a mapping from document identifiers to their paths. This
	// must be used to convert a document identifier in DocumentIDRangeIDs into
	// a key that can be used to fetch document data.
	DocumentPaths map[ID]string

	// DocumentIDRangeIDs is a mapping from a definition or result reference
	// identifier to the set of ranges that compose that result set. Each range
	// is paired with the identifier of the document in which it can found.
	DocumentIDRangeIDs map[ID][]DocumentIDRangeID
}

// DocumentIDRangeID is a pair of document and range identifiers.
type DocumentIDRangeID struct {
	// The identifier of the document to which the range belongs. This id is only
	// relevant within the containing result chunk.
	DocumentID ID

	// The identifier of the range.
	RangeID ID
}

// Location represents a range within a particular document relative to its
// containing bundle.
type LocationData struct {
	URI            string
	StartLine      int
	StartCharacter int
	EndLine        int
	EndCharacter   int
}
