package builtin

import (
	"app"
	"app/command"
	"app/fmtx"
	"app/style"
	"fmt"
	"os"
	"path/filepath"
	"shared/crypto"
	"strings"
	"time"
)

// MakeIdentityCommand returns a command handler for the identity command
func MakeIdentityCommand(app app.App) command.Handler {

	flags := []command.Argument{
		command.NewArgument("generate", "Generate app and server identity bundles", command.ArgTypeBool, false),
		command.NewArgument("output", "Output directory for identity bundles", command.ArgTypeDirectory, false, ".gonduit"),
		command.NewArgument("fingerprint", "Shows the fingerprint of an identity bundle", command.ArgTypeFile, false),
		command.NewArgument("info", "Shows information about the current identity", command.ArgTypeBool, false),
		command.NewArgument("self-signed", "Generates and uses a self-signed identity bundle", command.ArgTypeBool, false),
		command.NewArgument("load", "Load an identity bundle for use", command.ArgTypeFile, false),
		command.NewArgument("unload", "Unload the loaded identity bundle", command.ArgTypeBool, false),
	}

	return command.NewHandler("identity", wrappedExecutor(app, identity), "Manages certificates and identity bundles", nil, flags)

}

func identity(app app.App, ctx *command.Context) (string, error) {

	generate := ctx.BoolFlag("generate")
	output := ctx.Flag("output")
	load := ctx.Flag("load")
	unload := ctx.BoolFlag("unload")
	fingerprint := ctx.Flag("fingerprint")
	selfSigned := ctx.BoolFlag("self-signed")
	info := ctx.BoolFlag("info")

	if info {
		id, err := app.SessionManager().Identity()
		if err != nil {
			return "", fmt.Errorf("failed to get identity: %w", err)
		}

		selfSignedLabel := "CA-signed"
		if id.SelfSigned {
			selfSignedLabel = "Self-signed"
		}

		expiresIn := time.Until(id.NotAfter)
		expiryStatus := style.SuccessText.Render(fmt.Sprintf("valid for %d days", int(expiresIn.Hours()/24)))
		if expiresIn < 7*24*time.Hour {
			expiryStatus = style.ErrorText.Render(fmt.Sprintf("expires in %d days!", int(expiresIn.Hours()/24)))
		} else if expiresIn < 30*24*time.Hour {
			expiryStatus = style.WarningText.Render(fmt.Sprintf("expires soon (%d days)", int(expiresIn.Hours()/24)))
		}

		return fmtx.Infof(
			"Showing current identity information:\n\n"+
				"Common Name:   %s\n"+
				"Type:          %s\n"+
				"Issuer:        %s\n"+
				"SANs:          %s\n"+
				"Serial:        %s\n"+
				"Valid From:    %s\n"+
				"Valid Until:   %s (%s%s\n"+
				"Fingerprint:   %s",
			id.CommonName,
			selfSignedLabel,
			id.Issuer,
			strings.Join(id.SANs, ", "),
			id.SerialNumber,
			id.NotBefore.Format(time.RFC3339),
			id.NotAfter.Format(time.RFC3339),
			expiryStatus,
			style.InfoStyle.Render(")"), // lipgloss bug fix not coloring parenthesis after expiry render
			id.Fingerprint,
		), nil

	}

	if generate {

		if output == "" {
			output = ".gonduit"
		}

		dir, err := filepath.Abs(output)

		if err != nil {
			return "", fmt.Errorf("failed to generate absolute path for output: %w", err)
		}

		err = os.MkdirAll(dir, 0700)

		if err != nil {
			return "", fmt.Errorf("failed to create gonduit directory: %w", err)
		}

		err = crypto.GenerateBundles(filepath.Join(output, "server.pem"), filepath.Join(output, "app.pem"))

		if err != nil {
			return "", fmt.Errorf("failed to generate PKI: %w", err)
		}

		return fmtx.Successf("generated identities in %s", dir), nil

	}

	if fingerprint != "" {

		bundle, err := crypto.LoadBundle(fingerprint)

		if err != nil {
			return "", fmt.Errorf("failed to load identity bundle: %w", err)
		}

		return fmtx.Infof("fingerprint is %s", bundle.Fingerprint()), nil

	}

	if selfSigned {

		if err := app.SessionManager().UseSelfSignedIdentity(); err != nil {
			return "", fmt.Errorf("failed to use self-signed identity: %w", err)
		}

		return fmtx.Successf("using self-signed certificate"), nil

	}

	if load != "" {
		return fmtx.Successf("loaded identity %s", load), app.SessionManager().LoadIdentity(load)
	}

	if unload {
		return fmtx.Successf("unloaded identity"), app.SessionManager().UnloadIdentity()
	}

	return "", fmt.Errorf("no action specified")

}
