package app

import (
    "io/ioutil"
    "os"
    "path/filepath"
    "testing"

    "task-herald/internal/config"
)

func TestResolveHTTPAuthToken_FromFile(t *testing.T) {
    dir, err := ioutil.TempDir("", "authfiletest")
    if err != nil {
        t.Fatalf("tempdir: %v", err)
    }
    defer os.RemoveAll(dir)

    f := filepath.Join(dir, "token.txt")
    if err := ioutil.WriteFile(f, []byte("  s3cr3t-token\n"), 0600); err != nil {
        t.Fatalf("writefile: %v", err)
    }

    cfg := &config.Config{}
    cfg.HTTP.AuthTokenFile = f

    tok, err := resolveHTTPAuthToken(cfg)
    if err != nil {
        t.Fatalf("resolveHTTPAuthToken: %v", err)
    }
    if tok != "s3cr3t-token" {
        t.Fatalf("unexpected token: %q", tok)
    }
}

func TestResolveHTTPAuthToken_PreferInline(t *testing.T) {
    cfg := &config.Config{}
    cfg.HTTP.AuthToken = "inline"
    cfg.HTTP.AuthTokenFile = "/no/such/file"

    tok, err := resolveHTTPAuthToken(cfg)
    if err != nil {
        t.Fatalf("resolveHTTPAuthToken: %v", err)
    }
    if tok != "inline" {
        t.Fatalf("expected inline, got: %q", tok)
    }
}

func TestResolveTLSPaths_FileFallback(t *testing.T) {
    dir, err := ioutil.TempDir("", "tlsfiletest")
    if err != nil {
        t.Fatalf("tempdir: %v", err)
    }
    defer os.RemoveAll(dir)

    cert := filepath.Join(dir, "cert.pem")
    key := filepath.Join(dir, "key.pem")
    if err := ioutil.WriteFile(cert, []byte("cert"), 0600); err != nil {
        t.Fatalf("write cert: %v", err)
    }
    if err := ioutil.WriteFile(key, []byte("key"), 0600); err != nil {
        t.Fatalf("write key: %v", err)
    }

    cfg := &config.Config{}
    cfg.HTTP.TLSCertFile = cert
    cfg.HTTP.TLSKeyFile = key

    c, k := resolveTLSPaths(cfg)
    if c != cert || k != key {
        t.Fatalf("unexpected tls paths: %q %q", c, k)
    }
}
