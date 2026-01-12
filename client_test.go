package vercelblob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var hasToken = os.Getenv("BLOB_READ_WRITE_TOKEN") != ""

func Test_List_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected Method GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Auth header Bearer test-token, got %s", r.Header.Get("Authorization"))
		}

		resp := ListBlobResult{
			Blobs: []ListBlobResultBlob{
				{URL: "https://blob.com/1.txt", PathName: "1.txt", Size: 100},
			},
			HasMore: false,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	_ = os.Setenv("BLOB_READ_WRITE_TOKEN", "test-token")
	defer func() { _ = os.Unsetenv("BLOB_READ_WRITE_TOKEN") }()

	client := NewClient()
	client.baseURL = server.URL // Override for mock server

	res, err := client.List(context.Background(), ListCommandOptions{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(res.Blobs) != 1 {
		t.Errorf("Expected 1 blob, got %d", len(res.Blobs))
	}
	if res.Blobs[0].PathName != "1.txt" {
		t.Errorf("Expected 1.txt, got %s", res.Blobs[0].PathName)
	}
}

func Test_Put_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected Method PUT, got %s", r.Method)
		}
		if r.Header.Get("X-Access") != "public" {
			t.Errorf("Expected X-Access public, got %s", r.Header.Get("X-Access"))
		}
		resp := PutBlobPutResult{URL: "https://blob.com/test.txt", Pathname: "test.txt"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	_ = os.Setenv("BLOB_READ_WRITE_TOKEN", "test")

	res, err := client.Put(context.Background(), "test.txt", bytes.NewReader([]byte("hello")), PutCommandOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "https://blob.com/test.txt" {
		t.Errorf("Expected URL https://blob.com/test.txt, got %s", res.URL)
	}
}

func Test_Delete_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected Method POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL
	_ = os.Setenv("BLOB_READ_WRITE_TOKEN", "test")

	err := client.Delete(context.Background(), "https://blob.com/1.txt", "https://blob.com/2.txt")
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Download_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))
	defer server.Close()

	client := NewClient()
	_ = os.Setenv("BLOB_READ_WRITE_TOKEN", "test")

	data, err := client.Download(context.Background(), server.URL, DownloadCommandOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("Expected hello world, got %s", string(data))
	}
}

func Test_CountFiles(t *testing.T) {
	if !hasToken {
		t.Skip("Skipping test: BLOB_READ_WRITE_TOKEN not set")
	}
	client := NewClient()
	allFiles, err := client.List(context.Background(), ListCommandOptions{})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(len(allFiles.Blobs))
}

func Test_PutWithRandomSuffix(t *testing.T) {
	if !hasToken {
		t.Skip("Skipping test: BLOB_READ_WRITE_TOKEN not set")
	}
	client := NewClient()
	f, _ := os.Open("a.png")
	defer func() { _ = f.Close() }()
	file1, err := client.Put(
		context.Background(),
		"vercel_blob_unittest/a.png",
		io.Reader(f),
		PutCommandOptions{
			AddRandomSuffix: true,
			//ContentType:     "multipart/form-data",
		})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(file1.URL)
}

func Test_Copy(t *testing.T) {
	if !hasToken {
		t.Skip("Skipping test: BLOB_READ_WRITE_TOKEN not set")
	}
	//https://fetegzn4vw3t5yqf.public.blob.vercel-storage.com/vercel_blob_unittest/a.txt
	client := NewClient()
	res, err := client.Copy(context.Background(),
		"https://fetegzn4vw3t5yqf.public.blob.vercel-storage.com/vercel_blob_unittest/a.txt",
		"vercel_blob_unittest/B.txt",
		PutCommandOptions{})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(res.URL)
}

func Test_Partial_Download(t *testing.T) {
	if !hasToken {
		t.Skip("Skipping test: BLOB_READ_WRITE_TOKEN not set")
	}
	client := NewClient()
	bytes, err := client.Download(context.Background(),
		"vercel_blob_unittest/a.txt",
		DownloadCommandOptions{
			ByteRange: &Range{Start: 0, End: 4},
		})
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(bytes))
}
