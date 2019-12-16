package systray

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"

	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/misc/daemon"
	"github.com/manifold/tractor/pkg/misc/logging"
	"github.com/manifold/tractor/pkg/misc/subcmd"
	"github.com/skratchdot/open-golang/open"
)

type Service struct {
	Agent  *agent.Agent
	Logger logging.DebugLogger
	Daemon *daemon.Daemon

	subcmd *subcmd.Subcmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	inbox  chan Message
}

func (s *Service) InitializeDaemon() (err error) {
	go func() {
		for range s.Agent.WorkspacesChanged {
			if s.subcmd != nil {
				s.subcmd.Restart()
			}
		}
	}()
	return s.start()
}

func (s *Service) start() (err error) {
	s.subcmd = subcmd.New(os.Args[0], "--", "systray")
	s.subcmd.Started = make(chan *exec.Cmd)
	s.subcmd.Setup = func(cmd *exec.Cmd) error {

		cmd.Stderr = os.Stderr
		cmd.Env = []string{"SYSTRAY_SUBPROCESS=1"}
		if s.stdin, err = cmd.StdinPipe(); err != nil {
			return err
		}
		if s.stdout, err = cmd.StdoutPipe(); err != nil {
			return err
		}

		return nil
	}

	return s.subcmd.Start()
}

func (s *Service) Serve(ctx context.Context) {
	for range s.subcmd.Started {
		s.inbox = make(chan Message)
		go s.receiveMessages()

		workspaces, err := s.Agent.Workspaces()
		if err != nil {
			panic(err)
		}

		var items []MenuItem
		for idx, ws := range workspaces {
			items = append(items, MenuItem{
				Title:   ws.Name,
				Tooltip: "Open workspace",
				Icon:    ws.Status().String(),
				Enabled: true,
			})
			ws.OnStatusChange(func(ws *agent.Workspace) {
				s.send(Message{
					Type: ItemUpdate,
					Item: &MenuItem{
						Title:   ws.Name,
						Tooltip: "Open workspace",
						Icon:    ws.Status().String(),
						Enabled: true,
					},
					Idx: idx,
				})
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
				if msg.Item.Title == "Shutdown" {
					s.Daemon.Terminate()
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
}

func (s *Service) Reload() {
	s.subcmd.Restart()
}

func (s *Service) TerminateDaemon() error {
	return s.subcmd.Stop()
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
			break
		}
		s.inbox <- msg
	}
	if err := scanner.Err(); err != nil {
		s.Logger.Debug(err)
	}
	close(s.inbox)
}
