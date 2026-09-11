package cmd

import (
	"crypto/ed25519"
	"fmt"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/signing"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var signCmd = &cobra.Command{
	Use:   "sign [module]",
	Short: "Signer les archives .smp (Ed25519)",
	Long: `Gère les signatures numériques Ed25519 des modules : génère des clés,
signe les archives .smp et vérifie les signatures.

Sous-commandes :
  sign keygen            Générer une paire de clés Ed25519
  sign <module>          Signer l'archive .smp d'un module
  sign verify <module>   Vérifier la signature d'un module

Sans argument, affiche le fingerprint SHA-256 de la clé publique.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return printSigningFingerprint()
		}
		return signModule(args)
	},
}

func init() {
	signCmd.AddCommand(signKeygenCmd, signVerifyCmd)
}

var signKeygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Générer une paire de clés Ed25519",
	Long: `Génère une paire de clés Ed25519 et la stocke dans le keychain système
(repli : fichier chiffré ~/.sentient-cli/signing.enc).

Si des clés existent déjà, demande confirmation avant de les écraser
(régénération silencieuse en mode non interactif).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSignKeygen()
	},
}

var signVerifyCmd = &cobra.Command{
	Use:   "verify [module]",
	Short: "Vérifier la signature d'un module",
	Long: `Vérifie la validité du fichier .sig d'un module par rapport à son
archive .smp, à l'aide de la clé publique stockée dans le keychain.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return verifySignature(args)
	},
}

// printSigningFingerprint affiche le fingerprint SHA-256 de la clé publique (FR-024).
func printSigningFingerprint() error {
	store := signing.NewKeyStore()
	pub, err := store.GetPublicKey()
	if err != nil {
		return pkg.NewErrorWithFix("Signature",
			"aucune clé de signature trouvée dans le keychain",
			"Exécutez 'sentient sign keygen' pour générer une paire de clés.",
			pkg.ExitSigning)
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.SubHeader.Render("Clé de signature du développeur :"))
	fmt.Printf("  %s : %s\n", s.Muted.Render("Fingerprint"), s.Info.Render(signing.Fingerprint(ed25519.PublicKey(pub))))
	return nil
}

// runSignKeygen génère une paire de clés Ed25519 et la stocke (spec §5.13.1).
func runSignKeygen() error {
	store := signing.NewKeyStore()

	if store.HasKeys() {
		s := tui.NewStyles()
		fmt.Println()
		fmt.Println(s.Warning.Render("⚠ Des clés de signature existent déjà"))
		if pub, err := store.GetPublicKey(); err == nil {
			fmt.Printf("  %s : %s\n", s.Muted.Render("Fingerprint actuel"), s.Info.Render(signing.Fingerprint(ed25519.PublicKey(pub))))
		}
		if tui.IsInteractive() {
			regenerate, err := tui.Confirm("Régénérer la paire de clés (écrase les précédentes)", false)
			if err != nil {
				return err
			}
			if !regenerate {
				return nil
			}
		}
	}

	pub, priv, err := signing.GenerateKeyPair()
	if err != nil {
		return pkg.NewError("Signature", err.Error(), pkg.ExitSigning)
	}

	_, err = tui.RunWithSpinner("Sauvegarde des clés dans le keychain…", func() (struct{}, error) {
		return struct{}{}, signing.SaveKeyPair(store, pub, priv)
	})
	if err != nil {
		return pkg.NewError("Signature", err.Error(), pkg.ExitSigning)
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Paire de clés Ed25519 générée avec succès"))
	fmt.Printf("  %s : %s (SHA-256 de la clé publique)\n", s.Muted.Render("Fingerprint"), s.Info.Render(signing.Fingerprint(pub)))
	fmt.Printf("  %s : stockée dans le keychain système\n", s.Muted.Render("Clé privée"))
	fmt.Printf("  %s : stockée dans le keychain système\n", s.Muted.Render("Clé publique"))
	return nil
}

// signModule signe l'archive .smp d'un module (spec §5.13.2).
func signModule(args []string) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	name, err := resolveModule(root, args)
	if err != nil {
		return err
	}

	m, err := module.LoadManifest(config.ManifestPath(root, name))
	if err != nil {
		return pkg.NewError("Manifest", err.Error(), pkg.ExitManifest)
	}

	archivePath, err := signing.FindArchive(root, name, m.Version)
	if err != nil {
		return pkg.NewErrorWithFix("Signature",
			err.Error(),
			"Exécutez 'sentient pack "+name+"' pour construire l'archive d'abord.",
			pkg.ExitSigning)
	}

	sigPath := archivePath + ".sig"
	if pkg.FileExists(sigPath) && tui.IsInteractive() {
		overwrite, err := tui.Confirm("Un fichier .sig existe déjà, l'écraser", false)
		if err != nil {
			return err
		}
		if !overwrite {
			return nil
		}
	}

	keys, err := tui.RunWithSpinner("Chargement de la clé de signature…", func() (*signingKey, error) {
		pub, priv, err := signing.LoadKeyPair(signing.NewKeyStore())
		if err != nil {
			return nil, err
		}
		return &signingKey{pub: pub, priv: priv}, nil
	})
	if err != nil {
		return pkg.NewErrorWithFix("Signature",
			"clé de signature introuvable : "+err.Error(),
			"Exécutez 'sentient sign keygen' pour générer une paire de clés.",
			pkg.ExitSigning)
	}

	sigPath, err = tui.RunWithSpinner("Signature de l'archive…", func() (string, error) {
		return signing.SignArchive(archivePath, keys.priv)
	})
	if err != nil {
		return pkg.NewError("Signature", err.Error(), pkg.ExitSigning)
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Archive signée avec succès"))
	fmt.Printf("  %s    : %s v%s\n", s.Muted.Render("Module"), name, m.Version)
	fmt.Printf("  %s   : %s\n", s.Muted.Render("Archive"), s.Info.Render(archivePath))
	fmt.Printf("  %s : %s\n", s.Muted.Render("Signature"), s.Info.Render(sigPath))
	fmt.Printf("  %s : %s\n", s.Muted.Render("Signataire"), s.Info.Render(signing.Fingerprint(keys.pub)))
	return nil
}

// verifySignature vérifie la signature .sig d'un module (spec §5.13.3).
func verifySignature(args []string) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	name, err := resolveModule(root, args)
	if err != nil {
		return err
	}

	m, err := module.LoadManifest(config.ManifestPath(root, name))
	if err != nil {
		return pkg.NewError("Manifest", err.Error(), pkg.ExitManifest)
	}

	archivePath, err := signing.FindArchive(root, name, m.Version)
	if err != nil {
		return pkg.NewErrorWithFix("Signature",
			err.Error(),
			"Exécutez 'sentient pack "+name+"' pour construire l'archive d'abord.",
			pkg.ExitSigning)
	}

	sigPath := archivePath + ".sig"
	if !pkg.FileExists(sigPath) {
		return pkg.NewErrorWithFix("Signature",
			fmt.Sprintf("fichier de signature introuvable : %s", sigPath),
			"Exécutez 'sentient sign "+name+"' pour signer l'archive.",
			pkg.ExitSigning)
	}

	valid, err := tui.RunWithSpinner("Vérification de la signature…", func() (bool, error) {
		pubBytes, err := signing.NewKeyStore().GetPublicKey()
		if err != nil {
			return false, err
		}
		return signing.VerifySignature(archivePath, sigPath, ed25519.PublicKey(pubBytes))
	})
	if err != nil {
		return pkg.NewErrorWithFix("Signature",
			"clé de vérification introuvable : "+err.Error(),
			"Exécutez 'sentient sign keygen' pour générer une paire de clés.",
			pkg.ExitSigning)
	}

	s := tui.NewStyles()
	fmt.Println()
	if valid {
		fmt.Println(s.Success.Render("✓ Signature valide"))
		fmt.Printf("  %s     : %s v%s\n", s.Muted.Render("Module"), name, m.Version)
		fmt.Printf("  %s    : %s\n", s.Muted.Render("Archive"), s.Info.Render(archivePath))
		if pub, err := signing.NewKeyStore().GetPublicKey(); err == nil {
			fmt.Printf("  %s : %s\n", s.Muted.Render("Signataire"), s.Info.Render(signing.Fingerprint(ed25519.PublicKey(pub))))
		}
		return nil
	}

	fmt.Println(s.Error.Render("✗ Signature invalide"))
	fmt.Printf("  %s     : %s v%s\n", s.Muted.Render("Module"), name, m.Version)
	fmt.Printf("  %s    : %s\n", s.Muted.Render("Archive"), s.Info.Render(archivePath))
	fmt.Println("  → L'archive a pu être modifiée ou la clé de vérification est incorrecte.")
	return pkg.NewError("Signature", "signature invalide", pkg.ExitSigning)
}

type signingKey struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}
