package auth

import (
	"context"
	"fmt"
	"strings"
)

// MFAFactor is an available multi-factor authentication method.
type MFAFactor string

// Supported MFA factors.
const (
	FactorTOTP     MFAFactor = "totp"
	FactorRecovery MFAFactor = "recovery"
)

// Authenticator runs the MFA part of the sign-in flow.
type Authenticator struct {
	Connector *Connector
}

// Prompt collects a verification code from the user.
type Prompt func(factor MFAFactor) (string, error)

// Verify handles MFA verification for the given email. Available factors are
// requested from the API; TOTP is preferred, with recovery codes as a backup.
func (a *Authenticator) Verify(ctx context.Context, email string, prompt Prompt) (*TokenResponse, error) {
	var factors []MFAFactor
	if challenge, err := a.Connector.ChallengeMFA(ctx, email); err == nil {
		for _, raw := range challenge.Factors {
			if f := normalizeFactor(raw); f != "" {
				factors = append(factors, f)
			}
		}
	}
	if len(factors) == 0 {
		factors = []MFAFactor{FactorTOTP, FactorRecovery}
	}

	var lastErr error
	for _, factor := range factors {
		code, err := prompt(factor)
		if err != nil {
			return nil, err
		}
		code = strings.TrimSpace(code)
		if code == "" {
			return nil, fmt.Errorf("code MFA obligatoire")
		}

		switch factor {
		case FactorTOTP:
			resp, err := a.Connector.VerifyTOTP(ctx, code)
			if err == nil {
				return resp, nil
			}
			lastErr = err
		case FactorRecovery:
			resp, err := a.Connector.VerifyRecovery(ctx, code)
			if err == nil {
				return resp, nil
			}
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("vérification MFA échouée : %w", lastErr)
	}
	return nil, fmt.Errorf("vérification MFA échouée")
}

func normalizeFactor(raw string) MFAFactor {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "totp", "token", "otp", "authenticator":
		return FactorTOTP
	case "recovery", "backup", "recovery_code", "backup_code", "codes":
		return FactorRecovery
	}
	return ""
}
