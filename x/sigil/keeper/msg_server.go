package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/sigil/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// Genesis creates a new Sigil.
func (m msgServer) Genesis(goCtx context.Context, msg *types.MsgGenesis) (*types.MsgGenesisResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sigilID := types.DeriveSigilID(msg.PublicKey)

	// Check for duplicate.
	if _, found := m.Keeper.GetSigil(ctx, sigilID); found {
		return nil, types.ErrSigilExists.Wrapf("sigil %s already exists", sigilID)
	}

	// Validate lineage references.
	for _, parentID := range msg.Lineage {
		parent, found := m.Keeper.GetSigil(ctx, parentID)
		if !found {
			return nil, types.ErrSigilNotFound.Wrapf("lineage parent %s not found", parentID)
		}
		if types.SigilStatus(parent.Status) == types.SigilStatusDissolved {
			return nil, types.ErrSigilDissolved.Wrapf("lineage parent %s is dissolved", parentID)
		}
	}

	sigil := types.Sigil{
		SigilId:          sigilID,
		Creator:          msg.Signer,
		PublicKey:        msg.PublicKey,
		Status:           types.SigilStatusActive,
		CreationHeight:   ctx.BlockHeight(),
		LastActiveHeight: ctx.BlockHeight(),
		StateRoot:        msg.StateRoot,
		Lineage:          msg.Lineage,
		Metadata:         msg.Metadata,
	}

	if err := m.Keeper.SetSigil(ctx, sigil); err != nil {
		return nil, err
	}

	// Record lineage edges.
	for _, parentID := range msg.Lineage {
		m.Keeper.SetLineage(ctx, parentID, sigilID)
	}

	m.Keeper.IncrementActiveCount(ctx)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"sigil_genesis",
		sdk.NewAttribute("sigil_id", sigilID),
		sdk.NewAttribute("creator", msg.Signer),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return &types.MsgGenesisResponse{SigilId: sigilID}, nil
}

// Dissolve permanently retires a Sigil.
func (m msgServer) Dissolve(goCtx context.Context, msg *types.MsgDissolve) (*types.MsgDissolveResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sigil, found := m.Keeper.GetSigil(ctx, msg.SigilId)
	if !found {
		return nil, types.ErrSigilNotFound.Wrapf("sigil %s not found", msg.SigilId)
	}
	if types.SigilStatus(sigil.Status) == types.SigilStatusDissolved {
		return nil, types.ErrSigilDissolved.Wrapf("sigil %s already dissolved", msg.SigilId)
	}
	if sigil.Creator != msg.Signer {
		return nil, types.ErrNotSigilOwner.Wrapf("signer %s is not creator of sigil %s", msg.Signer, msg.SigilId)
	}

	oldStatus := types.SigilStatus(sigil.Status)

	// Remove from old indexes.
	m.Keeper.DeleteSigilFromStatusIndex(ctx, oldStatus, sigil.SigilId)
	if oldStatus == types.SigilStatusActive {
		m.Keeper.DeleteSigilFromLivenessIndex(ctx, sigil.LastActiveHeight, sigil.SigilId)
		m.Keeper.DecrementActiveCount(ctx)
	}

	sigil.Status = types.SigilStatusDissolved
	if err := m.Keeper.SetSigil(ctx, sigil); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"sigil_dissolve",
		sdk.NewAttribute("sigil_id", msg.SigilId),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return &types.MsgDissolveResponse{}, nil
}

