package cmd

import (
	"context"
	"fmt"
	"path/filepath"
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
	Use:   "link [module] [token]",
	Short: "Lier un module local à un module distant",
	Long: `Lie un module créé dans sentient-connect avec le module local (via son token).

Vérifie l'authentification, liste les modules en ligne, et associe
le module local choisi au token distant fourni.

En mode non interactif (CI), fournissez le module local et le token
distant en arguments : link <module> <token>.`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLink(cmd, args)
	},
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink [module]",
	Short: "Délier un module local de sentient-connect",
	Long: `Délie un module local de son correspondant dans sentient-connect.
Le token du manifest.json est remplacé par un nouveau token UUID local.

--sync-remote synchronise d'abord les métadonnées locales (nom, type,
description) vers le produit distant via PUT /api/developer-store/modules/:id
(best-effort, nécessite d'être connecté).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUnlink(cmd, args)
	},
}

var flagUnlinkSyncRemote bool

func init() {
	unlinkCmd.Flags().BoolVar(&flagUnlinkSyncRemote, "sync-remote", false,
		"mettre à jour les métadonnées du module distant avant la déliaison")
}

func runLink(cmd *cobra.Command, args []string) error {
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

	// Non-interactive: module + token supplied as arguments.
	if !tui.IsInteractive() && len(args) < 2 {
		return pkg.NewErrorWithFix("Sélection",
			"le mode non interactif exige le module local et le token distant",
			"Utilisez 'sentients link <module> <token>'.", pkg.ExitError)
	}

	// Select local module
	moduleArg := []string{}
	if len(args) > 0 {
		moduleArg = []string{args[0]}
	}
	localName, err := resolveModule(root, moduleArg)
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

	if len(remoteModules) == 0 && len(args) < 2 {
		return pkg.NewErrorWithFix("Store",
			"aucun module trouvé dans sentient-connect",
			"Publiez d'abord un module avec 'sentients publish', ou fournissez le token en argument : 'sentients link <module> <token>'.",
			pkg.ExitError)
	}

	// Ask for remote token
	remoteToken := ""
	if len(args) > 1 {
		remoteToken = args[1]
	} else if tui.IsInteractive() {
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

func runUnlink(cmd *cobra.Command, args []string) error {
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

	// Select module to unlink: positional arg, or interactive menu.
	localName := ""
	if len(args) > 0 {
		localName = args[0]
		if !pkg.DirExists(filepath.Join(root, config.ExternalModulesDir, localName)) {
			return pkg.NewErrorWithFix(
				"Module",
				fmt.Sprintf("le module %q n'existe pas dans %s", localName, config.ExternalModulesDir),
				"Vérifiez le nom du module.",
				pkg.ExitModuleNotFound,
			)
		}
		linkedOk := false
		for _, n := range linked {
			if n == localName {
				linkedOk = true
				break
			}
		}
		if !linkedOk {
			return pkg.NewError("Module",
				fmt.Sprintf("le module %q n'est pas lié à un module distant", localName),
				pkg.ExitError)
		}
	} else if len(linked) > 1 {
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
	} else {
		localName = linked[0]
	}

	// Show the current remote binding
	if m, err := module.LoadManifest(config.ManifestPath(root, localName)); err == nil && m.Token != "" {
		remote := fmt.Sprintf("%s (v%s)", m.Name, m.Version)
		if m.Name == "" {
			remote = m.Version
		}
		fmt.Printf("  Actuellement lié à : %s %s\n", m.Token, remote)
	}

	// Optional remote metadata sync before unlinking (PUT /modules/:id).
	if flagUnlinkSyncRemote {
		remoteToken := linker.RemoteToken(localName)
		if remoteToken == "" {
			warn("aucun module distant associé, synchronisation ignorée")
		} else if err := syncRemoteBeforeUnlink(root, localName, remoteToken); err != nil {
			debugf("synchronisation distante avant déliaison : %v", err)
			warn("synchronisation distante impossible : " + err.Error())
		} else {
			fmt.Println("  ✓ Métadonnées synchronisées vers le module distant")
		}
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

// syncRemoteBeforeUnlink pushes the local manifest metadata to the remote
// product (best-effort) — the developer-store PUT endpoint.
func syncRemoteBeforeUnlink(root, localName, remoteToken string) error {
	sess, err := auth.LoadSession(auth.NewStore())
	if err != nil || sess == nil || !sess.IsAuthenticated() {
		return fmt.Errorf("non connecté — exécutez 'sentients connect'")
	}
	m, err := module.LoadManifest(config.ManifestPath(root, localName))
	if err != nil {
		return err
	}
	client := store.NewClient()
	client.SetToken(sess.AccessToken)
	return client.UpdateModule(context.Background(), remoteToken, m)
}
