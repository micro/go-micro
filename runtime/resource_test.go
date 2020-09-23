package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResources(t *testing.T) {

	// Namespace:
	namespace, err := NewNamespace("")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidResource, err)
	assert.Nil(t, namespace)

	namespace, err = NewNamespace("test-namespace")
	assert.NoError(t, err)
	assert.NotNil(t, namespace)
	assert.Equal(t, TypeNamespace, namespace.Type())
	assert.Equal(t, "test-namespace", namespace.ID())

	// NetworkPolicy:
	networkPolicy, err := NewNetworkPolicy("", "", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidResource, err)
	assert.Nil(t, networkPolicy)

	networkPolicy, err = NewNetworkPolicy("test", "", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidResource, err)
	assert.Nil(t, networkPolicy)

	networkPolicy, err = NewNetworkPolicy("", "test", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidResource, err)
	assert.Nil(t, networkPolicy)

	networkPolicy, err = NewNetworkPolicy("ingress", "test", nil)
	assert.NoError(t, err)
	assert.NotNil(t, networkPolicy)
	assert.Equal(t, TypeNetworkPolicy, networkPolicy.Type())
	assert.Equal(t, "test.ingress", networkPolicy.ID())
	assert.Len(t, networkPolicy.AllowedLabels, 1)

	networkPolicy, err = NewNetworkPolicy("ingress", "test", map[string]string{"foo": "bar", "bar": "foo"})
	assert.Len(t, networkPolicy.AllowedLabels, 2)

	// Service:
	service, err := NewService("")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidResource, err)
	assert.Nil(t, service)

	service, err = NewService("test-service")
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, TypeService, service.Type())
	assert.Equal(t, "test-service", service.ID())
}
