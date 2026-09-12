package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/protorians/sentient-cli/internal/auth"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Se connecter à Sentient Connect",
	Long: `Authentifie le développeur avec son compte sentient-connect
(email + mot de passe, MFA prise en charge) et stocke les credentials
de manière sécurisée (keychain système).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConnect(cmd)
	},
}

func runConnect(cmd *cobra.Command) error {
	ctx := context.Background()
	store := auth.NewStore()

	// Existing session?
	sess, err := auth.LoadSession(store)
	if err != nil {
		return pkg.NewError("Authentification", err.Error(), pkg.ExitAuth)
	}
	if sess != nil && sess.IsAuthenticated() {
		email := "utilisateur"
		if sess.User != nil && sess.User.Email != "" {
			email = sess.User.Email
		}
		fmt.Println()
		fmt.Printf("  Déjà connecté en tant que %s\n", email)
		if tui.IsInteractive() {
			again, err := tui.Confirm("Voulez-vous vous reconnecter ?", false)
			if err != nil {
				return err
			}
			if !again {
				return nil
			}
		}
	}

	// Credentials input. In non-interactive (CI) mode the values come from
	// the environment (SENTIENT_CLI_CONNECT_EMAIL/PASSWORD/MFA_CODE) — the
	// same pattern as `SENTIENT_CLI_YES`.
	email := ""
	if tui.IsInteractive() {
		value, err := tui.AskText("Email", "")
		if err != nil {
			return err
		}
		email = strings.TrimSpace(value)
	} else {
		email = strings.TrimSpace(os.Getenv("SENTIENT_CLI_CONNECT_EMAIL"))
	}
	if email == "" {
		return pkg.NewError("Authentification", "l'email est obligatoire", pkg.ExitAuth)
	}

	password := ""
	if tui.IsInteractive() {
		value, err := tui.AskSecret("Mot de passe")
		if err != nil {
			return err
		}
		password = value
	} else {
		password = os.Getenv("SENTIENT_CLI_CONNECT_PASSWORD")
	}
	if password == "" {
		return pkg.NewError("Authentification", "le mot de passe est obligatoire", pkg.ExitAuth)
	}

	connector := auth.NewConnector()
	debugf("API sentient-connect : %s", connector.Client.BaseURL)

	signIn, err := tui.RunWithSpinner("Vérification des identifiants", func() (*auth.SignInResponse, error) {
		return connector.SignIn(ctx, auth.SignInRequest{Email: email, Password: password})
	})
	if err != nil {
		return classifyConnectorError("Authentification", err, "Vérifiez vos identifiants.")
	}

	session := &auth.Session{
		Store:       store,
		AccessToken: signIn.Token,
		Device:      signIn.Device.ID,
		User:        &signIn.User,
	}
	t := time.Now().Add(auth.TokenTTL)
	session.ExpiresAt = &t

	// MFA: challenge and verification run against the guarded `/api/mfa/*`
	// endpoints, so the session token must be attached to the connector first.
	authn := &auth.Authenticator{Connector: connector}
	connector.Client.Token = session.AccessToken
	mfaResp, err := runMFA(ctx, authn)
	if err != nil {
		return err
	}
	if mfaResp != nil && mfaResp.MFAVerified && mfaResp.MFAToken != "" {
		session.MFAToken = mfaResp.MFAToken
	}

	if err := session.Save(); err != nil {
		return pkg.NewError("Authentification", "stockage des credentials impossible : "+err.Error(), pkg.ExitAuth)
	}

	printConnectSummary(session)
	return nil
}

func runMFA(ctx context.Context, authn *auth.Authenticator) (*auth.VerifyResponse, error) {
	prompt := func(factor auth.FactorKind) (string, error) {
		label := "Code TOTP"
		if factor == auth.FactorRecovery {
			label = "Code de récupération"
		}
		if !tui.IsInteractive() {
			// CI: read the verification code from the environment.
			if code := os.Getenv("SENTIENT_CLI_MFA_CODE"); code != "" {
				return code, nil
			}
			return "", pkg.NewError("MFA", "cette exécution nécessite un terminal interactif pour la vérification MFA", pkg.ExitMFA)
		}
		return tui.AskText(label, "")
	}

	resp, err := tui.RunWithSpinner("Vérification du code…", func() (*auth.VerifyResponse, error) {
		return authn.Verify(ctx, prompt)
	})
	if err != nil {
		return nil, classifyConnectorError("MFA", err,
			"Le code est incorrect ou expiré. Vérifiez votre application TOTP / vos codes de récupération (backup codes).")
	}
	return resp, nil
}

func classifyConnectorError(category string, err error, fix string) error {
	if _, ok := err.(*pkg.APIError); ok {
		return pkg.NewErrorWithFix(category, err.Error(), fix, pkg.ExitAuth)
	}
	return pkg.NewErrorWithFix("Réseau", err.Error(),
		"Vérifiez votre connexion internet et la disponibilité de sentient-connect.", pkg.ExitNetwork)
}

func printConnectSummary(session *auth.Session) {
	s := tui.NewStyles()
	email := ""
	role := "Développeur"
	if session.User != nil {
		email = session.User.Email
		role = session.User.Role
		if role == "" {
			role = "Développeur"
		}
	}
	expiry := "—"
	if session.ExpiresAt != nil {
		expiry = session.ExpiresAt.UTC().Format("2006-01-02 15:04:05 UTC")
	}
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Connecté en tant que " + email))
	fmt.Printf("  %s : %s\n", s.Muted.Render("Rôle"), role)
	fmt.Printf("  %s : %s\n", s.Muted.Render("Token expire le"), expiry)
}
