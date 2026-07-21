package auth

import (
	"encoding/binary"

	"gkfeed/api/internal/models"

	"github.com/go-webauthn/webauthn/webauthn"
)

// webAuthnUserAdapter presents an application user in the form required by
// the go-webauthn library.
type webAuthnUserAdapter struct {
	models.User
	credentials []webauthn.Credential
}

var _ webauthn.User = webAuthnUserAdapter{}

func (u webAuthnUserAdapter) WebAuthnID() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(u.ID))
	return buf
}

func (u webAuthnUserAdapter) WebAuthnName() string {
	return u.Name
}

func (u webAuthnUserAdapter) WebAuthnDisplayName() string {
	return u.Name
}

func (u webAuthnUserAdapter) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}
