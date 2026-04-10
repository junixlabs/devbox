package errors

import "fmt"

// ConfigError represents a configuration-related error with an actionable suggestion.
type ConfigError struct {
	Message    string
	Suggestion string
	Err        error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ConfigError) Unwrap() error { return e.Err }

// NewConfigError creates a ConfigError with a user-facing suggestion.
func NewConfigError(message, suggestion string, err error) *ConfigError {
	return &ConfigError{Message: message, Suggestion: suggestion, Err: err}
}

// ConnectionError represents an SSH or network connectivity error.
type ConnectionError struct {
	Message    string
	Suggestion string
	Err        error
}

func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ConnectionError) Unwrap() error { return e.Err }

// NewConnectionError creates a ConnectionError with a user-facing suggestion.
func NewConnectionError(message, suggestion string, err error) *ConnectionError {
	return &ConnectionError{Message: message, Suggestion: suggestion, Err: err}
}

// DockerError represents a Docker-related error.
type DockerError struct {
	Message    string
	Suggestion string
	Err        error
}

func (e *DockerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *DockerError) Unwrap() error { return e.Err }

// NewDockerError creates a DockerError with a user-facing suggestion.
func NewDockerError(message, suggestion string, err error) *DockerError {
	return &DockerError{Message: message, Suggestion: suggestion, Err: err}
}

// Suggestible is implemented by errors that carry an actionable suggestion.
type Suggestible interface {
	error
	GetSuggestion() string
}

func (e *ConfigError) GetSuggestion() string     { return e.Suggestion }
func (e *ConnectionError) GetSuggestion() string  { return e.Suggestion }
func (e *DockerError) GetSuggestion() string      { return e.Suggestion }
