package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// SigilStatus represents the lifecycle status of a Sigil.
type SigilStatus int32

const (
	SigilStatusActive    SigilStatus = 0
	SigilStatusDormant   SigilStatus = 1
	SigilStatusDissolved SigilStatus = 2
)

func (s SigilStatus) String() string {
	switch s {
	case SigilStatusActive:
		return "ACTIVE"
	case SigilStatusDormant:
		return "DORMANT"
	case SigilStatusDissolved:
		return "DISSOLVED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// ForkMode represents the fork strategy.
type ForkMode int32

const (
	ForkModeSymmetric  ForkMode = 0
	ForkModeAsymmetric ForkMode = 1
)

// MergeMode represents the merge strategy.
type MergeMode int32

const (
	MergeModeSymmetric  MergeMode = 0
	MergeModeAbsorption MergeMode = 1
)

// DeriveSigilID deterministically creates a Sigil ID from a public key.
func DeriveSigilID(pubkey []byte) string {
	h := sha256.Sum256(pubkey)
	return "SIG_" + hex.EncodeToString(h[:16])
}

// DeriveBondID deterministically creates a Bond ID from two Sigil IDs.
// Order-independent: Bond(A,B) == Bond(B,A).
func DeriveBondID(sigilA, sigilB string) string {
	// Lexicographic ordering ensures determinism.
	if sigilA > sigilB {
		sigilA, sigilB = sigilB, sigilA
	}
	h := sha256.Sum256([]byte(sigilA + "|" + sigilB))
	return "BOND_" + hex.EncodeToString(h[:16])
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Sigils: []Sigil{},
		Bonds:  []Bond{},
		Params: DefaultParams(),
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seenSigils := make(map[string]bool)
	for _, s := range gs.Sigils {
		if seenSigils[s.SigilId] {
			return ErrDuplicateSigil.Wrapf("duplicate sigil_id in genesis: %s", s.SigilId)
		}
		seenSigils[s.SigilId] = true
		if s.SigilId == "" {
			return ErrInvalidSigilID.Wrap("sigil_id cannot be empty in genesis")
		}
		if s.Creator == "" {
			return ErrInvalidAddress.Wrap("creator cannot be empty in genesis")
		}
	}

	seenBonds := make(map[string]bool)
	for _, b := range gs.Bonds {
		if seenBonds[b.BondId] {
			return ErrBondExists.Wrapf("duplicate bond_id in genesis: %s", b.BondId)
		}
		seenBonds[b.BondId] = true
		if !seenSigils[b.SigilA] {
			return ErrSigilNotFound.Wrapf("bond references unknown sigil_a: %s", b.SigilA)
		}
		if !seenSigils[b.SigilB] {
			return ErrSigilNotFound.Wrapf("bond references unknown sigil_b: %s", b.SigilB)
		}
	}

	return nil
}
