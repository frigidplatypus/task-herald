package app

import (
    "io/ioutil"
    "os"
    "strings"

    "task-herald/internal/config"
)

// resolveHTTPAuthToken reads the auth token from the configured file if present.
// If cfg.HTTP.AuthToken is non-empty it is returned unchanged. If AuthTokenFile
// is set, the file is read, trimmed and returned. Errors are propagated.
func resolveHTTPAuthToken(cfg *config.Config) (string, error) {
    if cfg == nil {
        return "", nil
    }
    if cfg.HTTP.AuthToken != "" {
        return cfg.HTTP.AuthToken, nil
    }
    if cfg.HTTP.AuthTokenFile == "" {
        return "", nil
    }
    b, err := ioutil.ReadFile(cfg.HTTP.AuthTokenFile)
    if err != nil {
        return "", err
    }
    token := strings.TrimSpace(string(b))
    return token, nil
}

// resolveTLSPaths returns cert and key paths. Prefer the explicit TLSCert/TLSKey
// fields; if not set, fall back to TLSCertFile/TLSKeyFile. If file fields are
// set, ensure they exist by returning the path (validation may be done by caller).
func resolveTLSPaths(cfg *config.Config) (string, string) {
    if cfg == nil {
        return "", ""
    }
    if cfg.HTTP.TLSCert != "" || cfg.HTTP.TLSKey != "" {
        return cfg.HTTP.TLSCert, cfg.HTTP.TLSKey
    }
    // fall back to file-based fields
    cert := cfg.HTTP.TLSCertFile
    key := cfg.HTTP.TLSKeyFile
    // if files are specified but don't exist, return empty to allow caller
    // to decide; do a quick existence check
    if cert != "" {
        if _, err := os.Stat(cert); os.IsNotExist(err) {
            cert = ""
        }
    }
    if key != "" {
        if _, err := os.Stat(key); os.IsNotExist(err) {
            key = ""
        }
    }
    return cert, key
}
