package types

const (
	// ModuleName is the name of the reputation module.
	ModuleName = "reputation"

	// StoreKey is the store key string for reputation.
	StoreKey = ModuleName

	// RouterKey is the message route for reputation.
	RouterKey = ModuleName

	// QuerierRoute is the querier route for reputation.
	QuerierRoute = ModuleName
)

// Key prefixes for the reputation store.
var (
	ScoreKeyPrefix        = []byte{0x01}
	FeedbackKeyPrefix     = []byte{0x02}
	FeedbackByToPrefix    = []byte{0x03}
	ParamsKey             = []byte{0x04}
	FeedbackCounterKey    = []byte{0x05}
	ReportKeyPrefix       = []byte{0x06}
	ReportCounterKey      = []byte{0x07}
	FeedbackByInvPrefix   = []byte{0x08}
	CooldownKeyPrefix     = []byte{0x09}
)

// ScoreKey returns the store key for a reputation score by address.
func ScoreKey(address string) []byte {
	return append(ScoreKeyPrefix, []byte(address)...)
}

// FeedbackKey returns the store key for a feedback by ID.
func FeedbackKey(feedbackID string) []byte {
	return append(FeedbackKeyPrefix, []byte(feedbackID)...)
}

// FeedbackByToKey returns the store key for indexing feedback by target address.
func FeedbackByToKey(to, feedbackID string) []byte {
	key := append(FeedbackByToPrefix, []byte(to)...)
	key = append(key, '/')
	key = append(key, []byte(feedbackID)...)
	return key
}

// FeedbackByToIteratorPrefix returns the prefix for iterating feedback by target.
func FeedbackByToIteratorPrefix(to string) []byte {
	key := append(FeedbackByToPrefix, []byte(to)...)
	key = append(key, '/')
	return key
}

// FeedbackByInvKey returns the store key for indexing feedback by invocation+from.
func FeedbackByInvKey(invocationID, from string) []byte {
	key := append(FeedbackByInvPrefix, []byte(invocationID)...)
	key = append(key, '/')
	key = append(key, []byte(from)...)
	return key
}

// CooldownKey returns the store key for tracking cooldown by from+to.
func CooldownKey(from, to string) []byte {
	key := append(CooldownKeyPrefix, []byte(from)...)
	key = append(key, '/')
	key = append(key, []byte(to)...)
	return key
}

// ReportKey returns the store key for a misbehavior report by ID.
func ReportKey(reportID string) []byte {
	return append(ReportKeyPrefix, []byte(reportID)...)
}
