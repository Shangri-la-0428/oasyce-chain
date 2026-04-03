package types

// Default parameters.
const (
	DefaultDormantThreshold int64 = 100_000   // ~6 days at 5s/block
	DefaultDissolveThreshold int64 = 1_000_000 // ~58 days at 5s/block
	DefaultSubmitWindow     int64 = 100        // ~8 minutes at 5s/block
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		DormantThreshold:  DefaultDormantThreshold,
		DissolveThreshold: DefaultDissolveThreshold,
		SubmitWindow:      DefaultSubmitWindow,
	}
}

// Validate validates the params.
func (p Params) Validate() error {
	if p.DormantThreshold <= 0 {
		return ErrInvalidSigilID.Wrap("dormant_threshold must be positive")
	}
	if p.DissolveThreshold <= 0 {
		return ErrInvalidSigilID.Wrap("dissolve_threshold must be positive")
	}
	if p.DissolveThreshold <= p.DormantThreshold {
		return ErrInvalidSigilID.Wrap("dissolve_threshold must exceed dormant_threshold")
	}
	if p.SubmitWindow <= 0 {
		return ErrInvalidSigilID.Wrap("submit_window must be positive")
	}
	return nil
}
