package systray

import (
	"context"
	"log"

	"github.com/getlantern/systray"
	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/daemon"
	"github.com/manifold/tractor/pkg/data/icons"
	"github.com/skratchdot/open-golang/open"
)

type Service struct {
	Agent *agent.Agent
	dm    *daemon.Daemon
}

func (s *Service) Serve(ctx context.Context) {
	systray.Run(s.buildSystray(), func() {})
}

func (s *Service) TerminateDaemon() error {
	systray.Quit() // FIXME: THIS TERMINATES THE PROCESS
	return nil
}

func (s *Service) buildSystray() func() {
	return func() {
		systray.SetIcon(icons.Tractor)
		systray.SetTooltip("Tractor")

		workspaces, err := s.Agent.Workspaces()
		if err != nil {
			log.Fatal(err)
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
