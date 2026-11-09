package cli

import (
	"fmt"
	"net/http"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/server"
	"github.com/spf13/cobra"
)

func serveCmd(ctx *app.AppContext) *cobra.Command {
	var port int
	var root string
	var open bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve a read-only migration dashboard (slice 1)",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := server.New(root, ctx.Logger)
			if err != nil {
				return err
			}
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			ctx.Logger.Info("serving dashboard", "addr", addr, "root", root)
			if open {
				_ = openBrowser("http://" + addr)
			}
			return http.ListenAndServe(addr, srv.Handler())
		},
	}
	cmd.Flags().IntVar(&port, "port", 8080, "port to listen on (localhost only)")
	cmd.Flags().StringVar(&root, "projects-root", "projects", "directory scanned for projects; use . for a single project")
	cmd.Flags().BoolVar(&open, "open", false, "open the browser on start")
	return cmd
}

func wizardCmd(ctx *app.AppContext) *cobra.Command {
	c := serveCmd(ctx)
	c.Use = "wizard"
	c.Short = "Open the migration wizard (alias for serve --open)"
	c.RunE = func(cmd *cobra.Command, args []string) error {
		_ = cmd.Flags().Set("open", "true")
		return c.RunE(cmd, args)
	}
	return c
}