// Bond creates a bond between two Sigils.
func (m msgServer) Bond(goCtx context.Context, msg *types.MsgBond) (*types.MsgBondResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate both sigils exist and are active.
	for _, sigilID := range []string{msg.SigilA, msg.SigilB} {
		s, found := m.Keeper.GetSigil(ctx, sigilID)
		if !found {
			return nil, types.ErrSigilNotFound.Wrapf("sigil %s not found", sigilID)
		}
		if types.SigilStatus(s.Status) == types.SigilStatusDissolved {
			return nil, types.ErrSigilDissolved.Wrapf("sigil %s is dissolved", sigilID)
		}
	}

	bondID := types.DeriveBondID(msg.SigilA, msg.SigilB)

	if _, found := m.Keeper.GetBond(ctx, bondID); found {
		return nil, types.ErrBondExists.Wrapf("bond %s already exists", bondID)
	}

	bond := types.Bond{
		BondId:         bondID,
		SigilA:         msg.SigilA,
		SigilB:         msg.SigilB,
		TermsHash:      msg.TermsHash,
		CreationHeight: ctx.BlockHeight(),
		Scope:          msg.Scope,
	}

	if err := m.Keeper.SetBond(ctx, bond); err != nil {
		return nil, err
	}

	// Touch both sigils (update liveness).
	m.touchSigil(ctx, msg.SigilA)
	m.touchSigil(ctx, msg.SigilB)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"sigil_bond",
		sdk.NewAttribute("bond_id", bondID),
		sdk.NewAttribute("sigil_a", msg.SigilA),
		sdk.NewAttribute("sigil_b", msg.SigilB),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return &types.MsgBondResponse{BondId: bondID}, nil
}

// Unbond removes a bond.
func (m msgServer) Unbond(goCtx context.Context, msg *types.MsgUnbond) (*types.MsgUnbondResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	bond, found := m.Keeper.GetBond(ctx, msg.BondId)
	if !found {
		return nil, types.ErrBondNotFound.Wrapf("bond %s not found", msg.BondId)
	}

	// Signer must be creator of one of the bonded sigils.
	sigA, _ := m.Keeper.GetSigil(ctx, bond.SigilA)
	sigB, _ := m.Keeper.GetSigil(ctx, bond.SigilB)
	if sigA.Creator != msg.Signer && sigB.Creator != msg.Signer {
		return nil, types.ErrNotSigilOwner.Wrapf("signer %s is not creator of either bonded sigil", msg.Signer)
	}

	m.Keeper.DeleteBond(ctx, bond)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"sigil_unbond",
		sdk.NewAttribute("bond_id", msg.BondId),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return &types.MsgUnbondResponse{}, nil
}

// Fork creates a new Sigil from an existing parent.
func (m msgServer) Fork(goCtx context.Context, msg *types.MsgFork) (*types.MsgForkResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	parent, found := m.Keeper.GetSigil(ctx, msg.ParentSigilId)
	if !found {
		return nil, types.ErrSigilNotFound.Wrapf("parent sigil %s not found", msg.ParentSigilId)
	}
	if types.SigilStatus(parent.Status) == types.SigilStatusDissolved {
		return nil, types.ErrSigilDissolved.Wrapf("parent sigil %s is dissolved", msg.ParentSigilId)
	}
	if parent.Creator != msg.Signer {
		return nil, types.ErrNotSigilOwner.Wrapf("signer %s is not creator of parent sigil %s", msg.Signer, msg.ParentSigilId)
	}

	childID := types.DeriveSigilID(msg.PublicKey)
	if _, found := m.Keeper.GetSigil(ctx, childID); found {
		return nil, types.ErrSigilExists.Wrapf("child sigil %s already exists", childID)
	}

	// Child inherits parent's state root (Lamarckian: full state inheritance).
	stateRoot := parent.StateRoot

	child := types.Sigil{
		SigilId:          childID,
		Creator:          msg.Signer,
		PublicKey:        msg.PublicKey,
		Status:           types.SigilStatusActive,
		CreationHeight:   ctx.BlockHeight(),
		LastActiveHeight: ctx.BlockHeight(),
		StateRoot:        stateRoot,
		Lineage:          []string{msg.ParentSigilId},
		Metadata:         msg.Metadata,
	}

	if err := m.Keeper.SetSigil(ctx, child); err != nil {
		return nil, err
	}

	m.Keeper.SetLineage(ctx, msg.ParentSigilId, childID)
	m.Keeper.IncrementActiveCount(ctx)
	m.touchSigil(ctx, msg.ParentSigilId)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"sigil_fork",
		sdk.NewAttribute("parent_sigil_id", msg.ParentSigilId),
		sdk.NewAttribute("child_sigil_id", childID),
		sdk.NewAttribute("fork_mode", fmt.Sprintf("%d", msg.ForkMode)),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return &types.MsgForkResponse{ChildSigilId: childID}, nil
}

