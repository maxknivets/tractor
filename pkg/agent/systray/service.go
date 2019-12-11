package systray

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/daemon"
	"github.com/manifold/tractor/pkg/logging"
	"github.com/skratchdot/open-golang/open"
)

type Service struct {
	Agent    *agent.Agent
	Logger   logging.DebugLogger
	Daemon   *daemon.Daemon
	ReloadCh chan struct{}

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	inbox   chan Message
	started chan struct{}
}

func (s *Service) InitializeDaemon() (err error) {
	s.started = make(chan struct{})
	if s.ReloadCh != nil {
		go func() {
			for range s.ReloadCh {
				s.Reload()
			}
		}()
	}
	return s.start()
}

func (s *Service) start() (err error) {
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
	err = s.cmd.Start()
	if err != nil {
		return err
	}
	go func() {
		s.started <- struct{}{}
		err := s.cmd.Wait()
		if err != nil {
			s.Logger.Debug("systay exit err:", err)
			return
		}
		if s.started != nil {
			if err := s.start(); err != nil {
				s.Logger.Debug("systray start err:", err)
			}
		}
	}()
	return nil
}

func (s *Service) Serve(ctx context.Context) {
	for range s.started {
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
				Icon:    ws.Status.String(),
				Enabled: true,
			})
			ws.OnStatusChange(func(ws *agent.Workspace) {
				s.send(Message{
					Type: ItemUpdate,
					Item: &MenuItem{
						Title:   ws.Name,
						Tooltip: "Open workspace",
						Icon:    ws.Status.String(),
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
	if s.started == nil {
		return
	}
	if s.cmd == nil {
		return
	}
	syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
}

func (s *Service) TerminateDaemon() error {
	if s.started == nil {
		return nil
	}
	close(s.started)
	s.started = nil
	return syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
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

func exitStatus(err error) int {
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 0
}
