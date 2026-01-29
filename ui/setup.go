package ui

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Mode int

const (
	ModeLive Mode = iota
	ModeVideo
)

type Setup struct {
	Mode Mode
	VideoPath string
	Debug bool
}

func usage(){
	fmt.Println("Usage:")
	fmt.Println("  BeeSmartVideo <mode> [debug]")
	fmt.Println("")
	fmt.Println("Modes:")
	fmt.Println(" * live")
	fmt.Println(" * video:/path_to_file")
	fmt.Println("")
	fmt.Println("Examples... not required, go ahead and figure it out!")
}

func ParseArgs(args []string) Setup {
	
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}

	var s Setup

	switch {
	case args[1] == "live":
		s.Mode = ModeLive

	case string.HasPrefix(args[1], "video:"):
		s.Mode = ModeVideo
		s.VideoPath = strings.TrimPrefix(args[1], "video:")
		if s.VideoPath == "" {
			log.Println("error: video mode requires a file path")
			os.Exit(1)
		}

	default:
		log.Printf("error: unknown mode: %s \n", args[1])
		usage()
		os.Exit(1)
	}

	if len(args)
}