// Merge combines two Sigils into one.
func (m msgServer) Merge(goCtx context.Context, msg *types.MsgMerge) (*types.MsgMergeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sigA, found := m.Keeper.GetSigil(ctx, msg.SigilA)
	if !found {
		return nil, types.ErrSigilNotFound.Wrapf("sigil %s not found", msg.SigilA)
	}
	sigB, found := m.Keeper.GetSigil(ctx, msg.SigilB)
	if !found {
		return nil, types.ErrSigilNotFound.Wrapf("sigil %s not found", msg.SigilB)
	}

	for _, s := range []types.Sigil{sigA, sigB} {
		if types.SigilStatus(s.Status) == types.SigilStatusDissolved {
			return nil, types.ErrSigilDissolved.Wrapf("sigil %s is dissolved", s.SigilId)
		}
	}

	// Signer must be creator of at least one sigil.
	if sigA.Creator != msg.Signer && sigB.Creator != msg.Signer {
		return nil, types.ErrNotSigilOwner.Wrapf("signer %s is not creator of either sigil", msg.Signer)
	}

	// Determine survivor based on merge mode.
	var survivorID string
	if types.MergeMode(msg.MergeMode) == types.MergeModeAbsorption {
		// A absorbs B.
		survivorID = msg.SigilA
		m.dissolveSigil(ctx, &sigB)
	} else {
		// Symmetric: both dissolve, new sigil emerges.
		// For now, A is the survivor (simplification).
		survivorID = msg.SigilA
		m.dissolveSigil(ctx, &sigB)
	}

	// Touch survivor.
	m.touchSigil(ctx, survivorID)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"sigil_merge",
		sdk.NewAttribute("sigil_a", msg.SigilA),
		sdk.NewAttribute("sigil_b", msg.SigilB),
		sdk.NewAttribute("merged_sigil_id", survivorID),
		sdk.NewAttribute("merge_mode", fmt.Sprintf("%d", msg.MergeMode)),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return &types.MsgMergeResponse{MergedSigilId: survivorID}, nil
}

// UpdateParams governance-gated parameter update.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.Keeper.Authority() {
		return nil, types.ErrInvalidAddress.Wrapf("unauthorized: expected %s, got %s", m.Keeper.Authority(), msg.Authority)
	}

	if err := m.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// touchSigil updates a sigil's LastActiveHeight.
func (m msgServer) touchSigil(ctx sdk.Context, sigilID string) {
	s, found := m.Keeper.GetSigil(ctx, sigilID)
	if !found {
		return
	}
	if types.SigilStatus(s.Status) != types.SigilStatusActive {
		return
	}

	// Remove old liveness index entry.
	m.Keeper.DeleteSigilFromLivenessIndex(ctx, s.LastActiveHeight, s.SigilId)

	s.LastActiveHeight = ctx.BlockHeight()
	_ = m.Keeper.SetSigil(ctx, s) // re-inserts at new height
}

// dissolveSigil marks a sigil as dissolved and updates indexes.
func (m msgServer) dissolveSigil(ctx sdk.Context, s *types.Sigil) {
	oldStatus := types.SigilStatus(s.Status)
	m.Keeper.DeleteSigilFromStatusIndex(ctx, oldStatus, s.SigilId)
	if oldStatus == types.SigilStatusActive {
		m.Keeper.DeleteSigilFromLivenessIndex(ctx, s.LastActiveHeight, s.SigilId)
		m.Keeper.DecrementActiveCount(ctx)
	}
	s.Status = types.SigilStatusDissolved
	_ = m.Keeper.SetSigil(ctx, *s)
}
