package provider

import (
	"fmt"
	"sync"
)

type ProviderInfo struct {
	Name     string `json:"name"`
	Type     Type   `json:"type"`
	Label    string `json:"label"`
	Anon     bool   `json:"supports_anonymous"`
	Remote   bool   `json:"supports_remote_url"`
	HasAPI   bool   `json:"has_api"`
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	return p, nil
}

func (r *Registry) List() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]ProviderInfo, 0)
	for _, p := range r.providers {
		list = append(list, ProviderInfo{
			Name:   p.Name(),
			Type:   p.Type(),
			Label:  typeLabel(p.Type()),
			Anon:   p.SupportsAnonymous(),
			Remote: p.SupportsRemoteURL(),
			HasAPI: p.HasAPI(),
		})
	}
	return list
}

func typeLabel(t Type) string {
	switch t {
	case TypeVideoHost:
		return "Video Host"
	case TypeStorage:
		return "Storage"
	case TypeBoth:
		return "Video + Storage"
	default:
		return "Unknown"
	}
}

func (r *Registry) ListByType(t Type) []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []ProviderInfo
	for _, p := range r.providers {
		if p.Type() == t {
			list = append(list, ProviderInfo{
				Name:   p.Name(),
				Type:   p.Type(),
				Label:  typeLabel(p.Type()),
				Anon:   p.SupportsAnonymous(),
				Remote: p.SupportsRemoteURL(),
				HasAPI: p.HasAPI(),
			})
		}
	}
	return list
}
