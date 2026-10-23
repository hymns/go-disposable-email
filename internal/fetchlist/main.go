// Command fetchlist refreshes the embedded list: it downloads the upstream
// list, refuses it if it fails disposable.Validate, and writes it with a
// header naming the upstream commit. Run it with go generate from the
// module root.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	disposable "github.com/hymns/go-disposable-email"
)

const commitURL = "https://api.github.com/repos/disposable-email-domains/disposable-email-domains/commits/main"

func main() {
	out := flag.String("o", "domains.txt", "file to write")
	flag.Parse()
	log.SetFlags(0)

	body, err := get(disposable.DefaultURL, "")
	if err != nil {
		log.Fatal(err)
	}
	n, err := disposable.Validate(body)
	if err != nil {
		log.Fatalf("upstream list refused: %v", err)
	}
	commit := "unknown"
	if sha, err := get(commitURL, "application/vnd.github.sha"); err == nil && len(sha) >= 7 {
		commit = sha[:7]
	}
	header := fmt.Sprintf("# disposable-email-domains/disposable-email-domains, CC0 1.0, commit %s\n"+
		"# (disposable_email_blocklist.conf). Refresh: go generate ./...\n", commit)
	if err := os.WriteFile(*out, []byte(header+strings.TrimLeft(body, "\n")), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("%s: %d domains, commit %s", *out, n, commit)
}

func get(url, accept string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, disposable.MaxListBytes))
	return string(raw), err
}
