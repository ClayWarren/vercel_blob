# Go Vercel Blob Client

A lightweight, idiomatic Go client for the [Vercel Blob Storage API](https://vercel.com/docs/storage/vercel-blob). Inspired by the official TypeScript SDK, it provides a simple way to interact with your Vercel Blob store from Go applications.

## Installation

```bash
go get github.com/claywarren/vercel_blob
```

## Quick Start

### Within a Vercel Function

If your code is running in a Vercel Serverless or Edge function, authentication is handled automatically via the `BLOB_READ_WRITE_TOKEN` environment variable.

```go
import "github.com/claywarren/vercel_blob"

func main() {
    client := vercelblob.NewClient()
    
    // Use the client...
}
```

### Outside of Vercel (Client-side / External)

For external applications, you should use a `TokenProvider` to securely fetch short-lived tokens from your backend.

```go
import "github.com/claywarren/vercel_blob"

func main() {
    // Custom provider that fetches a token from your API
    provider := &MyTokenProvider{}
    client := vercelblob.NewClientExternal(provider)
    
    // Use the client...
}
```

## Operations

### List Blobs

```go
options := vercelblob.ListCommandOptions{
    Limit: 10,
    Prefix: "images/",
}

result, err := client.List(options)
if err != nil {
    log.Fatal(err)
}

for _, blob := range result.Blobs {
    fmt.Printf("Found blob: %s (%d bytes)\n", blob.PathName, blob.Size)
}
```

### Upload a Blob (Put)

```go
file, _ := os.Open("photo.jpg")
defile.Close()

options := vercelblob.PutCommandOptions{
    AddRandomSuffix: true,
    ContentType:     "image/jpeg",
}

result, err := client.Put("uploads/photo.jpg", file, options)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Uploaded to: %s\n", result.URL)
```

### Copy a Blob

```go
result, err := client.Copy(
    "https://your-store.public.blob.vercel-storage.com/old.txt",
    "new-destination.txt",
    vercelblob.PutCommandOptions{},
)
```

### Download a Blob

```go
// Download entire file
data, err := client.Download("uploads/photo.jpg", vercelblob.DownloadCommandOptions{})

// Download a specific range
rangeOptions := vercelblob.DownloadCommandOptions{
    ByteRange: &vercelblob.Range{Start: 0, End: 1024},
}
partialData, err := client.Download("uploads/large-file.bin", rangeOptions)
```

### Delete a Blob

```go
err := client.Delete("https://your-store.public.blob.vercel-storage.com/file-to-delete.txt")
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BLOB_READ_WRITE_TOKEN` | Your Vercel Blob read/write token (required if no provider used). |
| `VERCEL_BLOB_API_URL` | Override the default API endpoint (useful for testing). |
| `VERCEL_BLOB_API_VERSION` | Override the default API version (default: `9`). |

## License

MIT