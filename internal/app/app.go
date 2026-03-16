package app

import (
	"app/core/grpc"
	"app/log"
	"context"

	tea "charm.land/bubbletea/v2"
)

type App interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() tea.View
	WelcomeMessage() string
	PromptInput(ctx context.Context, prompt string, secure bool) (string, error)
	SessionManager() *grpc.Manager
	Logger() log.Logger
	Run() (tea.Model, error)
}
