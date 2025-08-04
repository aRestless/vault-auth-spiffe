package vault_auth_spiffe

import (
	"context"
	"testing"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/command/agentproxyshared/auth"
	"github.com/stretchr/testify/require"
)

func TestNewSPIFFEAuthMethod(t *testing.T) {
	t.Run("should return error if config is nil", func(t *testing.T) {
		m, err := NewSPIFFEAuthMethod(nil)
		require.Error(t, err)
		require.Nil(t, m)
		require.Equal(t, "empty config", err.Error())
	})

	t.Run("should return error if config data is nil", func(t *testing.T) {
		m, err := NewSPIFFEAuthMethod(&auth.AuthConfig{})
		require.Error(t, err)
		require.Nil(t, m)
		require.Equal(t, "empty config data", err.Error())
	})

	t.Run("should return error if role is missing", func(t *testing.T) {
		conf := &auth.AuthConfig{
			Config: map[string]interface{}{},
			Logger: hclog.NewNullLogger(),
		}
		m, err := NewSPIFFEAuthMethod(conf)
		require.Error(t, err)
		require.Nil(t, m)
		require.Equal(t, "missing 'role' value", err.Error())
	})

	t.Run("should return error if role is not a string", func(t *testing.T) {
		conf := &auth.AuthConfig{
			Config: map[string]interface{}{
				"role": 123,
			},
			Logger: hclog.NewNullLogger(),
		}
		m, err := NewSPIFFEAuthMethod(conf)
		require.Error(t, err)
		require.Nil(t, m)
		require.Equal(t, "could not convert 'role' config value to string", err.Error())
	})

	t.Run("should not return error if audience is missing", func(t *testing.T) {
		conf := &auth.AuthConfig{
			Config: map[string]interface{}{
				"role": "test-role",
			},
			Logger: hclog.NewNullLogger(),
		}
		m, err := NewSPIFFEAuthMethod(conf)
		require.NoError(t, err)
		require.NotNil(t, m)
		m.Shutdown()
	})

	t.Run("should return error if audience is not a string", func(t *testing.T) {
		conf := &auth.AuthConfig{
			Config: map[string]interface{}{
				"role":     "test-role",
				"audience": 123,
			},
			Logger: hclog.NewNullLogger(),
		}
		m, err := NewSPIFFEAuthMethod(conf)
		require.Error(t, err)
		require.Nil(t, m)
		require.Equal(t, "could not convert 'audience' config value to string", err.Error())
	})

	t.Run("should return error if role is empty", func(t *testing.T) {
		conf := &auth.AuthConfig{
			Config: map[string]interface{}{
				"role": "",
			},
			Logger: hclog.NewNullLogger(),
		}
		m, err := NewSPIFFEAuthMethod(conf)
		require.Error(t, err)
		require.Nil(t, m)
		require.Equal(t, "'role' value is empty", err.Error())
	})

	t.Run("should create a new spiffe auth method", func(t *testing.T) {
		conf := &auth.AuthConfig{
			MountPath: "spiffe",
			Config: map[string]interface{}{
				"role":     "test-role",
				"audience": "test-audience",
			},
			Logger: hclog.NewNullLogger(),
		}
		m, err := NewSPIFFEAuthMethod(conf)
		require.NoError(t, err)
		require.NotNil(t, m)

		spiffeMethod, ok := m.(*spiffeMethod)
		require.True(t, ok)
		require.Equal(t, "spiffe", spiffeMethod.mountPath)
		require.Equal(t, "test-role", spiffeMethod.role)
		require.Equal(t, "test-audience", spiffeMethod.audience)
		m.Shutdown()
	})
}

func TestAuthenticate(t *testing.T) {
	conf := &auth.AuthConfig{
		MountPath: "spiffe",
		Config: map[string]interface{}{
			"role":     "test-role",
			"audience": "test-audience",
		},
		Logger: hclog.NewNullLogger(),
	}
	m, err := NewSPIFFEAuthMethod(conf)
	require.NoError(t, err)
	require.NotNil(t, m)
	defer m.Shutdown()

	spiffeMethod, ok := m.(*spiffeMethod)
	require.True(t, ok)

	t.Run("should return error if token is not available", func(t *testing.T) {
		path, header, data, err := m.Authenticate(context.Background(), nil)
		require.Error(t, err)
		require.Empty(t, path)
		require.Nil(t, header)
		require.Nil(t, data)
		require.Equal(t, "spiffe jwt-svid is not available", err.Error())
	})

	t.Run("should return login details if token is available", func(t *testing.T) {
		spiffeMethod.latestToken.Store("test-token")
		path, header, data, err := m.Authenticate(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, "spiffe/login", path)
		require.Nil(t, header)
		require.Equal(t, map[string]interface{}{
			"role": "test-role",
			"jwt":  "test-token",
		}, data)
	})
}

func TestShutdown(t *testing.T) {
	conf := &auth.AuthConfig{
		Config: map[string]interface{}{
			"role": "test-role",
		},
		Logger: hclog.NewNullLogger(),
	}
	m, err := NewSPIFFEAuthMethod(conf)
	require.NoError(t, err)
	require.NotNil(t, m)

	spiffeMethod, ok := m.(*spiffeMethod)
	require.True(t, ok)

	m.Shutdown()

	select {
	case <-spiffeMethod.doneCh:
		// expected
	case <-time.After(1 * time.Second):
		t.Fatal("shutdown did not close doneCh")
	}
}
