package plugin

// Type represents the kind of plugin.
type Type string

const (
	TypeProvider Type = "provider"
	TypeHook     Type = "hook"
)

// ValidTypes lists all accepted plugin types.
var ValidTypes = []Type{TypeProvider, TypeHook}

// IsValid returns true if the type is a known plugin type.
func (t Type) IsValid() bool {
	for _, v := range ValidTypes {
		if t == v {
			return true
		}
	}
	return false
}

// Event represents a lifecycle event that hooks can subscribe to.
type Event string

const (
	PreCreate    Event = "pre_create"
	PostCreate   Event = "post_create"
	PreStart     Event = "pre_start"
	PostStart    Event = "post_start"
	PreStop      Event = "pre_stop"
	PostStop     Event = "post_stop"
	PreDestroy   Event = "pre_destroy"
	PostDestroy  Event = "post_destroy"
)

// HookContext carries information about the lifecycle event being processed.
type HookContext struct {
	WorkspaceName string
	Event         Event
	ServerHost    string
	Env           map[string]string
}

// Provider defines the interface for container backend plugins (Docker, Podman, LXC).
type Provider interface {
	// Create provisions a new container/workspace.
	Create(name string, image string, opts CreateOpts) error

	// Start starts a stopped container.
	Start(name string) error

	// Stop stops a running container.
	Stop(name string) error

	// Destroy removes a container and its resources.
	Destroy(name string) error

	// Status returns the current status of a container.
	Status(name string) (string, error)

	// ProviderName returns the name of this provider (e.g. "docker").
	ProviderName() string
}

// CreateOpts holds optional parameters for container creation.
type CreateOpts struct {
	Ports     map[string]int
	Env       map[string]string
	CPUs      float64
	Memory    string
	Volumes   []string
	Command   string
}

// Hook defines the interface for lifecycle hook plugins.
type Hook interface {
	// Execute runs the hook for the given event.
	Execute(ctx HookContext) error

	// Events returns the events this hook subscribes to.
	Events() []Event

	// HookName returns the name of this hook.
	HookName() string
}

// Manifest describes a plugin's metadata, loaded from plugin.yaml.
type Manifest struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Type        Type   `yaml:"type"`
	Entrypoint  string `yaml:"entrypoint"`
	Description string `yaml:"description,omitempty"`
	Events      []Event `yaml:"events,omitempty"`
	BuiltIn     bool   `yaml:"-"`
}
