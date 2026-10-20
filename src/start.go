package main

import (
	"context"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const redrawInterval = 500 * time.Millisecond

func redrawAndWrite(ctx context.Context) {
	ticker := time.NewTicker(redrawInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s := thunder.Playback.Snapshot()
			if s.Active() {
				thunder.App.QueueUpdateDraw(func() {})
				thunder.Server.Write(s)
			}
		}
	}

}

func logErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case err := <-thunder.ErrCh:
					if err != nil {
						thunder.Logger.Error(err)
					}
				default:
					return
				}
			}
		case err := <-thunder.ErrCh:
			if err != nil {
				thunder.Logger.Error(err)
			}
		}
	}
}

func start(app *tview.Application) {
	thunder = NewThunder()
	thunder.InitLogger()

	thunder.InitPanels(app)
	thunder.InitServer()

	thunder.renderPlayback(thunder.Playback.Snapshot())

	flex := tview.NewFlex().
		AddItem(thunder.Library, 0, 2, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(thunder.PlayPanel, 0, 7, false).AddItem(thunder.PlayBar, 0, 3, false), 0, 6, false)

	thunder.Pages.AddPage("main", flex, true, true)
	thunder.App.SetRoot(thunder.Pages, true).SetFocus(thunder.Library)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		pageName, _ := thunder.Pages.GetFrontPage()
		if pageName != "main" && !strings.Contains(pageName, "timeout_popup") {
			return e
		}
		switch e.Key() {
		case tcell.KeyCtrlQ:
			cancel()
		case tcell.KeyTab:
			thunder.CyclePanels(true)
		case tcell.KeyBacktab:
			thunder.CyclePanels(false)
		}
		return thunder.handlePlaybackKey(e)
	})

	tview.Styles.PrimitiveBackgroundColor = thunder.Colors.Background

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		redrawAndWrite(ctx)
	}()

	go func() {
		defer wg.Done()
		logErrors(ctx)
	}()

	go func() {
		<-ctx.Done()
		thunder.Stop()
	}()

	thunder.Logger.Info("Successfully initialized application")

	defer func() {
		if p := recover(); p != nil {
			thunder.Logger.Errorf("panic: %s", p)
		}
	}()

	// Main loop
	err := thunder.App.Run()
	cancel()
	wg.Wait()

	if err != nil {
		exit(err)
	}
}
