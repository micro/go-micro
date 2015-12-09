package registry

type mockRegistry struct{}

func (m *mockRegistry) GetService(service string) ([]*Service, error) {
	return []*Service{
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*Node{
				{
					Id:      "foo-1.0.0-123",
					Address: "localhost",
					Port:    9999,
				},
				{
					Id:      "foo-1.0.0-321",
					Address: "localhost",
					Port:    9999,
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.1",
			Nodes: []*Node{
				{
					Id:      "foo-1.0.1-321",
					Address: "localhost",
					Port:    6666,
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.3",
			Nodes: []*Node{
				{
					Id:      "foo-1.0.3-345",
					Address: "localhost",
					Port:    8888,
				},
			},
		},
	}, nil
}

func (m *mockRegistry) ListServices() ([]*Service, error) {
	return []*Service{}, nil
}

func (m *mockRegistry) Register(s *Service) error {
	return nil
}

func (m *mockRegistry) Deregister(s *Service) error {
	return nil
}

func (m *mockRegistry) Watch() (Watcher, error) {
	return nil, nil
}
