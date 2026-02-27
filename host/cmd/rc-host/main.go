package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ericbosch/cli-remote-control/host/internal/server"
	"github.com/spf13/cobra"
)

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
	serveCmd.Flags().String("port", "8765", "Port")
	serveCmd.Flags().String("token", "", "Bearer token for API/WS auth (overrides env RC_TOKEN)")
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
	logDir, _ := cmd.Flags().GetString("log-dir")
	generateDevToken, _ := cmd.Flags().GetBool("generate-dev-token")
	webDir, _ := cmd.Flags().GetString("web-dir")

	if token == "" {
		token = os.Getenv("RC_TOKEN")
	}
	if token == "" && generateDevToken {
		devToken, err := server.GenerateAndWriteDevToken("host/.dev-token")
		if err != nil {
			log.Printf("Warning: could not write dev token: %v", err)
		} else {
			token = devToken
			log.Printf("Dev token written to host/.dev-token — use as Bearer token. Do not expose.")
		}
	}
	if token == "" {
		log.Fatal("No auth token set. Use --token, RC_TOKEN, or --generate-dev-token.")
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
