package clients

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"

	providerv1alpha1 "github.com/maximilianbraun/provider-matomo/apis/v1alpha1"
)

// Client talks to the Matomo Reporting API over HTTP.
type Client struct {
	baseURL    string
	tokenAuth  string
	httpClient *http.Client
}

// Call invokes a Matomo API method (e.g. "SitesManager.addSite") with the
// given parameters and returns the raw JSON response. module=API,
// format=JSON and token_auth are added automatically. Params are sent as a
// POST body so the token never lands in a URL or server log.
func (c *Client) Call(ctx context.Context, method string, params url.Values) (json.RawMessage, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("module", "API")
	params.Set("method", method)
	params.Set("format", "JSON")
	params.Set("token_auth", c.tokenAuth)

	endpoint := strings.TrimRight(c.baseURL, "/") + "/index.php"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling %s: %w", method, err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response body

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s response: %w", method, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned HTTP %d: %s", method, resp.StatusCode, string(body))
	}
	return parseResponse(body)
}

// parseResponse detects Matomo's JSON error envelope and otherwise returns
// the body verbatim. Matomo reports errors as {"result":"error","message":...}
// even on HTTP 200, so the envelope must be inspected explicitly.
func parseResponse(body []byte) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var env struct {
			Result  string `json:"result"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(trimmed, &env); err == nil && env.Result == "error" {
			return nil, fmt.Errorf("matomo API error: %s", env.Message)
		}
	}
	return json.RawMessage(trimmed), nil
}

// Cache is a thread-safe cache of Matomo clients keyed by ProviderConfig identity.
type Cache struct {
	mu      sync.Mutex
	clients map[string]*Client
}

// NewCache creates a new client cache.
func NewCache() *Cache {
	return &Cache{clients: make(map[string]*Client)}
}

func cacheKey(pc *providerv1alpha1.ProviderConfig) string {
	return fmt.Sprintf("%s@%s", pc.Name, pc.ResourceVersion)
}

// GetOrBuild returns a cached client or builds a new one for the ProviderConfig.
func (c *Cache) GetOrBuild(ctx context.Context, kube client.Client, pc *providerv1alpha1.ProviderConfig) (*Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(pc)
	if cl, ok := c.clients[key]; ok {
		return cl, nil
	}
	c.evictLocked(pc.Name)

	cl, err := newMatomoClient(ctx, kube, pc)
	if err != nil {
		return nil, err
	}
	c.clients[key] = cl
	return cl, nil
}

// Evict removes all cached clients for a given ProviderConfig name.
func (c *Cache) Evict(pcName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictLocked(pcName)
}

func (c *Cache) evictLocked(pcName string) {
	prefix := pcName + "@"
	for k := range c.clients {
		if strings.HasPrefix(k, prefix) {
			delete(c.clients, k)
		}
	}
}

func newMatomoClient(ctx context.Context, kube client.Client, pc *providerv1alpha1.ProviderConfig) (*Client, error) {
	token, err := readSecretValue(ctx, kube, pc.Spec.Credentials.TokenAuthSecretRef)
	if err != nil {
		return nil, fmt.Errorf("reading token_auth secret: %w", err)
	}

	tlsConfig, err := buildTLSConfig(ctx, kube, pc)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:    pc.Spec.URL,
		tokenAuth:  token,
		httpClient: &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}},
	}, nil
}

func buildTLSConfig(ctx context.Context, kube client.Client, pc *providerv1alpha1.ProviderConfig) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if pc.Spec.TLS == nil {
		return cfg, nil
	}
	cfg.InsecureSkipVerify = pc.Spec.TLS.InsecureSkipVerify //nolint:gosec // opt-in via ProviderConfig
	if pc.Spec.TLS.CASecretRef != nil {
		caPEM, err := readSecretValue(ctx, kube, *pc.Spec.TLS.CASecretRef)
		if err != nil {
			return nil, fmt.Errorf("reading CA secret: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caPEM)) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

func readSecretValue(ctx context.Context, kube client.Client, ref xpv1.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	nn := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
	if err := kube.Get(ctx, nn, secret); err != nil {
		return "", fmt.Errorf("getting secret %s: %w", nn, err)
	}
	val, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s", ref.Key, nn)
	}
	return string(val), nil
}
