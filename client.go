package vercelblob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// BlobAPIVersion is the version of the Vercel Blob API.
const (
	BlobAPIVersion = "9"
	DefaultBaseURL = "https://blob.vercel-storage.com"
)

// Client is a client for the Vercel Blob Storage API.
type Client struct {
	tokenProvider TokenProvider
	baseURL       string
	apiVersion    string
	httpClient    *http.Client
}

// BlobAPIErrorDetail contains details about a blob API error.
type BlobAPIErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BlobAPIError is the error response from the Vercel Blob API.
type BlobAPIError struct {
	Error BlobAPIErrorDetail `json:"error"`
}

// NewClient creates a new client for use inside a Vercel function.
func NewClient() *Client {
	return &Client{
		baseURL:    getEnv("VERCEL_BLOB_API_URL", getEnv("NEXT_PUBLIC_VERCEL_BLOB_API_URL", DefaultBaseURL)),
		apiVersion: getEnv("VERCEL_BLOB_API_VERSION", BlobAPIVersion),
		httpClient: &http.Client{},
	}
}

// NewClientExternal creates a new client for use outside of Vercel.
func NewClientExternal(tokenProvider TokenProvider) *Client {
	return &Client{
		tokenProvider: tokenProvider,
		baseURL:       getEnv("VERCEL_BLOB_API_URL", getEnv("NEXT_PUBLIC_VERCEL_BLOB_API_URL", DefaultBaseURL)),
		apiVersion:    getEnv("VERCEL_BLOB_API_VERSION", BlobAPIVersion),
		httpClient:    &http.Client{},
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Client) getAPIURL(pathname string) string {
	base, _ := url.Parse(c.baseURL)
	base.Path = pathname
	return base.String()
}

func (c *Client) addAPIVersionHeader(req *http.Request) {
	req.Header.Set("x-api-version", c.apiVersion)
}

func (c *Client) addAuthorizationHeader(req *http.Request, operation, pathname string) error {
	var token string
	if c.tokenProvider != nil {
		token, _ = c.tokenProvider.GetToken(operation, pathname)
	} else {
		token = os.Getenv("BLOB_READ_WRITE_TOKEN")
	}

	if token == "" {
		return ErrNotAuthenticated
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (c *Client) handleError(resp *http.Response) error {
	if resp.StatusCode >= 500 {
		return NewUnknownError(resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var errResp BlobAPIError
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return err
	}

	switch errResp.Error.Code {
	case "store_suspended":
		return ErrStoreSuspended
	case "forbidden":
		return ErrForbidden
	case "not_found":
		return ErrBlobNotFound
	case "store_not_found":
		return ErrStoreNotFound
	case "bad_request":
		return ErrBadRequest(errResp.Error.Message)
	default:
		return NewUnknownError(resp.StatusCode, errResp.Error.Message)
	}
}

// List files in the blob store.
func (c *Client) List(ctx context.Context, options ListCommandOptions) (*ListBlobResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if options.Limit > 0 {
		q.Add("limit", strconv.FormatUint(options.Limit, 10))
	}
	if options.Prefix != "" {
		q.Add("prefix", options.Prefix)
	}
	if options.Cursor != "" {
		q.Add("cursor", options.Cursor)
	}
	if options.Mode != "" {
		q.Add("mode", options.Mode)
	}
	req.URL.RawQuery = q.Encode()

	c.addAPIVersionHeader(req)
	err = c.addAuthorizationHeader(req, "list", "")
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result ListBlobResult
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Put uploads a file to the blob store.
func (c *Client) Put(ctx context.Context, pathname string, body io.Reader, options PutCommandOptions) (*PutBlobPutResult, error) {
	if len(pathname) == 0 {
		return nil, NewInvalidInputError("pathname")
	}

	// Determine if we should use multipart
	var size int64 = -1
	if sizer, ok := body.(interface{ Size() int64 }); ok {
		size = sizer.Size()
	} else if seeker, ok := body.(io.Seeker); ok {
		curr, _ := seeker.Seek(0, io.SeekCurrent)
		size, _ = seeker.Seek(0, io.SeekEnd)
		_, _ = seeker.Seek(curr, io.SeekStart)
	}

	if size > MultipartThreshold {
		return c.putMultipart(ctx, pathname, body, options)
	}

	apiURL := c.getAPIURL(pathname)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, body)
	if err != nil {
		return nil, err
	}

	c.addAPIVersionHeader(req)
	err = c.addAuthorizationHeader(req, "put", pathname)
	if err != nil {
		return nil, err
	}

	c.setPutHeaders(req, options)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result PutBlobPutResult
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) setPutHeaders(req *http.Request, options PutCommandOptions) {
	if !options.AddRandomSuffix {
		req.Header.Set("X-Add-Random-Suffix", "0")
	}
	if options.ContentType != "" {
		req.Header.Set("X-Content-Type", options.ContentType)
	}
	if options.CacheControlMaxAge > 0 {
		req.Header.Set("X-Cache-Control-Max-Age", strconv.FormatUint(options.CacheControlMaxAge, 10))
	}
	access := options.Access
	if access == "" {
		access = "public"
	}
	req.Header.Set("X-Access", access)
}

// Head gets the metadata for a file in the blob store.
func (c *Client) Head(ctx context.Context, pathname string) (*HeadBlobResult, error) {
	apiURL := c.getAPIURL(pathname)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	c.addAPIVersionHeader(req)
	_ = c.addAuthorizationHeader(req, "put", pathname)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrBlobNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result HeadBlobResult
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

type deleteRequest struct {
	URLs []string `json:"urls"`
}

// Delete blobs from the blob store.
func (c *Client) Delete(ctx context.Context, urls ...string) error {
	if len(urls) == 0 {
		return nil
	}
	apiURL := c.getAPIURL("/delete")
	reqBody, _ := json.Marshal(deleteRequest{URLs: urls})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.addAPIVersionHeader(req)
	_ = c.addAuthorizationHeader(req, "delete", urls[0])

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}
	return nil
}

// Copy copies an existing blob object to a new path within the blob store.
func (c *Client) Copy(ctx context.Context, fromURL, toPath string, options PutCommandOptions) (*PutBlobPutResult, error) {
	if len(fromURL) == 0 {
		return nil, NewInvalidInputError("fromURL")
	}
	if len(toPath) == 0 {
		return nil, NewInvalidInputError("toPath")
	}
	apiURL := c.getAPIURL(toPath)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, nil)
	q := req.URL.Query()
	q.Add("fromUrl", fromURL)
	req.URL.RawQuery = q.Encode()

	c.addAPIVersionHeader(req)
	_ = c.addAuthorizationHeader(req, "put", toPath)
	c.setPutHeaders(req, options)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}
	var result PutBlobPutResult
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

// Download a blob from the blob store.
func (c *Client) Download(ctx context.Context, urlPath string, options DownloadCommandOptions) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlPath, nil)
	c.addAPIVersionHeader(req)
	_ = c.addAuthorizationHeader(req, "download", urlPath)

	if options.ByteRange != nil {
		req.Header.Set("range", fmt.Sprintf("bytes=%d-%d", options.ByteRange.Start, options.ByteRange.End))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, c.handleError(resp)
	}
	return io.ReadAll(resp.Body)
}
