package crawler

import (
	"testing"
	"time"
)

func TestNewCrawlerClientConfig(t *testing.T) {
	client := NewCrawlerClient(3 * time.Second)

	if client.Timeout != 3*time.Second {
		t.Fatalf("unexpected timeout: %v", client.Timeout)
	}
	if client.Jar == nil {
		t.Fatal("expected cookie jar")
	}
	if client.CheckRedirect == nil {
		t.Fatal("expected redirect policy")
	}
}
