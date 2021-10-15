package message

import "encoding/json"

// TypeItemUpdate and others are sync message types
const (
	TypeItemUpdate = "item.update"
)

// ItemUpdateEvent is the data for a sync status message
type ItemUpdateEvent struct {
	VaultUUID   string `json:"vault_uuid"`
	ItemUUID    string `json:"item_uuid"`
	ItemVersion string `json:"item_version"`
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
