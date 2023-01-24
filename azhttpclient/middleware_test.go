package azhttpclient

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
	"github.com/grafana/grafana-azure-sdk-go/aztokenprovider"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

func TestAzureMiddleware(t *testing.T) {
	azureSettings := &azsettings.AzureSettings{
		Cloud: azsettings.AzurePublic,
	}

	clientOpts := httpclient.Options{}
	next := &testRoundTripper{}

	t.Run("should use custom provider if registered for given custom credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})
		testTokenProvider := &customTokenProvider{}
		authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
			return testTokenProvider, nil
		})

		credentials := &customCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		resp, err := middleware.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.True(t, testTokenProvider.Called)
	})

	t.Run("should return error if custom provider not registered for given custom credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})

		credentials := &customCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		_, err = middleware.RoundTrip(req)
		assert.Error(t, err)
	})

	t.Run("should use custom provider if registered for built-in credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})
		testTokenProvider := &customTokenProvider{}
		authOpts.AddTokenProvider(azcredentials.AzureAuthManagedIdentity, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
			return testTokenProvider, nil
		})

		credentials := &azcredentials.AzureManagedIdentityCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		_, err = middleware.RoundTrip(req)
		require.NoError(t, err)
		assert.True(t, testTokenProvider.Called)
	})

	t.Run("should not use custom provider if registered for different credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})
		testTokenProvider := &customTokenProvider{}
		authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
			return testTokenProvider, nil
		})

		credentials := &azcredentials.AzureManagedIdentityCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		_, err = middleware.RoundTrip(req)
		assert.EqualError(t, err, "invalid Azure configuration: managed identity authentication is not enabled in Grafana config")
		assert.False(t, testTokenProvider.Called)
	})
}

const (
	azureAuthCustom = "custom"
)

type customCredentials struct {
}

func (credentials *customCredentials) AzureAuthType() string {
	return azureAuthCustom
}

type customTokenProvider struct {
	Called bool
}

func (provider *customTokenProvider) GetAccessToken(ctx context.Context, scopes []string) (string, error) {
	if ctx == nil {
		err := fmt.Errorf("parameter 'ctx' cannot be nil")
		return "", err
	}
	if scopes == nil {
		err := fmt.Errorf("parameter 'scopes' cannot be nil")
		return "", err
	}

	provider.Called = true

	return "FAKE-ACCESS-TOKEN", nil
}

type testRoundTripper struct {
}

func (rt *testRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{Status: "200 OK", StatusCode: 200}, nil
}
