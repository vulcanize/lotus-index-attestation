package attestation

// Config holds the configuration params for the attestation service
type Config struct {
	// support checksumming
	Checksum bool

	// Directory with the source msgindex.db sqlite file
	SrcDBDir string
	// Directory with/for the checksums.db sqlite file
	RepoDBDir string
	// Chunk range size for checksumming
	ChecksumChunkSize uint
	// Whether to check for gaps in the source msgindex.db while checksumming
	CheckForGaps uint
}
