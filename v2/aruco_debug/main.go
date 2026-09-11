// Command aruco_debug sends the recorded debug crops to ArUcoReader02.py and
// reports which ones decode. It talks to the worker through the exact same
// package as v2/main.go, so it can be used to troubleshoot the decoder without
// running the whole video pipeline.
//
// Run it from the v2 folder:
//
//	go run ./aruco_debug                              # worker default dictionary
//	go run ./aruco_debug -dict DICT_4X4_50            # predefined dictionary
//	go run ./aruco_debug -channel gray -upscale-min 0 # tweak worker settings
package main

import (
	"BeeSmartVideo/aruco"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	dir := flag.String("dir", "debugImages", "folder containing recorded crops")
	script := flag.String("script", "./ArUcoReader02.py", "path to ArUcoReader02.py")
	python := flag.String("python", "python3", "python interpreter")
	dictName := flag.String("dict", "", "predefined dictionary, e.g. DICT_4X4_50 (empty = worker default)")
	channel := flag.String("channel", "", "worker channel: red/gray/green/blue (empty = worker default)")
	upscaleMin := flag.Int("upscale-min", -1, "worker upscale-min (-1 = worker default)")
	batch := flag.Int("batch", 64, "images per worker request")
	limit := flag.Int("limit", 0, "only test the first N images (0 = all)")
	quiet := flag.Bool("quiet", false, "only print the summary")
	flag.Parse()

	var workerArgs []string
	if *dictName != "" {
		workerArgs = append(workerArgs, "--dict", *dictName)
	}
	if *channel != "" {
		workerArgs = append(workerArgs, "--channel", *channel)
	}
	if *upscaleMin >= 0 {
		workerArgs = append(workerArgs, "--upscale-min", fmt.Sprint(*upscaleMin))
	}

	files, _ := filepath.Glob(filepath.Join(*dir, "*.jpg"))
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no .jpg images found in %s\n", *dir)
		os.Exit(1)
	}
	sort.Strings(files)
	if *limit > 0 && len(files) > *limit {
		files = files[:*limit]
	}

	client, err := aruco.NewClient(*python, *script, workerArgs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start worker: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	decoded, errored := 0, 0
	idFrames := map[int]int{}

	for start := 0; start < len(files); start += *batch {
		end := start + *batch
		if end > len(files) {
			end = len(files)
		}

		images := make([]aruco.Image, 0, end-start)
		for i := start; i < end; i++ {
			raw, err := os.ReadFile(files[i])
			if err != nil {
				continue
			}
			images = append(images, aruco.Image{
				TrackID: i,
				JPEGB64: base64.StdEncoding.EncodeToString(raw),
			})
		}

		results, err := client.Decode(images)
		if err != nil {
			fmt.Fprintf(os.Stderr, "decode batch %d-%d: %v\n", start, end-1, err)
			errored += end - start
			continue
		}

		for _, r := range results {
			name := filepath.Base(files[r.TrackID])
			if r.Error != nil && *r.Error != "" {
				errored++
			}
			if r.Found && r.MarkerID != nil {
				decoded++
				idFrames[*r.MarkerID]++
				if !*quiet {
					fmt.Printf("OK  %-16s marker_id=%-4d label=%s\n", name, *r.MarkerID, r.Label)
				}
			} else if !*quiet {
				fmt.Printf("--  %-16s no marker\n", name)
			}
		}
	}

	fmt.Printf("\nsummary: images=%d decoded=%d errors=%d\n", len(files), decoded, errored)
	if decoded > 0 {
		ids := make([]int, 0, len(idFrames))
		for id := range idFrames {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		fmt.Print("distinct markers: ")
		for _, id := range ids {
			fmt.Printf("%d(%d) ", id, idFrames[id])
		}
		fmt.Println()
	}
}
