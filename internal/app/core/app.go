package core

import (
	"app"
	"app/command"
	"app/component"
	"app/component/completion"
	"app/component/completion/providers"
	"app/core/grpc"
	"app/log"
	"app/style"
	"context"
	"fmt"
	"os"
	"os/signal"
	"shared/pkg"
	"shared/util"
	"sync"
	"syscall"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type gonduitApp struct {

	// program is the main Bubble Tea program
	program *tea.Program

	// view is the main Bubble Tea view
	view tea.View

	// cmdMgr manages commands and their execution
	cmdMgr *command.Manager

	// sessionMgr creates and manages grpc sessions
	sessionMgr *grpc.Manager

	// history records and manages the command history
	history *component.History

	// scrollView is the main output view for the app
	scrollView *component.ScrollView

	// prompter manages prompt requests
	prompter *component.Prompter

	// completer provides command completion suggestions
	completer *component.Completer

	// loader is the top / bottom separator bar loader
	loader *component.Loader

	// progress is the file transfer progress bar
	progress progress.Model

	// spinner is the spinner used to indicate the app is busy (during command execution or transfers)
	spinner spinner.Model

	// lastFrame is the time of the last frame update
	lastFrame time.Time

	// transferActive indicates whether a file transfer is in progress
	transferActive bool

	// transferBytes represents the number of bytes transferred for the current transfer
	transferBytes uint64

	// transferTotal represents the total size of the file being transferred
	transferTotal uint64

	// transferName represents the name of the file being transferred
	transferName string

	// cancelCommand is used to cancel the current command execution
	cancelCommand context.CancelFunc

	// closeChan is used to signal the app to exit
	closeChan chan os.Signal

	// closeOnce guards the call to Close()
	closeOnce sync.Once
}

// NewApp creates a new terminal UI app
func NewApp() app.App {

	cmd := command.NewManager()

	instance := &gonduitApp{
		view:       tea.View{},
		cmdMgr:     cmd,
		history:    component.NewHistory(),
		scrollView: component.NewScrollView(),
		prompter:   component.NewPrompter(),
		completer:  makeCompleter(cmd),
		loader:     component.NewLoader(80, component.LoaderWrap),
		progress:   progress.New(progress.WithDefaultBlend()),
		spinner:    spinner.New(),
		lastFrame:  time.Now(),
		closeChan:  make(chan os.Signal, 1),
	}

	var p *tea.Program

	// Create a new tea program
	if pkg.IsDebug() {
		p = tea.NewProgram(instance, tea.WithoutSignalHandler(), tea.WithoutCatchPanics())
	} else {
		p = tea.NewProgram(instance, tea.WithoutSignalHandler())
	}

	// Initialize shell manager with app as listener
	instance.program = p

	instance.sessionMgr = grpc.NewManager(
		instance.scrollView,
		instance.prompter,
		instance.setAltView,
		instance.handleFileTransferProgress)

	instance.spinner.Spinner = spinner.MiniDot
	instance.spinner.Style = style.Running
	instance.view.WindowTitle = fmt.Sprintf("Gonduit - Version %s", pkg.Version)

	// Set up a channel to receive OS signals
	signal.Notify(instance.closeChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-instance.closeChan
		p.Send(ExitMsg{})
	}()

	util.ClearTerminal()

	return instance
}

func (app *gonduitApp) WelcomeMessage() string {
	return style.Muted.Render(" Welcome to Gonduit • Type 'help' for commands • Press Ctrl+C to cancel operations or type 'exit' to quit")
}

// Init initializes the Bubble Tea program
func (app *gonduitApp) Init() tea.Cmd {

	app.registerCommands()
	app.lastFrame = time.Now()

	// Add welcome message
	app.Logger().Write(app.WelcomeMessage())
	app.Logger().Write("")

	return tea.Batch(
		tea.ClearScreen,
		textinput.Blink,
		app.spinner.Tick,
		app.prompter.WaitForRequest(),
	)

}

func (app *gonduitApp) Close() {
	app.closeOnce.Do(func() {
		app.view.AltScreen = false
		app.sessionMgr.Close()
		app.prompter.Close()
		util.ClearTerminal()
		close(app.closeChan)
	})
}

func (app *gonduitApp) Logger() log.Logger {
	return app.scrollView
}

func (app *gonduitApp) SessionManager() *grpc.Manager {
	return app.sessionMgr
}

func (app *gonduitApp) Run() (tea.Model, error) {
	return app.program.Run()
}

// tick returns a command that sends periodic tick messages
func tick() tea.Cmd {
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func makeCompleter(m *command.Manager) *component.Completer {

	file := providers.NewFileProvider()
	dir := providers.NewDirectoryProvider()
	fs := providers.NewFileSystemProvider(file, dir)

	providerMap := map[completion.TokenType]completion.Provider{
		completion.TokenCommand:    providers.NewCommandProvider(m),
		completion.TokenFlag:       providers.NewFlagProvider(),
		completion.TokenFlagValue:  fs,
		completion.TokenPositional: fs,
	}

	return component.NewCompleter(m, providerMap)

}

func (app *gonduitApp) setAltView(v bool) {

	app.view.AltScreen = v

	if v {
		_ = app.program.ReleaseTerminal()
	} else {
		_ = app.program.RestoreTerminal()
	}

}
