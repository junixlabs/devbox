package registry

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/junixlabs/devbox/internal/template"
	"gopkg.in/yaml.v3"
)

// DefaultRegistryURL is the base URL for the community template registry.
const DefaultRegistryURL = "https://raw.githubusercontent.com/junixlabs/devbox-templates/main"

// maxResponseBytes limits HTTP response body size to prevent memory exhaustion.
const maxResponseBytes = 1 << 20 // 1 MB

// IndexEntry describes a single template in the remote registry index.
type IndexEntry struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	Description string    `yaml:"description"`
	URL         string    `yaml:"url"`
	UpdatedAt   time.Time `yaml:"updated_at"`
}

// Index is the top-level structure of the registry index file.
type Index struct {
	Templates []IndexEntry `yaml:"templates"`
}

// RemoteRegistry fetches templates from an HTTP-based registry.
type RemoteRegistry struct {
	baseURL string
	client  *http.Client
}

// NewRemoteRegistry creates a new remote registry client.
// If baseURL is empty, DefaultRegistryURL is used.
func NewRemoteRegistry(baseURL string) *RemoteRegistry {
	if baseURL == "" {
		baseURL = DefaultRegistryURL
	}
	return &RemoteRegistry{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchIndex downloads and parses the registry index.
func (r *RemoteRegistry) FetchIndex() ([]IndexEntry, error) {
	url := r.baseURL + "/index.yaml"
	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read registry index: %w", err)
	}

	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse registry index: %w", err)
	}

	return idx.Templates, nil
}

// Search returns index entries matching the query (case-insensitive substring
// match on name or description).
func (r *RemoteRegistry) Search(query string) ([]IndexEntry, error) {
	entries, err := r.FetchIndex()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(query)
	var matches []IndexEntry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Description), q) {
			matches = append(matches, e)
		}
	}
	return matches, nil
}

// Pull downloads a template by name from the registry and saves it to the
// local registry.
func (r *RemoteRegistry) Pull(name string, localReg *template.LocalRegistry) (*template.Template, error) {
	entries, err := r.FetchIndex()
	if err != nil {
		return nil, err
	}

	var entry *IndexEntry
	for i := range entries {
		if entries[i].Name == name {
			entry = &entries[i]
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("template %q not found in registry", name)
	}

	// Only allow relative URLs to prevent SSRF via malicious index entries.
	templateURL := entry.URL
	if strings.HasPrefix(templateURL, "http://") || strings.HasPrefix(templateURL, "https://") {
		return nil, fmt.Errorf("template %q has absolute URL which is not allowed", name)
	}
	templateURL = r.baseURL + "/" + strings.TrimLeft(templateURL, "/")

	resp, err := r.client.Get(templateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download template %q: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download template %q: HTTP %d", name, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read template %q: %w", name, err)
	}

	var t template.Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %w", name, err)
	}

	if t.Name == "" {
		t.Name = name
	}

	if err := localReg.Save(&t); err != nil {
		return nil, fmt.Errorf("failed to save template %q: %w", name, err)
	}

	return &t, nil
}

// Push reads a template from the local registry and outputs its YAML content
// for manual submission to the community registry.
func (r *RemoteRegistry) Push(name string, localReg *template.LocalRegistry) (string, error) {
	t, err := localReg.Get(name)
	if err != nil {
		return "", fmt.Errorf("failed to read local template %q: %w", name, err)
	}

	data, err := yaml.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template %q: %w", name, err)
	}

	return string(data), nil
}
