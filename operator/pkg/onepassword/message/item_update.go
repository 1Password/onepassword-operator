package message

import "encoding/json"

// TypeItemUpdate and others are sync message types
const (
	TypeItemUpdate = "item.update"
)

// ItemUpdateEvent is the data for a sync status message
type ItemUpdateEvent struct {
	VaultId string `json:"vaultId"`
	ItemId  string `json:"itemId"`
	Version string `json:"version"`
}

// Type returns a the syns status data type
func (s *ItemUpdateEvent) Type() string {
	return TypeItemUpdate
}

// Bytes returns Bytes
func (s *ItemUpdateEvent) Bytes() []byte {
	bytes, _ := json.Marshal(s)

	return bytes
}
