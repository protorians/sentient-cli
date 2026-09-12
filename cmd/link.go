package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/protorians/sentient-cli/internal/auth"
	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/store"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Lier un module local à un module distant",
	Long: `Lie un module créé dans sentient-connect avec le module local (via son token).

Vérifie l'authentification, liste les modules en ligne, et associe
le module local choisi au token distant fourni.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLink(cmd)
	},
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Délier un module local de sentient-connect",
	Long: `Délie un module local de son correspondant dans sentient-connect.
Le token du manifest.json est remplacé par un nouveau token UUID local.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUnlink(cmd)
	},
}

func runLink(cmd *cobra.Command) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	// Check auth
	sess, err := auth.LoadSession(auth.NewStore())
	if err != nil || sess == nil || !sess.IsAuthenticated() {
		return pkg.NewErrorWithFix("Authentification",
			"vous devez être connecté pour lier un module",
			"Exécutez 'sentients connect' d'abord.", pkg.ExitAuth)
	}

	// Select local module
	localName, err := resolveModule(root, nil)
	if err != nil {
		return err
	}

	// Fetch remote modules
	client := store.NewClient()
	client.SetToken(sess.AccessToken)

	remoteModules, err := tui.RunWithSpinner("Récupération des modules distants…", func() ([]store.RemoteModule, error) {
		return client.ListModules(context.Background())
	})
	if err != nil {
		return pkg.NewErrorWithFix("Store", err.Error(),
			"Vérifiez votre connexion et votre authentification.", pkg.ExitNetwork)
	}

	if len(remoteModules) == 0 {
		return pkg.NewErrorWithFix("Store",
			"aucun module trouvé dans sentient-connect",
			"Publiez d'abord un module avec 'sentients publish'.", pkg.ExitError)
	}

	// Ask for remote token
	remoteToken := ""
	if tui.IsInteractive() {
		var items []string
		for _, m := range remoteModules {
			items = append(items, fmt.Sprintf("%s — %s v%s", m.Token, m.Name, m.Version))
		}
		selected, err := tui.Select("Module distant à lier", items)
		if err != nil {
			return err
		}
		remoteToken = strings.Split(selected, " — ")[0]
	} else {
		return pkg.NewError("Sélection",
			"fournissez le token du module distant en argument",
			pkg.ExitError)
	}

	// Validate remote module
	remote, err := tui.RunWithSpinner("Vérification du module distant…", func() (*store.RemoteModuleResponse, error) {
		return client.GetModule(context.Background(), remoteToken)
	})
	if err != nil {
		return pkg.NewErrorWithFix("Store", err.Error(),
			"Le token distant est invalide ou le module n'existe pas.", pkg.ExitError)
	}

	// Link
	linker := &module.Linker{Root: root}
	result, err := linker.Link(localName, module.RemoteInfo{
		Token:         remote.Token,
		Name:          remote.Name,
		Description:   remote.Description,
		Version:       remote.Version,
		PublisherID:   remote.Publisher.ID,
		PublisherName: remote.Publisher.Name,
	})
	if err != nil {
		return pkg.NewError("Liaison", err.Error(), pkg.ExitError)
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Module lié avec succès"))
	fmt.Printf("  %s : %s\n", s.Muted.Render("Local"), config.ExternalModulesDir+"/"+result.LocalModule+"/")
	fmt.Printf("  %s : %s (%s v%s)\n", s.Muted.Render("Distant"), result.RemoteToken, result.RemoteName, result.RemoteVersion)
	return nil
}

func runUnlink(cmd *cobra.Command) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	linker := &module.Linker{Root: root}

	// Find linked modules (spec §5.8: list local modules with a remote token)
	linked, err := linker.LinkedModules()
	if err != nil {
		return err
	}
	if len(linked) == 0 {
		return pkg.NewError("Module",
			"aucun module lié à un module distant trouvé dans external_modules",
			pkg.ExitModuleNotFound)
	}

	// Select module to unlink from the linked list
	localName := linked[0]
	if len(linked) > 1 {
		if !tui.IsInteractive() {
			return pkg.NewError("Sélection",
				"plusieurs modules liés, fournissez le nom du module en argument",
				pkg.ExitError)
		}
		selected, err := tui.Select("Sélectionner le module à délier", linked)
		if err != nil {
			return err
		}
		localName = selected
	}

	// Show the current remote binding
	if m, err := module.LoadManifest(config.ManifestPath(root, localName)); err == nil && m.Token != "" {
		remote := fmt.Sprintf("%s (v%s)", m.Name, m.Version)
		if m.Name == "" {
			remote = m.Version
		}
		fmt.Printf("  Actuellement lié à : %s %s\n", m.Token, remote)
	}

	// Confirm
	if tui.IsInteractive() {
		confirm, err := tui.Confirm(fmt.Sprintf("Délier le module %q ?", localName), false)
		if err != nil {
			return err
		}
		if !confirm {
			return nil
		}
	}

	if err := linker.Unlink(localName); err != nil {
		return pkg.NewError("Déliaison", err.Error(), pkg.ExitError)
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Module délié avec succès"))
	fmt.Printf("  %s n'est plus lié à un module distant.\n",
		s.Muted.Render(config.ExternalModulesDir+"/"+localName+"/"))
	return nil
}
