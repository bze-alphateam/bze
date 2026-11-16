package v5

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// Store prefixes of the order book queue that existed up to consensus version 4.
const (
	QueueMessagePrefix        = "Qm/value/"
	QueueMessageCounterPrefix = "Qm/counter/"
	PendingCancelPrefix       = "Qm/pc/"

	// messageTypeCancel is the queue message type of a cancel request. Every other type (buy, sell,
	// fill_buy, fill_sell) had its order coins escrowed when it was queued.
	messageTypeCancel = "cancel"
)

// queueMessage holds the fields of the removed bze.tradebin.QueueMessage proto message the
// migration needs. The type itself is gone from the protos, so stored values are decoded by hand.
type queueMessage struct {
	MessageId   string // field 1
	MarketId    string // field 2
	MessageType string // field 3
	Amount      string // field 4
	Price       string // field 5
	OrderType   string // field 8
	Owner       string // field 9
}

// decodeQueueMessage decodes the protobuf encoding of a stored bze.tradebin.QueueMessage.
// Fields it does not need (created_at, order_id) and unknown fields are skipped.
func decodeQueueMessage(b []byte) (queueMessage, error) {
	var qm queueMessage
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return qm, fmt.Errorf("invalid queue message tag: %w", protowire.ParseError(n))
		}
		b = b[n:]

		if typ != protowire.BytesType {
			n = protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return qm, fmt.Errorf("invalid queue message field %d: %w", num, protowire.ParseError(n))
			}
			b = b[n:]
			continue
		}

		v, n := protowire.ConsumeBytes(b)
		if n < 0 {
			return qm, fmt.Errorf("invalid queue message field %d: %w", num, protowire.ParseError(n))
		}
		b = b[n:]

		switch num {
		case 1:
			qm.MessageId = string(v)
		case 2:
			qm.MarketId = string(v)
		case 3:
			qm.MessageType = string(v)
		case 4:
			qm.Amount = string(v)
		case 5:
			qm.Price = string(v)
		case 8:
			qm.OrderType = string(v)
		case 9:
			qm.Owner = string(v)
		}
	}

	return qm, nil
}
