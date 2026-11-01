package types

// Reasons carried by the order book engine's outcome events. They are fixed values so the
// event content is identical on every node.
const (
	// RefundReasonNoMatch : a fill order found nothing at its price, the whole amount is returned
	RefundReasonNoMatch = "no_match"
	// RefundReasonUnfilledRemainder : a fill order was partly matched, the rest is returned
	RefundReasonUnfilledRemainder = "unfilled_remainder"
	// RefundReasonBelowMinAmount : a limit order was partly matched and its remainder was not placed
	// on the book, the remainder is returned
	RefundReasonBelowMinAmount = "below_min_amount"
	// RefundReasonUpgrade : a message still queued at the v8.3.0 upgrade was returned to its owner by
	// the module migration instead of being executed
	RefundReasonUpgrade = "upgrade"
)
