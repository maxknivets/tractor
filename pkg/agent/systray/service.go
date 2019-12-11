package systray

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/getlantern/systray"
	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/daemon"
	"github.com/manifold/tractor/pkg/data/icons"
	"github.com/manifold/tractor/pkg/log"
	"github.com/skratchdot/open-golang/open"
)

type Service struct {
	Agent  *agent.Agent
	Logger log.DebugLogger

	dm     *daemon.Daemon
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	inbox  chan Message
}

func (s *Service) InitializeDaemon() (err error) {
	s.inbox = make(chan Message)

	s.cmd = exec.Command(os.Args[0], "--", "systray")
	s.cmd.Stderr = os.Stderr
	s.cmd.Env = []string{"SYSTRAY_SUBPROCESS=1"}
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if s.stdin, err = s.cmd.StdinPipe(); err != nil {
		return err
	}
	if s.stdout, err = s.cmd.StdoutPipe(); err != nil {
		return err
	}

	return s.cmd.Start()
}

func (s *Service) Serve(ctx context.Context) {
	workspaces, err := s.Agent.Workspaces()
	if err != nil {
		panic(err)
	}

	go s.receiveMessages()

	var items []MenuItem
	for _, ws := range workspaces {
		items = append(items, MenuItem{
			Title:   ws.Name,
			Tooltip: "Open workspace",
			Icon:    "available",
			Enabled: true,
		})
	}
	items = append(items, MenuItem{
		Title: "-",
	}, MenuItem{
		Title:   "Shutdown",
		Tooltip: "Quit and shutdown all workspaces",
		Enabled: true,
	})
	s.send(Message{
		Type: InitMenu,
		Menu: &Menu{
			Tooltip: "Tractor",
			Icon:    "tractor",
			Items:   items,
		},
	})
	for msg := range s.inbox {
		switch msg.Type {
		case ItemClicked:
			s.Logger.Debug("clicked:", msg.Item.Title)
			if msg.Item.Title == "Shutdown" {
				s.Logger.Debug("wtf??")
				s.dm.Terminate()
				break
			}
			for _, ws := range workspaces {
				if ws.Name == msg.Item.Title {
					open.StartWith(ws.TargetPath, "Visual Studio Code.app")
				}
			}
		default:
			s.Logger.Debug("unknown:", msg)
		}
	}
}

func (s *Service) TerminateDaemon() error {
	return syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
}

func (s *Service) buildSystray() func() {
	return func() {
		systray.SetIcon(icons.Tractor)
		systray.SetTooltip("Tractor")

		workspaces, err := s.Agent.Workspaces()
		if err != nil {
			panic(err)
		}

		for _, ws := range workspaces {
			openItem := systray.AddMenuItem(ws.Name, "Open workspace")

			ws.OnStatusChange(func(ws *agent.Workspace) {
				openItem.SetIcon(ws.Status.Icon())
			})

			go func(mi *systray.MenuItem, ws *agent.Workspace) {
				for {
					<-openItem.ClickedCh
					open.StartWith(ws.TargetPath, "Visual Studio Code.app")
				}
			}(openItem, ws)
		}

		systray.AddSeparator()
		mQuitOrig := systray.AddMenuItem("Shutdown", "Quit and shutdown all workspaces")
		go func(mi *systray.MenuItem) {
			<-mi.ClickedCh
			s.dm.Terminate()
		}(mQuitOrig)
	}
}

func (s *Service) send(msg Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	s.stdin.Write(append(b, '\n'))
}

func (s *Service) receiveMessages() {
	scanner := bufio.NewScanner(s.stdout)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			s.Logger.Debug(err)
		}
		s.inbox <- msg
	}
	if err := scanner.Err(); err != nil {
		s.Logger.Debug(err)
	}
	close(s.inbox)
}
