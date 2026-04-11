package main

import (
	"encoding/binary"
	"fmt"
	"sort"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sigilkeeper "github.com/oasyce/chain/x/sigil/keeper"
	sigiltypes "github.com/oasyce/chain/x/sigil/types"
)

type bucketReport struct {
	Bucket  string `json:"bucket"`
	Height  int64  `json:"height"`
	SigilID string `json:"sigil_id"`
}

type sigilAudit struct {
	SigilID              string           `json:"sigil_id"`
	Status               string           `json:"status"`
	LastActiveHeight     int64            `json:"last_active_height"`
	MaxPulseHeight       int64            `json:"max_pulse_height"`
	DimensionPulses      map[string]int64 `json:"dimension_pulses,omitempty"`
	ActiveBucketHeights  []int64          `json:"active_bucket_heights,omitempty"`
	DormantBucketHeights []int64          `json:"dormant_bucket_heights,omitempty"`
}

type auditReport struct {
	BlockHeight        int64          `json:"block_height"`
	ModuleVersion      uint64         `json:"module_version"`
	ActiveCount        uint64         `json:"active_count"`
	ActiveStatusCount  int            `json:"active_status_count"`
	DormantStatusCount int            `json:"dormant_status_count"`
	DissolvedCount     int            `json:"dissolved_count"`
	Sigils             []sigilAudit   `json:"sigils"`
	ActiveBucket       []bucketReport `json:"active_bucket"`
	DormantBucket      []bucketReport `json:"dormant_bucket"`
	OrphanIndexEntries []bucketReport `json:"orphan_index_entries,omitempty"`
	InvariantErrors    []string       `json:"invariant_errors,omitempty"`
}

type replayReport struct {
	SourceHome  string      `json:"source_home"`
	WorkingHome string      `json:"working_home"`
	Before      auditReport `json:"before"`
	After       auditReport `json:"after"`
	Status      string      `json:"status"`
}

type bucketKind string

const (
	activeBucketKind  bucketKind = "0x05"
	dormantBucketKind bucketKind = "0x09"
)

func collectAudit(ctx sdk.Context, k sigilkeeper.Keeper, storeKey storetypes.StoreKey, moduleVersion uint64) auditReport {
	report := auditReport{
		BlockHeight:   ctx.BlockHeight(),
		ModuleVersion: moduleVersion,
		ActiveCount:   k.GetActiveCount(ctx),
	}

	store := ctx.KVStore(storeKey)
	activeEntries := collectBucketEntries(store, sigiltypes.LivenessIndexPrefix, activeBucketKind)
	dormantEntries := collectBucketEntries(store, sigiltypes.DormantLivenessIndexPrefix, dormantBucketKind)
	report.ActiveBucket = activeEntries
	report.DormantBucket = dormantEntries

	activeBySigil := map[string][]int64{}
	dormantBySigil := map[string][]int64{}
	for _, entry := range activeEntries {
		activeBySigil[entry.SigilID] = append(activeBySigil[entry.SigilID], entry.Height)
	}
	for _, entry := range dormantEntries {
		dormantBySigil[entry.SigilID] = append(dormantBySigil[entry.SigilID], entry.Height)
	}

	sigilMap := map[string]sigilAudit{}
	k.IterateAllSigils(ctx, func(s sigiltypes.Sigil) bool {
		status := sigiltypes.SigilStatus(s.Status)
		switch status {
		case sigiltypes.SigilStatusActive:
			report.ActiveStatusCount++
		case sigiltypes.SigilStatusDormant:
			report.DormantStatusCount++
		case sigiltypes.SigilStatusDissolved:
			report.DissolvedCount++
		}

		snapshot := sigilAudit{
			SigilID:              s.SigilId,
			Status:               status.String(),
			LastActiveHeight:     s.LastActiveHeight,
			MaxPulseHeight:       sigilkeeper.MaxPulseHeight(s),
			DimensionPulses:      clonePulseMap(s.DimensionPulses),
			ActiveBucketHeights:  append([]int64(nil), activeBySigil[s.SigilId]...),
			DormantBucketHeights: append([]int64(nil), dormantBySigil[s.SigilId]...),
		}
		sort.Slice(snapshot.ActiveBucketHeights, func(i, j int) bool { return snapshot.ActiveBucketHeights[i] < snapshot.ActiveBucketHeights[j] })
		sort.Slice(snapshot.DormantBucketHeights, func(i, j int) bool { return snapshot.DormantBucketHeights[i] < snapshot.DormantBucketHeights[j] })
		sigilMap[s.SigilId] = snapshot
		return false
	})

	report.OrphanIndexEntries = findOrphanEntries(activeEntries, dormantEntries, sigilMap)
	report.InvariantErrors = append(report.InvariantErrors, validateAudit(sigilless(report.OrphanIndexEntries), sigilMap)...)
	if report.ActiveCount != uint64(report.ActiveStatusCount) {
		report.InvariantErrors = append(report.InvariantErrors,
			fmt.Sprintf("active_count mismatch: counter=%d status_index=%d", report.ActiveCount, report.ActiveStatusCount),
		)
	}

	report.Sigils = make([]sigilAudit, 0, len(sigilMap))
	for _, snapshot := range sigilMap {
		report.Sigils = append(report.Sigils, snapshot)
	}
	sort.Slice(report.Sigils, func(i, j int) bool { return report.Sigils[i].SigilID < report.Sigils[j].SigilID })

	return report
}

