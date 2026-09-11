package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/protorians/sentient-cli/internal/audit"
	"github.com/protorians/sentient-cli/internal/auth"
	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/store"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish [module]",
	Short: "Publier un module sur le store",
	Long: `Construit et publie un module dans le store via l'API sentient-connect.

Vérifie l'authentification, valide le manifest, construit l'archive .smp,
puis envoie au store.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPublish(cmd, args)
	},
}

func runPublish(cmd *cobra.Command, args []string) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	// Check auth
	sess, err := auth.LoadSession(auth.NewStore())
	if err != nil || sess == nil || !sess.IsAuthenticated() {
		return pkg.NewErrorWithFix("Authentification",
			"vous devez être connecté pour publier un module",
			"Exécutez 'sentient connect' d'abord.", pkg.ExitAuth)
	}

	email := ""
	if sess.User != nil {
		email = sess.User.Email
	}

	// Identify module
	name, err := resolveModule(root, args)
	if err != nil {
		return err
	}

	// Load and validate manifest
	manifestPath := config.ManifestPath(root, name)
	manifest, err := module.LoadManifest(manifestPath)
	if err != nil {
		return pkg.NewError("Manifest", err.Error(), pkg.ExitManifest)
	}

	// Auto-audit before publishing (spec §5.6) — configurable via
	// `[publish] auto_audit` in `.sentient-cli.toml` (default: true).
	cfg, err := config.Load(config.ConfigPath(root))
	if err != nil {
		debugf("lecture de la configuration : %v", err)
	}
	if cfg.Publish.AutoAudit {
		auditRes, aerr := (&audit.Auditor{Root: root}).AuditModules(name)
		if aerr != nil {
			return pkg.NewError("Audit", aerr.Error(), pkg.ExitError)
		}
		if errs := auditRes.TotalErrors(); errs > 0 {
			printAuditResult(auditRes)
			if !tui.IsInteractive() {
				return pkg.NewErrorWithFix("Audit",
					fmt.Sprintf("le module %q contient %d erreur(s) d'audit", name, errs),
					"Corrigez les erreurs puis réessayez, ou exécutez 'sentient audit "+name+"'.",
					pkg.ExitError)
			}
			continueAnyway, cerr := tui.Confirm("Publier malgré les erreurs d'audit", false)
			if cerr != nil {
				return cerr
			}
			if !continueAnyway {
				return nil
			}
			warn("Publication malgré les erreurs d'audit")
		}
	}

	// Check if metadata is incomplete and prompt
	if tui.IsInteractive() {
		updated := false
		if manifest.Name == "" || manifest.Name == name {
			n, err := tui.AskText("Nom affiché du module", manifest.Name)
			if err != nil {
				return err
			}
			if strings.TrimSpace(n) != "" {
				manifest.Name = strings.TrimSpace(n)
				updated = true
			}
		}
		if manifest.Description == "" {
			d, err := tui.AskText("Description du module", "")
			if err != nil {
				return err
			}
			if strings.TrimSpace(d) != "" {
				manifest.Description = strings.TrimSpace(d)
				updated = true
			}
		}
		if manifest.Publisher.ID == "" {
			p, err := tui.AskText("ID développeur", "")
			if err != nil {
				return err
			}
			if strings.TrimSpace(p) != "" {
				manifest.Publisher.ID = strings.TrimSpace(p)
				updated = true
			}
		}
		if manifest.Publisher.Name == "" {
			pn, err := tui.AskText("Nom affiché du développeur", "")
			if err != nil {
				return err
			}
			if strings.TrimSpace(pn) != "" {
				manifest.Publisher.Name = strings.TrimSpace(pn)
				updated = true
			}
		}
		if updated {
			if err := manifest.Save(manifestPath); err != nil {
				warn("Mise à jour du manifest impossible : " + err.Error())
			}
		}
	}

	// Display metadata summary
	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Authentifié : " + email))
	fmt.Println(s.SubHeader.Render("  Métadonnées du module :"))
	fmt.Printf("    %s : %s\n", s.Muted.Render("Nom"), manifest.Name)
	fmt.Printf("    %s : %s\n", s.Muted.Render("Description"), manifest.Description)
	fmt.Printf("    %s : %s\n", s.Muted.Render("Version"), manifest.Version)

	// Confirm
	if tui.IsInteractive() {
		confirm, err := tui.Confirm("Confirmer la publication", true)
		if err != nil {
			return err
		}
		if !confirm {
			return nil
		}
	}

	// Pack
	packer := &module.Packer{Root: root}
	packResult, err := tui.RunWithSpinner("Construction de l'archive…", func() (*module.PackResult, error) {
		return packer.Pack(name)
	})
	if err != nil {
		if _, ok := err.(*pkg.Error); ok {
			return err
		}
		return pkg.NewError("Pack", err.Error(), pkg.ExitBuild)
	}

	// Publish
	client := store.NewClient()
	client.SetToken(sess.AccessToken)

	pubResult, err := tui.RunWithSpinner("Publication sur le store…", func() (*store.PublishResponse, error) {
		return client.Publish(context.Background(), packResult.Path, manifest)
	})
	if err != nil {
		return pkg.NewErrorWithFix("Publication", err.Error(),
			"Vérifiez votre connexion et réessayez.", pkg.ExitPublish)
	}

	fmt.Println()
	fmt.Println(s.Success.Render("✓ Publié avec succès"))
	fmt.Printf("  %s : %s v%s\n", s.Muted.Render("Module"), name, pubResult.Version)
	if pubResult.URL != "" {
		fmt.Printf("  %s : %s\n", s.Muted.Render("URL"), s.Info.Render(pubResult.URL))
	}
	return nil
}
