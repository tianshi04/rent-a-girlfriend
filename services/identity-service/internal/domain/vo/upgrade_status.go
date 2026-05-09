package vo

// UpgradeStatus represents the state of a companion upgrade request.
type UpgradeStatus string

const (
	UpgradeStatusPending  UpgradeStatus = "PENDING"
	UpgradeStatusApproved UpgradeStatus = "APPROVED"
	UpgradeStatusRejected UpgradeStatus = "REJECTED"
)

// CanApprove returns true if the request can be approved.
func (s UpgradeStatus) CanApprove() bool {
	return s == UpgradeStatusPending
}

// CanReject returns true if the request can be rejected.
func (s UpgradeStatus) CanReject() bool {
	return s == UpgradeStatusPending
}

// IsPending returns true if the request is still awaiting review.
func (s UpgradeStatus) IsPending() bool {
	return s == UpgradeStatusPending
}

// String returns the string representation.
func (s UpgradeStatus) String() string {
	return string(s)
}

// ParseUpgradeStatus converts a string to UpgradeStatus.
func ParseUpgradeStatus(s string) (UpgradeStatus, error) {
	switch s {
	case "PENDING":
		return UpgradeStatusPending, nil
	case "APPROVED":
		return UpgradeStatusApproved, nil
	case "REJECTED":
		return UpgradeStatusRejected, nil
	default:
		return "", nil // or error
	}
}
