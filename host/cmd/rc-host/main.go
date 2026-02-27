package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ericbosch/cli-remote-control/host/internal/server"
	"github.com/spf13/cobra"
)

func defaultTokenFilePath() string {
	// Preferred location is <repo>/host/.dev-token regardless of current working directory.
	// Try to locate the repo root by walking up to find a `.git` directory.
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Join("host", ".dev-token")
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && st.IsDir() {
			return filepath.Join(dir, "host", ".dev-token")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: best-effort relative path.
	return filepath.Join("host", ".dev-token")
}

func normalizeTokenValue(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "Bearer ")
	v = strings.TrimPrefix(v, "bearer ")
	return strings.TrimSpace(v)
}

func tokenSHA256Hex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func readTokenFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func main() {
	root := &cobra.Command{
		Use: "rc-host",
	}
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the remote control host daemon",
		RunE:  runServe,
	}
	serveCmd.Flags().String("bind", "127.0.0.1", "Bind address (use 0.0.0.0 to expose; triggers security warning)")
	serveCmd.Flags().String("port", "8787", "Port")
	serveCmd.Flags().String("token", "", "Bearer token for API/WS auth (overrides env RC_TOKEN)")
	serveCmd.Flags().String("token-file", "", "Path to token file (overrides env RC_TOKEN_FILE). Used for --generate-dev-token and for loading an existing token.")
	serveCmd.Flags().String("log-dir", "logs", "Directory for session logs (rotated)")
	serveCmd.Flags().Bool("generate-dev-token", false, "Generate and write dev token to .dev-token if no token set")
	serveCmd.Flags().String("web-dir", "", "Serve static web from this directory at / (empty = no static)")
	root.AddCommand(serveCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServe(cmd *cobra.Command, _ []string) error {
	bind, _ := cmd.Flags().GetString("bind")
	port, _ := cmd.Flags().GetString("port")
	token, _ := cmd.Flags().GetString("token")
	tokenFile, _ := cmd.Flags().GetString("token-file")
	logDir, _ := cmd.Flags().GetString("log-dir")
	generateDevToken, _ := cmd.Flags().GetBool("generate-dev-token")
	webDir, _ := cmd.Flags().GetString("web-dir")

	if token == "" {
		token = os.Getenv("RC_TOKEN")
	}
	token = normalizeTokenValue(token)
	if tokenFile == "" {
		tokenFile = os.Getenv("RC_TOKEN_FILE")
	}
	if tokenFile == "" {
		tokenFile = defaultTokenFilePath()
	}

	// If no token is set explicitly, try to load it from tokenFile (if it exists),
	// or generate it if requested.
	if token == "" {
		if t, err := readTokenFile(tokenFile); err == nil {
			token = normalizeTokenValue(t)
			if token != "" {
				log.Printf("Auth token loaded: file=%s len=%d sha256=%s", tokenFile, len(token), tokenSHA256Hex(token))
			}
		}
	}
	if token == "" && generateDevToken {
		devToken, err := server.GenerateAndWriteDevToken(tokenFile)
		if err != nil {
			log.Printf("Warning: could not write dev token: %v", err)
		} else {
			token = normalizeTokenValue(devToken)
			log.Printf("Dev token written: file=%s len=%d sha256=%s (use as Bearer token; do not expose)", tokenFile, len(token), tokenSHA256Hex(token))
		}
	}
	if token == "" {
		log.Fatal("No auth token set. Use --token/RC_TOKEN, --token-file/RC_TOKEN_FILE, or --generate-dev-token.")
	}

	// Policy: do not use CURSOR_API_KEY (subscription login only).
	if os.Getenv("CURSOR_API_KEY") != "" {
		log.Printf("Warning: CURSOR_API_KEY is set but API keys are disabled by policy; ignoring it.")
		os.Unsetenv("CURSOR_API_KEY")
	}

	if bind == "0.0.0.0" {
		log.Printf("WARNING: Binding to 0.0.0.0 — service is exposed to the network. Use only on trusted LAN or VPN.")
	}

	cfg := server.Config{
		Bind:   bind,
		Port:   port,
		Token:  token,
		LogDir: logDir,
		WebDir: webDir,
	}
	srv, err := server.New(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	return srv.Run(ctx)
}
