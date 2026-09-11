# BeeSmartVideo v2

v2 runs the same detection/tracking pipeline as the root `main.go`, but with a
single, simple interface. There are **only two modes**.

| Mode | Command | Description |
| ---- | ------- | ----------- |
| **CLI** (default) | `go run .` | Headless. Reads `settings.json`, no window. systemd / server friendly. |
| **huh + GUI** | `go run . --huh` | Interactive `huh` form, then an `ffplay` window with the processed video. |

Both modes share the same processing loop, web dashboard, MQTT telemetry, and
dataset export.

## Quick start

```bash
cd v2

# CLI mode, using settings.json
go run .

# huh + GUI mode
go run . --huh
```

You can also start it from the repository root:

```bash
go run ./v2          # CLI
go run ./v2 --huh    # huh + GUI
```

v2 is **self-contained**: it always runs relative to this folder, so
`settings.json`, `videoSamples/`, and `dataset/` resolve here no matter where
the command is launched from.

## Choosing the video source

By default the source comes from `v2/settings.json`. The default is a video
sample, not the webcam:

```json
{
  "source": "./videoSamples/40secBeesWithPolen.mp4",
  "device_index": "2",
  "tracking_method": 3,
  "loop_video": true
}
```

There are three ways to pick a source:

1. **settings.json** — edit `source` (a path, or `"live"` for the webcam) and
   run `go run .`.
2. **`--source` flag** — override it for one run:

   ```bash
   go run . --source ./videoSamples/30secBeesWithPolen.mp4
   go run . --source ./videoSamples/beesCrossingSampleVideo1min.mp4
   go run . --source live          # force the webcam
   go run . ./videoSamples/1.5minBeesCrossing.mp4   # bare path shorthand
   ```

3. **huh form** — `go run . --huh` lists every file in `videoSamples/` plus
   `Live Webcam`, and saves your choice back to `settings.json`.

`videoSamples/` is a symlink to the repository's samples, so all four sample
videos are available without duplicating them.

Note: `--source` applies to CLI mode. In huh + GUI mode the source is chosen in
the form.

## Flags

| Flag | Effect |
| ---- | ------ |
| *(none)* | CLI mode, load `settings.json`. |
| `--huh` | huh + GUI mode. Aliases: `--gui`, `-gui`, `huhForm`, `--tui`, `-tui`, `-i`, `--interactive`. |
| `--source <path\|live>` | CLI source override. Also `--source=<path>` and `-source <path>`. |
| `<path>` | Shorthand for `--source <path>`. |
| `--no-aruco` | Disable the ArUco worker for this run. |
| `--aruco-script <path>` | Path to `ArUcoReader02.py` (default `./ArUcoReader02.py`). |
| `--python <path>` | Python interpreter for the worker (default `python3`). |
| `--debugImage` | Save the last 500 detected bee crops to `debugImages/` (troubleshooting). |

Any other argument is ignored, so a stray token never crashes the tool.

## Layout (self-contained)

```
v2/
├── main.go                 # entry point (two modes)
├── settings.json           # v2 configuration
├── ArUcoReader02.py        # ArUco worker (called by the Go client)
├── aruco_dict_10000x5.npz  # cached marker dictionary (fast startup)
├── videoSamples -> ../videoSamples
├── debugImages/            # --debugImage output (gitignored, last 500)
└── dataset/                # created when save_data = true
```

Other runtime outputs:

- Web dashboard: `http://localhost:8080`
- Dataset CSV + crops: `v2/dataset/` (when `save_data` is enabled)

## Configuration

`settings.json` fields are documented in `../menu/menu.go`. A quick summary:

| Field | Meaning |
| ----- | ------- |
| `source` | Video path or `"live"` for the webcam. |
| `device_index` | `/dev/video<n>` index when `source` is `live`. |
| `method` | Processing algorithm (1 = Efficient filtering v1). |
| `threshold_mode` | Threshold strategy ID (1 = Static 128). |
| `tracking_method` | Tracker ID (0 = None, 3 = Kalman, etc.). |
| `show_ids` | Overlay track IDs on the output. |
| `smoothness` | Pre-threshold blur level (0 = off). |
| `save_data` | Export CSV + cropped bee images. |
| `crop_size` | Exported crop side in **processing px** (full-res = x3). Default `64` = 192x192. |
| `loop_video` | Loop file playback. |
| `enable_telemetry` | Publish stats over MQTT. |
| `red_channel` | Threshold on the red channel (default `true`, see below). |
| `enable_aruco` | Decode ArUco marker IDs on confirmed tracks (default `true`). |

## IMPORTANT: Red LED illumination

The hive entrance is illuminated with **red LEDs only**, so the pipeline
thresholds on the **red channel** instead of luminance (that is what
`red_channel` controls, default on). Green/blue are mostly noise under red
light. The ArUco worker reads the same red channel. Set `"red_channel": false`
only when testing under white light (e.g. the sample videos).

## ArUco marker IDs

`v2` starts `ArUcoReader02.py` once (long-lived worker) and, for each newly
confirmed track, sends the bee crop and stores the decoded marker id on the
track. See `ArUcoReader02.py` for the protocol and `../ARUCO_PYTHON_TOOL_PROMPT.txt`
for the design. If Python/OpenCV is unavailable, v2 warns and continues without
marker decoding.

The live status second line shows the cumulative number of frames that contained
at least one ArUco tag, followed by the detected ids:

```
ArUco frames: 137 | track 7=H1-0042, track 12=H3-0242
```

While a tagged bee is on screen this grows about 30 per second (one per frame).

## Debugging crops (`--debugImage`)

Run `go run . --debugImage` to dump the crop of every confirmed track to
`debugImages/`. Files are named `bee_000.jpg` .. `bee_499.jpg`; the index wraps,
so the folder always contains only the **last 500** crops (in no particular
order). Use it to check what the ArUco worker actually receives. The folder is
gitignored.

