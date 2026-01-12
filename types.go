package vercelblob

import (
	"time"
)

// ListBlobResultBlob contains details about a blob returned by the list operation.
type ListBlobResultBlob struct {
	URL        string    `json:"url"`
	PathName   string    `json:"pathname"`
	Size       uint64    `json:"size"`
	UploadedAt time.Time `json:"uploadedAt"`
}

// ListBlobResult is the response from the list operation.
type ListBlobResult struct {
	Blobs   []ListBlobResultBlob `json:"blobs"`
	Folders []string             `json:"folders,omitempty"`
	Cursor  string               `json:"cursor"`
	HasMore bool                 `json:"hasMore"`
}

// ListCommandOptions contains options for the list operation.
type ListCommandOptions struct {
	Limit  uint64
	Prefix string
	Cursor string
	// Mode for the list operation: "expanded" (default) or "folded"
	Mode string
}

// PutCommandOptions contains options for the put operation.
type PutCommandOptions struct {
	AddRandomSuffix    bool
	CacheControlMaxAge uint64
	ContentType        string
	// Access for the blob: "public" (default)
	Access string
}

// PutBlobPutResult is the response from the put operation.
type PutBlobPutResult struct {
	URL                string `json:"url"`
	Pathname           string `json:"pathname"`
	ContentType        string `json:"contentType"`
	ContentDisposition string `json:"contentDisposition"`
}

// HeadBlobResult is the response from the head operation.
type HeadBlobResult struct {
	URL                string    `json:"url"`
	Size               uint64    `json:"size"`
	UploadedAt         time.Time `json:"uploadedAt"`
	Pathname           string    `json:"pathname"`
	ContentType        string    `json:"contentType"`
	ContentDisposition string    `json:"contentDisposition"`
	CacheControl       string    `json:"cacheControl"`
}

// Range represents a byte range for download operations.
type Range struct {
	Start uint
	End   uint
}

// DownloadCommandOptions contains options for the download operation.
type DownloadCommandOptions struct {
	// The range of bytes to download.
	ByteRange *Range
}
