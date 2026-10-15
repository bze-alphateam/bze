package types

// Reasons carried by the order book engine's outcome events. They are fixed values so the
// event content is identical on every node.
const (
	// RefundReasonNoMatch : a fill message found nothing at its price, the whole amount is returned
	RefundReasonNoMatch = "no_match"
	// RefundReasonUnfilledRemainder : a fill message was partly matched, the rest is returned
	RefundReasonUnfilledRemainder = "unfilled_remainder"
	// RefundReasonBelowMinAmount : a create message was partly matched and its remainder was not placed
	// as an order, the remainder is returned
	RefundReasonBelowMinAmount = "below_min_amount"

	// CancelSkippedReasonOrderNotFound : the order to cancel no longer exists
	CancelSkippedReasonOrderNotFound = "order_not_found"

	// MessageFailedReasonProcessingError : processing the message failed, it stays queued
	MessageFailedReasonProcessingError = "processing_error"
)
