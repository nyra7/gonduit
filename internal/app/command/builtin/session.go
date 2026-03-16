package builtin

import (
	"app"
	"app/command"
	"app/fmtx"
	"app/style"
	"errors"
	"fmt"
	"shared/platform"
	"sort"
	"strings"
)

func MakeSessionCommand(app app.App) command.Handler {

	flags := []command.Argument{
		command.NewArgument("list", "List all current sessions", command.ArgTypeBool, false),
		command.NewArgument("kill", "Kill the specified session", command.ArgTypeInt, false),
		command.NewArgument("interact", "Use the specified session", command.ArgTypeInt, false),
	}

	return command.NewHandler("session", wrappedExecutor(app, sessionCmd), "Manage active server connections", nil, flags)

}

func sessionCmd(app app.App, ctx *command.Context) (string, error) {

	list := ctx.BoolFlag("list")
	kill := ctx.IntFlag("kill")
	i := ctx.IntFlag("interact")

	if !list && kill == 0 && i == 0 {
		return "", errors.New("the command requires at least one flag")
	}

	if list {
		return formatSessions(app.SessionManager().Sessions()), nil
	}

	if kill > 0 {
		return fmtx.Successf("closed session %d", kill), app.SessionManager().CloseSession(uint64(kill))
	}

	if i > 0 {
		return fmtx.Successf("now using session %d", i), app.SessionManager().UseSession(uint64(i))
	}

	return "", nil

}

func formatSessions(ids map[uint64]platform.HostInfo) string {
	if len(ids) == 0 {
		return style.InfoLabel.Render("Sessions: ") +
			style.InfoStyle.Render("none available")
	}

	// Sort for stable output
	keys := make([]uint64, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	header := style.InfoLabel.Render("Sessions: ") +
		style.InfoStyle.Render(fmt.Sprintf("%d available", len(ids)))

	lines := make([]string, 0, len(ids)+1)
	lines = append(lines, header)

	for _, id := range keys {
		s := ids[id]

		userHost := fmt.Sprintf("%s@%s", s.Username, s.Hostname)
		ipAdapter := fmt.Sprintf("(%s via %s at %s)", s.Ip, s.NetworkAdapter, s.WorkingDir)

		line := fmt.Sprintf(
			"%s %s %s %s → %s",
			style.Muted.Render("•"),
			style.Primary.Render(fmt.Sprintf("Session %d:", id)),
			userHost,
			style.InfoStyle.Render(ipAdapter),
			style.Muted.Render(s.OsInfo.String()),
		)

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
