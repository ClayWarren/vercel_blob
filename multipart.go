package vercelblob

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// MultipartThreshold is the minimum size for multipart uploads (5MB).
const MultipartThreshold = 5 * 1024 * 1024

type createMultipartUploadResponse struct {
	UploadID string `json:"uploadId"`
	Key      string `json:"key"`
}

// Part represents a part of a multipart upload.
type Part struct {
	ETag       string `json:"etag"`
	PartNumber int    `json:"partNumber"`
}

type completeMultipartUploadRequest struct {
	UploadID string `json:"uploadId"`
	Key      string `json:"key"`
	Parts    []Part `json:"parts"`
}

func (c *Client) putMultipart(ctx context.Context, pathname string, body io.Reader, options PutCommandOptions) (*PutBlobPutResult, error) {
	// 1. Create Multipart Upload
	apiURL := c.getAPIURL("/mpu")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return nil, err
	}
	c.addAPIVersionHeader(req)
	_ = c.addAuthorizationHeader(req, "put", pathname)
	c.setPutHeaders(req, options)
	req.Header.Set("X-MPU-Action", "create")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}
	var createResp createMultipartUploadResponse
	_ = json.NewDecoder(resp.Body).Decode(&createResp)
	_ = resp.Body.Close()

	// 2. Upload Parts
	var parts []Part
	partNumber := 1
	buffer := make([]byte, MultipartThreshold)
	for {
		n, err := io.ReadFull(body, buffer)
		if n > 0 {
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewReader(buffer[:n]))
			if err != nil {
				return nil, err
			}
			c.addAPIVersionHeader(req)
			_ = c.addAuthorizationHeader(req, "put", pathname)
			req.Header.Set("X-MPU-Action", "upload")
			req.Header.Set("X-MPU-Upload-Id", createResp.UploadID)
			req.Header.Set("X-MPU-Key", createResp.Key)
			req.Header.Set("X-MPU-Part-Number", strconv.Itoa(partNumber))

			resp, err := c.httpClient.Do(req)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode != http.StatusOK {
				return nil, c.handleError(resp)
			}
			etag := resp.Header.Get("ETag")
			_ = resp.Body.Close()

			parts = append(parts, Part{ETag: etag, PartNumber: partNumber})
			partNumber++
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	// 3. Complete
	completeReq, _ := json.Marshal(completeMultipartUploadRequest{
		UploadID: createResp.UploadID,
		Key:      createResp.Key,
		Parts:    parts,
	})
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(completeReq))
	c.addAPIVersionHeader(req)
	_ = c.addAuthorizationHeader(req, "put", pathname)
	req.Header.Set("X-MPU-Action", "complete")

	resp, err = c.httpClient.Do(req)
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
