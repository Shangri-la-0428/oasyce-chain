package params

const (
	// BondDenom is the staking token denomination.
	// 1 OAS = 10^6 uoas (micro-OAS).
	BondDenom = "uoas"

	// AccountAddressPrefix is the Bech32 prefix for account addresses.
	AccountAddressPrefix = "oasyce"

	// Name is the application name.
	Name = "oasyce"

	// DisplayDenom is the human-readable denomination.
	DisplayDenom = "OAS"

	// OASExponent is the exponent for converting uoas to OAS.
	OASExponent = 6
)

// Bech32 prefixes derived from the account address prefix.
var (
	AccountPubKeyPrefix    = AccountAddressPrefix + "pub"
	ValidatorAddressPrefix = AccountAddressPrefix + "valoper"
	ValidatorPubKeyPrefix  = AccountAddressPrefix + "valoperpub"
	ConsNodeAddressPrefix  = AccountAddressPrefix + "valcons"
	ConsNodePubKeyPrefix   = AccountAddressPrefix + "valconspub"
)
