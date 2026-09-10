package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/protorians/sentient-cli/internal/auth"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// RemoteModule is a module registered on the store.
type RemoteModule struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
}

// RemoteModuleResponse is the response from GET /store/modules/:token.
type RemoteModuleResponse struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Publisher   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"publisher"`
}

// PublishResponse is the response from POST /store/modules/publish.
type PublishResponse struct {
	Token   string `json:"token"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

// Client talks to the store API.
type Client struct {
	Connector *auth.Connector
}

// NewClient builds a store client with the current session token.
func NewClient() *Client {
	return &Client{Connector: auth.NewConnector()}
}

// SetToken sets the bearer token for API calls.
func (c *Client) SetToken(token string) {
	c.Connector.Client.Token = token
}

// ListModules returns the modules owned by the authenticated developer.
func (c *Client) ListModules(ctx context.Context) ([]RemoteModule, error) {
	var modules []RemoteModule
	if err := c.Connector.Client.Do(ctx, "GET", "/store/modules", nil, &modules); err != nil {
		return nil, fmt.Errorf("récupération des modules échouée : %w", err)
	}
	return modules, nil
}

// GetModule returns a remote module by its token.
func (c *Client) GetModule(ctx context.Context, token string) (*RemoteModuleResponse, error) {
	var mod RemoteModuleResponse
	if err := c.Connector.Client.Do(ctx, "GET", "/store/modules/"+token, nil, &mod); err != nil {
		return nil, fmt.Errorf("récupération du module %q échouée : %w", token, err)
	}
	return &mod, nil
}

// Publish sends a module archive to the store.
func (c *Client) Publish(ctx context.Context, archivePath string, manifest *module.Manifest) (*PublishResponse, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("ouverture de l'archive impossible : %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("archive", archivePath)
	if err != nil {
		return nil, fmt.Errorf("construction du formulaire impossible : %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copie de l'archive impossible : %w", err)
	}

	manifestPart, err := writer.CreateFormField("manifest")
	if err != nil {
		return nil, fmt.Errorf("ajout du manifest au formulaire impossible : %w", err)
	}
	if err := json.NewEncoder(manifestPart).Encode(manifest); err != nil {
		return nil, fmt.Errorf("sérialisation du manifest impossible : %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", c.Connector.Client.BaseURL+"/store/modules/publish", &buf)
	if err != nil {
		return nil, fmt.Errorf("construction de la requête impossible : %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.Connector.Client.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Connector.Client.Token)
	}
	req.Header.Set("User-Agent", "sentient-cli")

	resp, err := c.Connector.Client.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur réseau : %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lecture de la réponse impossible : %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr pkg.APIError
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Message != "" {
			apiErr.StatusCode = resp.StatusCode
			return nil, &apiErr
		}
		return nil, fmt.Errorf("publication échouée (HTTP %d) : %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var pubResp PublishResponse
	if err := json.Unmarshal(data, &pubResp); err != nil {
		return nil, fmt.Errorf("décodage de la réponse impossible : %w", err)
	}
	return &pubResp, nil
}
