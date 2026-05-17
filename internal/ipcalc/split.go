package ipcalc

import (
	"errors"
	"net/netip"
)

var (
	ErrSplitTooMany   = errors.New("requested more subnets than the parent can hold")
	ErrVLSMDoesNotFit = errors.New("requested host allocations do not fit in the parent network")
	ErrInvalidParent  = errors.New("parent prefix is not a valid IPv4 network")
)

// SplitEqual divides parent into n equally-sized subnets. The chosen prefix
// length is the smallest that yields >= n subnets.
//
// TODO(v1): implement after Calculate is in.
func SplitEqual(parent netip.Prefix, n int) ([]Result, error) {
	return nil, errors.New("SplitEqual: not implemented")
}

// SplitVLSM packs the requested host-count allocations into parent using
// variable-length subnet masking (largest-first). Returns the allocated
// subnets in caller-supplied order so labels line up with inputs.
//
// TODO(v1): implement after SplitEqual is in.
func SplitVLSM(parent netip.Prefix, hosts []int) ([]Result, error) {
	return nil, errors.New("SplitVLSM: not implemented")
}
