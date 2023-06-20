package attestation

import (
	"github.com/vulcanize/lotus-index-attestation/pkg/types"
)

var _ types.API = (*API)(nil)

// API is kind of an unnecessary abstraction since this only wraps a single backing struct, but it will make it easier
// to extend the API in the future (add and use new backing components)
type API struct {
	backend types.ChecksumRepository
}

// NewAPI returns a new API object
func NewAPI(repo types.ChecksumRepository) *API {
	return &API{backend: repo}
}

// ChecksumExists returns true if the given checksum is published in the backing checksum repository
func (A API) ChecksumExists(hash string, res *bool) error {
	exists, err := A.backend.ChecksumExists(hash)
	if err != nil {
		return err
	}
	*res = exists
	return nil
}

// GetChecksum returns the checksum for the given start and stop values
func (A API) GetChecksum(rng types.GetChecksumRequest, res *string) error {
	hash, err := A.backend.GetChecksum(rng.Start, rng.Stop)
	if err != nil {
		return err
	}
	*res = hash
	return nil
}
