package authorization

type Option func(*Manager)

type Manager struct {
	config *Config
}

func NewManager(opts ...Option) *Manager {
	manager := &Manager{
		config: &Config{},
	}
	for _, opt := range opts {
		opt(manager)
	}
	return manager
}

func (m *Manager) Set(opts ...Option) {
	for _, opt := range opts {
		opt(m)
	}
}

func (m *Manager) Init() error {
	return nil
}
