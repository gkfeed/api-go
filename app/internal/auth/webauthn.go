package auth

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"gkfeed/api/internal/config"
	"gkfeed/api/internal/db"
	"gkfeed/api/internal/models"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type webAuthnUser struct {
	models.User
	credentials []webauthn.Credential
}

func (u webAuthnUser) WebAuthnID() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(u.ID))
	return buf
}

func (u webAuthnUser) WebAuthnName() string {
	return u.Name
}

func (u webAuthnUser) WebAuthnDisplayName() string {
	return u.Name
}

func (u webAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

type Service struct {
	webAuthn          *webauthn.WebAuthn
	sessions          map[string]*webauthn.SessionData
	registrationNames map[string]string
	mu                sync.RWMutex
}

func NewService(cfg config.Config) (*Service, error) {
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.WebAuthnRPDisplay,
		RPID:          cfg.WebAuthnRPID,
		RPOrigins:     []string{cfg.WebAuthnRPOrigin},
	})
	if err != nil {
		return nil, fmt.Errorf("create webauthn: %w", err)
	}

	svc := &Service{
		webAuthn:          wa,
		sessions:          make(map[string]*webauthn.SessionData),
		registrationNames: make(map[string]string),
	}
	go svc.cleanupSessions()
	return svc, nil
}

func (s *Service) SetRegistrationName(challenge, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registrationNames[string(challenge)] = name
}

func (s *Service) PopRegistrationName(challenge string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := s.registrationNames[string(challenge)]
	delete(s.registrationNames, string(challenge))
	return name
}

func (s *Service) BeginRegistration(user models.User) (*protocol.CredentialCreation, error) {
	credentials, err := db.GetWebAuthnCredentialsByUserID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}

	wu := webAuthnUser{User: user, credentials: credentials}
	creation, sessionData, err := s.webAuthn.BeginRegistration(
		wu,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationPreferred,
		}),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	)
	if err != nil {
		return nil, fmt.Errorf("begin registration: %w", err)
	}

	s.storeSession(sessionData)
	return creation, nil
}

func (s *Service) FinishRegistration(user models.User, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	credentials, err := db.GetWebAuthnCredentialsByUserID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}

	wu := webAuthnUser{User: user, credentials: credentials}
	sessionData := s.popSession(string(response.Response.CollectedClientData.Challenge))
	if sessionData == nil {
		return nil, fmt.Errorf("session not found")
	}

	credential, err := s.webAuthn.CreateCredential(wu, *sessionData, response)
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}
	return credential, nil
}

func (s *Service) BeginLogin() (*protocol.CredentialAssertion, error) {
	assertion, sessionData, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("begin discoverable login: %w", err)
	}
	s.storeSession(sessionData)
	return assertion, nil
}

func (s *Service) FinishLogin(response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, models.User, error) {
	sessionData := s.popSession(string(response.Response.CollectedClientData.Challenge))
	if sessionData == nil {
		return nil, models.User{}, fmt.Errorf("session not found")
	}

	credentialJSON, userID, _, err := db.GetWebAuthnCredentialRowByID(response.RawID)
	if err != nil {
		return nil, models.User{}, fmt.Errorf("credential not found: %w", err)
	}

	user, err := db.GetUserFromDBByID(userID)
	if err != nil {
		return nil, models.User{}, fmt.Errorf("user not found: %w", err)
	}

	credentials, err := db.GetWebAuthnCredentialsByUserID(user.ID)
	if err != nil {
		return nil, models.User{}, fmt.Errorf("get credentials: %w", err)
	}
	wu := webAuthnUser{User: user, credentials: credentials}

	credential, err := s.webAuthn.ValidateDiscoverableLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			if string(userHandle) != string(wu.WebAuthnID()) {
				return nil, fmt.Errorf("user handle mismatch")
			}
			return wu, nil
		},
		*sessionData,
		response,
	)
	if err != nil {
		return nil, models.User{}, fmt.Errorf("validate login: %w", err)
	}

	if err := db.UpdateWebAuthnCredential(*credential); err != nil {
		return nil, models.User{}, fmt.Errorf("update credential: %w", err)
	}

	_ = credentialJSON

	return credential, user, nil
}

func (s *Service) RegisterCredential(user models.User, credential *webauthn.Credential, name string) error {
	return db.AddWebAuthnCredential(user.ID, *credential, name)
}

func (s *Service) storeSession(data *webauthn.SessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[string(data.Challenge)] = data
}

func (s *Service) popSession(challenge string) *webauthn.SessionData {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.sessions[string(challenge)]
	if !ok {
		return nil
	}
	delete(s.sessions, string(challenge))
	return data
}

func (s *Service) cleanupSessions() {
	for range time.Tick(5 * time.Minute) {
		s.mu.Lock()
		for k := range s.sessions {
			delete(s.sessions, k)
		}
		s.mu.Unlock()
	}
}
