package identity

import "testing"

func TestGetIdentity(t *testing.T) {
	identity := GetIdentity()
	t.Log(identity)
}
