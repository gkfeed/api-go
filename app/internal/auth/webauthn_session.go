package auth

import (
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

type pendingCeremony struct {
	session          *webauthn.SessionData
	registrationName string
}

type ceremonyStore struct {
	pendingCeremonies map[string]pendingCeremony
	mu                sync.Mutex
}

func newCeremonyStore() *ceremonyStore {
	return &ceremonyStore{
		pendingCeremonies: make(map[string]pendingCeremony),
	}
}

func (s *ceremonyStore) store(session *webauthn.SessionData, registrationName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpired()
	s.pendingCeremonies[session.Challenge] = pendingCeremony{
		session:          session,
		registrationName: registrationName,
	}
}

func (s *ceremonyStore) pop(challenge string) (pendingCeremony, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpired()
	ceremony, ok := s.pendingCeremonies[challenge]
	if !ok {
		return pendingCeremony{}, false
	}
	delete(s.pendingCeremonies, challenge)
	return ceremony, true
}

// removeExpired must be called while s.mu is held.
func (s *ceremonyStore) removeExpired() {
	now := time.Now()
	for challenge, ceremony := range s.pendingCeremonies {
		if !ceremony.session.Expires.IsZero() && ceremony.session.Expires.Before(now) {
			delete(s.pendingCeremonies, challenge)
		}
	}
}
