package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"gastank/internal/auth"
	"gastank/internal/providers/copilot"
	"gastank/internal/usage"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if handleCLI() {
		return
	}

	app := NewApp()

	wailsApp := application.New(application.Options{
		Name:        "Gastank",
		Description: "AI token usage monitor",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			// Accessory policy keeps the app out of the Dock and
			// removes it from Cmd+Tab, matching tray-app conventions.
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	// Tray window — starts hidden, frameless, always on top.
	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:           "Gastank",
		Width:           360,
		Height:          420,
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnFocusLost: true,
		HideOnEscape:    true,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
	})

	// Closing the window hides instead of quitting.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	// System tray — left-click toggles, right-click shows menu.
	tray := wailsApp.SystemTray.New()

	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayIconBytes())
	} else {
		tray.SetIcon(trayIconBytes())
	}

	tray.SetTooltip("Gastank — AI usage monitor")
	tray.AttachWindow(window).WindowOffset(5)

	menu := wailsApp.Menu.New()
	menu.Add("Show").OnClick(func(_ *application.Context) {
		tray.ShowWindow()
	})
	menu.Add("Hide").OnClick(func(_ *application.Context) {
		tray.HideWindow()
	})
	menu.AddSeparator()
	menu.Add("Quit Gastank").OnClick(func(_ *application.Context) {
		wailsApp.Quit()
	})
	tray.SetMenu(menu)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

// handleCLI processes command-line subcommands (usage, version, help).
// Returns true if a CLI command was handled (caller should exit), false to
// start the GUI.
func handleCLI() bool {
	if len(os.Args) < 2 {
		return false
	}

	switch os.Args[1] {
	case "--version", "version":
		fmt.Println(Version)
		return true

	case "usage":
		runUsageCLI(os.Args[2:])
		return true

	case "--help", "help":
		printHelp()
		return true
	}

	return false
}

// runUsageCLI fetches and prints usage data for a provider as JSON.
func runUsageCLI(args []string) {
	store := auth.NewStore()

	credsPath, err := auth.DefaultCredentialsPath()
	if err != nil {
		log.Printf("gastank: could not resolve credentials path: %v", err)
	} else {
		if err := store.Load(credsPath); err != nil {
			log.Printf("gastank: could not load credentials: %v", err)
		}
	}

	service := usage.NewService(
		copilot.NewProvider(copilot.Config{CredStore: store}),
	)

	providerName := copilot.ProviderName
	if len(args) > 0 {
		providerName = args[0]
	}

	report, err := service.Fetch(context.Background(), providerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gastank: %v\n", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "gastank: encode report: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	svc := usage.NewService(
		copilot.NewProvider(copilot.Config{}),
	)
	fmt.Fprintf(os.Stderr, "Gastank %s — AI token usage monitor\n\n", Version)
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  gastank                     Start the tray app (default)\n")
	fmt.Fprintf(os.Stderr, "  gastank usage [provider]    Fetch usage data as JSON\n")
	fmt.Fprintf(os.Stderr, "  gastank --version           Print version\n")
	fmt.Fprintf(os.Stderr, "  gastank --help              Show this help\n\n")
	fmt.Fprintf(os.Stderr, "Available providers: %s\n", strings.Join(svc.Providers(), ", "))
}
