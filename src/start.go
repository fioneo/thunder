package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"os/signal"
	"path/filepath"
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

const usage = `thunder - terminal music player

Usage:
  thunder [options] [song]

Options:
  -h, --help          Print this help and exit.
  -s, --sock path     Control socket path (default: thunder.sock in the temp dir).
  -l, --log path      Log file path (default: thunder.log in the temp dir).
  -p, --path path 	  Path to music directory (default: ~/Music).
  -v, --volume level  Initial volume (default: 100).

Examples:
  thunder
  thunder -p ~/music
  thunder -v 50 -s /tmp/thunder.sock ~/Music/song.mp3
`

type Args struct {
	sockPath  string
	logPath   string
	musicPath string
	volume    int
	song      string
}

func args() Args {
	flag.Usage = func() {
		fmt.Printf("%s\n", usage)
	}

	tmp := os.TempDir()

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "get home directory: %v", err)
		flag.Usage()
		os.Exit(1)
	}

	var help bool

	sockPath := filepath.Join(tmp, "thunder.sock")
	logPath := filepath.Join(tmp, "thunder.log")
	musicPath := filepath.Join(home, "Music")
	song := ""
	volume := 100

	// long
	flag.BoolVar(&help, "help", false, "")
	flag.StringVar(&sockPath, "sock", sockPath, "")
	flag.StringVar(&logPath, "log", logPath, "")
	flag.StringVar(&musicPath, "path", musicPath, "")
	flag.IntVar(&volume, "volume", 100, "")

	// short
	flag.BoolVar(&help, "h", false, "")
	flag.StringVar(&sockPath, "s", sockPath, "")
	flag.StringVar(&logPath, "l", logPath, "")
	flag.StringVar(&musicPath, "p", musicPath, "")
	flag.IntVar(&volume, "v", 100, "")

	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	if volume < 0 || volume > 100 {
		fmt.Fprintf(os.Stderr, "invalid volume: %d: must be between 0 and 100.\n\n", volume)
		flag.Usage()
		os.Exit(1)
	}

	if args := flag.Args(); len(args) > 0 {
		song = args[0]
	}

	return Args{
		sockPath:  sockPath,
		logPath:   logPath,
		musicPath: musicPath,
		volume:    volume,
		song:      song,
	}
}

func start(app *tview.Application) {
	thunder = NewThunder()
	args := args()
	thunder.InitLogger(args)

	thunder.InitPanels(app, args)
	thunder.InitServer(args)

	thunder.renderPlayback(thunder.Playback.Snapshot())

	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(thunder.Queue, 0, 1, false).AddItem(thunder.PlayBar, 9, 0, false)
	changed := false

	flex := tview.NewFlex().
		AddItem(thunder.Library, 0, 2, true).
		AddItem(innerFlex, 0, 6, false)

	thunder.Pages.AddPage("main", flex, true, true)
	thunder.App.SetRoot(thunder.Pages, true).SetFocus(thunder.Library)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyCtrlC:
			return e
		case tcell.KeyCtrlQ:
			cancel()
			return nil
		}

		if thunder.Popups.Count() > 0 {
			return e
		}

		switch e.Key() {
		case tcell.KeyCtrlT:
			downloadPopup()
			return nil
		case tcell.KeyTab:
			thunder.CyclePanels(true)
			return nil
		case tcell.KeyBacktab:
			thunder.CyclePanels(false)
			return nil
		case tcell.KeyRune:
			if e.Rune() == 'q' {
				cancel()
				return nil
			}
		}
		return thunder.handlePlaybackKey(e)
	})

	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		_, h := screen.Size()
		size := 9
		if h <= 25 {
			size = 7
			changed = true
		} else if changed {
			size = 9
			changed = false
		}
		innerFlex.ResizeItem(thunder.PlayBar, size, 0)

		return false
	})

	tview.Styles.PrimitiveBackgroundColor = thunder.Colors.Background

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		redrawAndWrite(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		logErrors(ctx)
	}()

	go func() {
		<-ctx.Done()
		thunder.Stop()
	}()

	thunder.Logger.Info("Successfully initialized application")

	if args.song != "" {
		name := filepath.Base(args.song)
		song := NewSong(name, args.song)
		thunder.Playback.Enqueue(song)
		thunder.Playback.Next()
	}

	defer func() {
		if p := recover(); p != nil {
			app.Stop()
			thunder.Logger.Errorf("panic: %s", p)
			fmt.Fprintf(os.Stderr, "Unexpected panic.\nSee log file.\n")
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
