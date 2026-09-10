package cmd

import (
	"context"
	"fmt"
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

	// Credentials input
	email := ""
	if value, err := askOptionalText("Email", ""); err != nil {
		return err
	} else {
		email = strings.TrimSpace(value)
	}
	if email == "" {
		return pkg.NewError("Authentification", "l'email est obligatoire", pkg.ExitAuth)
	}

	password, err := tui.AskSecret("Mot de passe")
	if err != nil {
		return err
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
		Store:        store,
		AccessToken:  signIn.AccessToken,
		RefreshToken: signIn.RefreshToken,
		User:         &signIn.User,
	}
	if signIn.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(signIn.ExpiresIn) * time.Second)
		session.ExpiresAt = &t
	}

	if signIn.MFARequired {
		if err := runMFA(ctx, connector, email, session); err != nil {
			return err
		}
	}

	if err := session.Save(); err != nil {
		return pkg.NewError("Authentification", "stockage des credentials impossible : "+err.Error(), pkg.ExitAuth)
	}

	printConnectSummary(session)
	return nil
}

func runMFA(ctx context.Context, connector *auth.Connector, email string, session *auth.Session) error {
	authn := &auth.Authenticator{Connector: connector}
	prompt := func(factor auth.MFAFactor) (string, error) {
		label := "Code TOTP"
		if factor == auth.FactorRecovery {
			label = "Code de récupération"
		}
		if !tui.IsInteractive() {
			return "", pkg.NewError("MFA", "cette exécution nécessite un terminal interactif pour la vérification MFA", pkg.ExitMFA)
		}
		return tui.AskText(label, "")
	}

	tok, err := tui.RunWithSpinner("Vérification du code…", func() (*auth.TokenResponse, error) {
		return authn.Verify(ctx, email, prompt)
	})
	if err != nil {
		return classifyConnectorError("MFA", err,
			"Le code est incorrect ou expiré. Vérifiez votre application TOTP / vos codes de récupération (backup codes).")
	}

	session.AccessToken = tok.AccessToken
	session.RefreshToken = tok.RefreshToken
	if tok.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
		session.ExpiresAt = &t
	}
	if tok.User.ID != "" {
		session.User = &tok.User
	}
	return nil
}

func classifyConnectorError(category string, err error, fix string) error {
	if _, ok := err.(*pkg.APIError); ok {
		return pkg.NewErrorWithFix(category, err.Error(), fix, pkg.ExitAuth)
	}
	return pkg.NewErrorWithFix("Réseau", err.Error(),
		"Vérifiez votre connexion internet et la disponibilité de sentient-connect.", pkg.ExitNetwork)
}

func askOptionalText(title, placeholder string) (string, error) {
	if tui.IsInteractive() {
		return tui.AskText(title, placeholder)
	}
	return "", nil
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
