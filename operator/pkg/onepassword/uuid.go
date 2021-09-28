package onepassword

// UUIDLength defines the required length of UUIDs
const UUIDLength = 26

// IsValidClientUUID returns true if the given client uuid is valid.
func IsValidClientUUID(uuid string) bool {
	if len(uuid) != UUIDLength {
		return false
	}

	for _, c := range uuid {
		valid := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		if !valid {
			return false
		}
	}

	return true
}
