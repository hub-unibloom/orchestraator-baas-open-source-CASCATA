package privacy

// MaskingTier defines how data is presented to various roles.
type MaskingTier string

const (
	Normal       MaskingTier = "normal"
	Hide         MaskingTier = "hide"
	Mask         MaskingTier = "mask"
	SemiMask     MaskingTier = "semi-mask"
	Blur         MaskingTier = "blur"
	HybridEnc    MaskingTier = "hybrid_encrypt"
	HyperEnc     MaskingTier = "hyper_encrypt"
)

// LockLevel defines write-access restrictions on columns.
type LockLevel string

const (
	Unlocked     LockLevel = "unlocked"
	Immutable    LockLevel = "immutable"
	SystemPut    LockLevel = "system_put"
	InsertOnly   LockLevel = "insert_only"
	OtpProtected LockLevel = "otp_protected"
)

// ProjectPrivacyConfig represents the metadata structure for masking/locks.
type ProjectPrivacyConfig struct {
	MaskedColumns map[string]map[string]MaskingTier `json:"masked_columns"`
	LockedColumns map[string]map[string]LockLevel   `json:"locked_columns"`
}
