package cmd

import (
	"context"
	"fmt"

	"github.com/protorians/sentient-cli/internal/auth"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Se déconnecter",
	Long:  "Supprime toutes les credentials stockées et déconnecte le développeur.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDisconnect(cmd)
	},
}

func runDisconnect(cmd *cobra.Command) error {
	sess, err := auth.LoadSession(auth.NewStore())
	if err != nil {
		return pkg.NewError("Authentification", err.Error(), pkg.ExitAuth)
	}
	if sess == nil || !sess.IsAuthenticated() {
		fmt.Println()
		fmt.Println(tui.NewStyles().Muted.Render("Vous n'êtes pas connecté — rien à faire."))
		return nil
	}

	email := ""
	if sess.User != nil {
		email = sess.User.Email
	}
	fmt.Println()
	fmt.Printf("  Déconnecter %s ?\n", email)
	confirm, err := tui.Confirm("Confirmer la déconnexion", false)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	// Best-effort server-side token invalidation (FR-008, POST /api/auth/logout).
	connector := auth.NewConnector()
	connector.Client.Token = sess.AccessToken
	if err := connector.SignOut(context.Background(), sess.Device); err != nil {
		debugf("invalidation du token côté serveur : %v", err)
	}

	if _, err := tui.RunWithSpinner("Déconnexion…", func() (struct{}, error) {
		return struct{}{}, sess.Clear()
	}); err != nil {
		return pkg.NewError("Authentification", "suppression des credentials impossible : "+err.Error(), pkg.ExitAuth)
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Déconnecté avec succès"))
	fmt.Println(s.Muted.Render("  Toutes les credentials ont été supprimées."))
	return nil
}