func collectBucketEntries(store storetypes.KVStore, prefix []byte, kind bucketKind) []bucketReport {
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var entries []bucketReport
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < len(prefix)+8 {
			continue
		}
		height := int64(binary.BigEndian.Uint64(key[len(prefix) : len(prefix)+8]))
		sigilID := string(key[len(prefix)+8:])
		if sigilID == "" {
			sigilID = string(iter.Value())
		}
		entries = append(entries, bucketReport{
			Bucket:  string(kind),
			Height:  height,
			SigilID: sigilID,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Height == entries[j].Height {
			return entries[i].SigilID < entries[j].SigilID
		}
		return entries[i].Height < entries[j].Height
	})
	return entries
}

func findOrphanEntries(activeEntries, dormantEntries []bucketReport, sigilMap map[string]sigilAudit) []bucketReport {
	var orphans []bucketReport
	for _, entry := range append(append([]bucketReport(nil), activeEntries...), dormantEntries...) {
		if _, ok := sigilMap[entry.SigilID]; !ok {
			orphans = append(orphans, entry)
		}
	}
	return orphans
}

func sigilless(entries []bucketReport) bool {
	return len(entries) == 0
}

func validateAudit(noOrphans bool, sigilMap map[string]sigilAudit) []string {
	var errs []string
	if !noOrphans {
		errs = append(errs, "orphan liveness index entries present")
	}

	for _, snapshot := range sigilMap {
		activeCount := len(snapshot.ActiveBucketHeights)
		dormantCount := len(snapshot.DormantBucketHeights)

		if activeCount > 0 && dormantCount > 0 {
			errs = append(errs, fmt.Sprintf("sigil %s appears in both active and dormant liveness buckets", snapshot.SigilID))
		}

		switch snapshot.Status {
		case sigiltypes.SigilStatusActive.String():
			if activeCount != 1 {
				errs = append(errs, fmt.Sprintf("active sigil %s must appear exactly once in 0x05 bucket, got %d", snapshot.SigilID, activeCount))
				continue
			}
			if dormantCount != 0 {
				errs = append(errs, fmt.Sprintf("active sigil %s must not appear in 0x09 bucket", snapshot.SigilID))
			}
			if snapshot.ActiveBucketHeights[0] != snapshot.MaxPulseHeight {
				errs = append(errs, fmt.Sprintf("active sigil %s indexed at %d, want MaxPulseHeight=%d", snapshot.SigilID, snapshot.ActiveBucketHeights[0], snapshot.MaxPulseHeight))
			}
		case sigiltypes.SigilStatusDormant.String():
			if dormantCount != 1 {
				errs = append(errs, fmt.Sprintf("dormant sigil %s must appear exactly once in 0x09 bucket, got %d", snapshot.SigilID, dormantCount))
				continue
			}
			if activeCount != 0 {
				errs = append(errs, fmt.Sprintf("dormant sigil %s must not appear in 0x05 bucket", snapshot.SigilID))
			}
			if snapshot.DormantBucketHeights[0] != snapshot.MaxPulseHeight {
				errs = append(errs, fmt.Sprintf("dormant sigil %s indexed at %d, want frozen MaxPulseHeight=%d", snapshot.SigilID, snapshot.DormantBucketHeights[0], snapshot.MaxPulseHeight))
			}
		case sigiltypes.SigilStatusDissolved.String():
			if activeCount != 0 || dormantCount != 0 {
				errs = append(errs, fmt.Sprintf("dissolved sigil %s must not appear in liveness buckets", snapshot.SigilID))
			}
		default:
			errs = append(errs, fmt.Sprintf("sigil %s has unknown status %s", snapshot.SigilID, snapshot.Status))
		}
	}

	sort.Strings(errs)
	return errs
}

func clonePulseMap(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
