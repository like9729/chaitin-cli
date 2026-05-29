package cert

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	safelinecmd "github.com/chaitin/chaitin-cli/products/safeline/cmd"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	c := &cobra.Command{Use: "cert", Short: "SSL certificate management commands"}
	c.AddCommand(newListCmd(), newGetCmd(), newUploadCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List SSL certificates", RunE: func(c *cobra.Command, args []string) error {
		env, err := safelinecmd.NewClient().Do("GET", "/api/CertAPI", nil, nil)
		if err != nil {
			return err
		}
		return safelinecmd.PrintEnvelope(c, env)
	}}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get SSL certificate details", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		env, err := safelinecmd.NewClient().Do("GET", "/api/CertAPI", nil, map[string]string{"id": args[0]})
		if err != nil {
			return err
		}
		return safelinecmd.PrintEnvelope(c, env)
	}}
}

func newUploadCmd() *cobra.Command {
	var name, crtPath, keyPath, password string
	cmd := &cobra.Command{Use: "upload", Short: "Upload an ordinary SSL certificate and private key", RunE: func(c *cobra.Command, args []string) error {
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if crtPath == "" {
			return fmt.Errorf("--crt is required")
		}
		if keyPath == "" {
			return fmt.Errorf("--key is required")
		}
		body := &bytes.Buffer{}
		mw := multipart.NewWriter(body)
		_ = mw.WriteField("name", name)
		_ = mw.WriteField("password", password)
		if err := addFile(mw, "crt_file", crtPath); err != nil {
			return err
		}
		if err := addFile(mw, "key_file", keyPath); err != nil {
			return err
		}
		if err := mw.Close(); err != nil {
			return err
		}

		if safelinecmd.IsDryRun() {
			fmt.Fprintf(c.ErrOrStderr(), "[DRY-RUN] POST /api/UploadSSLCertAPI\n")
			fmt.Fprintf(c.ErrOrStderr(), "Fields: name=%s crt_file=%s key_file=%s\n", name, crtPath, keyPath)
			return nil
		}
		env, err := safelinecmd.NewClient().DoRaw("POST", "/api/UploadSSLCertAPI", body, mw.FormDataContentType(), nil)
		if err != nil {
			return err
		}
		return safelinecmd.PrintEnvelope(c, env)
	}}
	cmd.Flags().StringVar(&name, "name", "", "Certificate name")
	cmd.Flags().StringVar(&crtPath, "crt", "", "Certificate file path")
	cmd.Flags().StringVar(&keyPath, "key", "", "Private key file path")
	cmd.Flags().StringVar(&password, "password", "", "Private key password, if required")
	return cmd
}

func addFile(mw *multipart.Writer, field, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	part, err := mw.CreateFormFile(field, filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, f)
	return err
}
