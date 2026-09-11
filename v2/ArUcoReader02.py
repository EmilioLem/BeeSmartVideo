#!/usr/bin/env python3
"""ArUcoReader02.py - headless ArUco batch worker for BeeSmartVideo.

This is the callable-tool version of the standalone Raspberry Pi daemon
(ArUcoReader01.py). The camera, the transit/CSV logic and the GUI were removed;
what remains is a fast, headless marker decoder that the Go video pipeline can
drive. Detection settings match 01:

    * dictionary: DICT_4X4_50 by default. The tags used in the recorded
      samples are 4x4 markers. See the tag-size note below.
    * red channel (the hive LED is red)
    * minMarkerPerimeterRate = 0.010
    * CORNER_REFINE_SUBPIX

TAG-SIZE NOTE
-------------
ArUcoReader01.py used cv2.aruco.extendDictionary(10000, 5), a custom 5x5
dictionary with room for 10000 ids (the H1..H5 scheme). The tags in the
recorded samples are standard 4x4 markers, which no 5x5 dictionary can read.
A 4x4 dictionary is much smaller: DICT_4X4_50 has 50 ids, and even
DICT_4X4_1000 only reaches 1000. So the 10000-marker / H1..H5 scheme cannot be
used with 4x4 tags; the decoder defaults to DICT_4X4_50 to match the samples.
Pass --dict custom (optionally with --custom-markers/--marker-bits) to use the
old 5x5 dictionary instead.

The custom dictionary is generated once and cached next to this script
(aruco_dict_10000x5.npz), because extendDictionary() is O(n^2) and takes
~2 minutes for 10000 markers. It is only used with --dict custom. Delete the
.npz (or pass --rebuild-dict) to regenerate it.

Two ways to run
---------------
Worker mode (default, long-lived - this is how Go calls it):

    echo '{"v":1,"id":1,"images":[{"track_id":7,"jpeg_b64":"<base64>"}]}' \\
        | python3 ArUcoReader02.py

    Reads newline-delimited JSON on stdin, writes newline-delimited JSON on
    stdout. stdout carries JSON only; all logs go to stderr.

One-shot testing:

    python3 ArUcoReader02.py --image crop.jpg
    python3 ArUcoReader02.py --benchmark ./dataset/images

Protocol (v1)
-------------
Request:
    {"v":1,"id":<int>,"images":[
        {"track_id":<int>,"jpeg_b64":"<base64 jpeg>"},
        {"track_id":<int>,"jpeg_path":"/path/to.jpg"}, ...]}

Response:
    {"v":1,"id":<int>,"results":[
        {"track_id":<int>,"found":<bool>,"marker_id":<int|null>,
         "marker_ids":[<int>,...],"label":<string|null>,
         "corners":[[x,y],...]|null,"confidence":<float>,
         "error":<string|null>}, ...]}
"""

from __future__ import annotations

import argparse
import base64
import glob
import json
import os
import statistics
import sys
import time
from concurrent.futures import ThreadPoolExecutor

import cv2
import numpy as np

PROTOCOL_VERSION = 1
DEFAULT_DICT = "DICT_4X4_50"  # 4x4 tags, matching the recorded samples
DEFAULT_CUSTOM_MARKERS = 10000
DEFAULT_MARKER_BITS = 5
DEFAULT_CHANNEL = "red"
DEFAULT_UPSCALE_MIN = 150  # upscale crops so the shortest side reaches this


def log(msg: str) -> None:
    """Log to stderr so stdout stays a clean JSON stream."""
    print(msg, file=sys.stderr, flush=True)


def emit(obj) -> None:
    """Write one compact JSON line to stdout and flush immediately."""
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def obtener_nomenclatura(id_entero: int) -> str:
    """H1..H5 hive + 4-digit individual, same rule as ArUcoReader01.py."""
    grupo = (id_entero // 2000) + 1
    individuo = id_entero % 2000
    return f"H{grupo}-{individuo:04d}"


def build_detector(args):
    if args.dict and args.dict.lower() != "custom":
        attr = args.dict if args.dict.startswith("DICT_") else f"DICT_{args.dict}"
        if not hasattr(cv2.aruco, attr):
            raise SystemExit(f"Unknown predefined dictionary: {attr}")
        dictionary = cv2.aruco.getPredefinedDictionary(getattr(cv2.aruco, attr))
        dict_name = f"predefined:{attr}"
    else:
        dictionary = load_or_build_custom_dict(
            args.custom_markers, args.marker_bits, args.rebuild_dict
        )
        dict_name = f"extend:{args.custom_markers}x{args.marker_bits}"

    params = cv2.aruco.DetectorParameters()
    params.minMarkerPerimeterRate = 0.010
    params.cornerRefinementMethod = cv2.aruco.CORNER_REFINE_SUBPIX

    return cv2.aruco.ArucoDetector(dictionary, params), dict_name


def custom_dict_cache_path(n_markers: int, marker_bits: int) -> str:
    here = os.path.dirname(os.path.abspath(__file__))
    return os.path.join(here, f"aruco_dict_{n_markers}x{marker_bits}.npz")


def load_or_build_custom_dict(n_markers: int, marker_bits: int, rebuild: bool = False):
    """extendDictionary() is O(n^2): 10000x5 takes ~2 minutes. Build it once and
    cache it next to this script so subsequent starts are instant. The custom
    dictionary is deterministic (default seed), so the cache always matches
    what ArUcoReader01.py builds at boot."""
    cache = custom_dict_cache_path(n_markers, marker_bits)

    if not rebuild and os.path.exists(cache):
        data = np.load(cache)
        log(f"Loaded cached dictionary {os.path.basename(cache)}")
        return cv2.aruco.Dictionary(
            data["bytesList"], int(data["markerSize"]), int(data["maxCorrectionBits"])
        )

    log(
        f"Building custom dictionary extendDictionary({n_markers},{marker_bits}) "
        "- this can take a while, but only once..."
    )
    dictionary = cv2.aruco.extendDictionary(n_markers, marker_bits)
    try:
        np.savez(
            cache,
            bytesList=dictionary.bytesList,
            markerSize=dictionary.markerSize,
            maxCorrectionBits=dictionary.maxCorrectionBits,
        )
        log(f"Cached dictionary to {os.path.basename(cache)}")
    except Exception as exc:
        log(f"Warning: could not write dictionary cache: {exc}")
    return dictionary


def select_channel(img_bgr, mode: str):
    if mode == "red":
        return img_bgr[:, :, 2]
    if mode == "green":
        return img_bgr[:, :, 1]
    if mode == "blue":
        return img_bgr[:, :, 0]
    return cv2.cvtColor(img_bgr, cv2.COLOR_BGR2GRAY)


def maybe_upscale(gray, min_side: int):
    """Upscale small crops so tiny markers survive detection. Returns (img, scale)."""
    h, w = gray.shape[:2]
    shortest = min(h, w)
    if min_side <= 0 or shortest >= min_side:
        return gray, 1.0
    scale = min_side / float(shortest)
    resized = cv2.resize(
        gray,
        (int(round(w * scale)), int(round(h * scale))),
        interpolation=cv2.INTER_CUBIC,
    )
    return resized, scale


def detect_markers(gray, detector, upscale_min: int):
    prepared, scale = maybe_upscale(gray, upscale_min)
    corners, ids, _ = detector.detectMarkers(prepared)
    if ids is None:
        return []

    inv = 1.0 / scale if scale else 1.0
    markers = []
    for i, id_arr in enumerate(ids):
        marker_id = int(id_arr[0])
        pts = corners[i][0]
        area = abs(cv2.contourArea(pts.astype(np.float32))) * inv * inv
        markers.append(
            {
                "marker_id": marker_id,
                "label": obtener_nomenclatura(marker_id),
                "corners": [[float(x * inv), float(y * inv)] for x, y in pts],
                "area": float(area),
            }
        )

    markers.sort(key=lambda m: m["area"], reverse=True)
    return markers


def decode_bytes(raw: bytes, detector, channel: str, upscale_min: int):
    arr = np.frombuffer(raw, dtype=np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError("could not decode image bytes")
    gray = select_channel(img, channel)
    return detect_markers(gray, detector, upscale_min)


def make_result(track_id, markers=None, error=None):
    if error is not None:
        return {
            "track_id": track_id,
            "found": False,
            "marker_id": None,
            "marker_ids": [],
            "label": None,
            "corners": None,
            "confidence": 0.0,
            "error": error,
        }

    ids = [m["marker_id"] for m in markers]
    best = markers[0] if markers else None
    return {
        "track_id": track_id,
        "found": bool(best),
        "marker_id": best["marker_id"] if best else None,
        "marker_ids": ids,
        "label": best["label"] if best else None,
        "corners": best["corners"] if best else None,
        "confidence": 1.0 if best else 0.0,
        "error": None,
    }


def decode_entry(entry, detector, args):
    track_id = entry.get("track_id")
    try:
        if entry.get("jpeg_b64"):
            raw = base64.b64decode(entry["jpeg_b64"])
        elif entry.get("jpeg_path"):
            with open(entry["jpeg_path"], "rb") as f:
                raw = f.read()
        else:
            return make_result(track_id, error="missing jpeg_b64/jpeg_path")
        markers = decode_bytes(raw, detector, args.channel, args.upscale_min)
        return make_result(track_id, markers)
    except Exception as exc:  # one bad crop must not kill the worker
        return make_result(track_id, error=f"{type(exc).__name__}: {exc}")


def handle_request(req, detector, args):
    rid = req.get("id")
    if req.get("v") != PROTOCOL_VERSION:
        return {
            "v": PROTOCOL_VERSION,
            "id": rid,
            "results": [],
            "error": f"unsupported version: {req.get('v')}",
        }

    images = req.get("images") or []
    if not isinstance(images, list):
        return {
            "v": PROTOCOL_VERSION,
            "id": rid,
            "results": [],
            "error": "images must be a list",
        }

    if args.threads > 1 and len(images) > 1:
        with ThreadPoolExecutor(max_workers=args.threads) as pool:
            results = list(pool.map(lambda e: decode_entry(e, detector, args), images))
    else:
        results = [decode_entry(e, detector, args) for e in images]

    return {"v": PROTOCOL_VERSION, "id": rid, "results": results}


def worker_loop(detector, args, dict_name: str) -> None:
    log(
        f"ArUcoReader02 worker ready | dict={dict_name} | channel={args.channel} "
        f"| upscale_min={args.upscale_min} | threads={args.threads}"
    )
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
        except Exception as exc:
            emit(
                {
                    "v": PROTOCOL_VERSION,
                    "id": None,
                    "results": [],
                    "error": f"invalid json: {exc}",
                }
            )
            continue

        if not isinstance(req, dict):
            emit(
                {
                    "v": PROTOCOL_VERSION,
                    "id": None,
                    "results": [],
                    "error": "request must be a JSON object",
                }
            )
            continue

        try:
            resp = handle_request(req, detector, args)
        except Exception as exc:
            resp = {
                "v": PROTOCOL_VERSION,
                "id": req.get("id"),
                "results": [],
                "error": f"{type(exc).__name__}: {exc}",
            }
        emit(resp)


def mode_image(path: str, detector, args) -> None:
    try:
        with open(path, "rb") as f:
            raw = f.read()
        result = make_result(None, decode_bytes(raw, detector, args.channel, args.upscale_min))
    except Exception as exc:
        result = make_result(None, error=f"{type(exc).__name__}: {exc}")
    emit({"v": PROTOCOL_VERSION, "id": None, "image": path, "results": [result]})


def mode_benchmark(dirpath: str, detector, args) -> None:
    files = []
    for pattern in ("*.jpg", "*.jpeg", "*.png", "*.bmp"):
        files.extend(glob.glob(os.path.join(dirpath, pattern)))
        files.extend(glob.glob(os.path.join(dirpath, pattern.upper())))
    files = sorted(set(files))

    times_ms = []
    found = 0
    for path in files:
        start = time.perf_counter()
        try:
            with open(path, "rb") as f:
                raw = f.read()
            markers = decode_bytes(raw, detector, args.channel, args.upscale_min)
            if markers:
                found += 1
        except Exception:
            pass
        times_ms.append((time.perf_counter() - start) * 1000.0)

    def stat(q):
        if not times_ms:
            return None
        ordered = sorted(times_ms)
        idx = min(len(ordered) - 1, int(round(q * (len(ordered) - 1))))
        return round(ordered[idx], 3)

    emit(
        {
            "v": PROTOCOL_VERSION,
            "mode": "benchmark",
            "dir": dirpath,
            "images": len(files),
            "found": found,
            "ms_mean": round(statistics.mean(times_ms), 3) if times_ms else None,
            "ms_median": round(statistics.median(times_ms), 3) if times_ms else None,
            "ms_p95": stat(0.95),
            "ms_max": round(max(times_ms), 3) if times_ms else None,
        }
    )


def parse_args(argv):
    parser = argparse.ArgumentParser(
        description="Headless ArUco batch worker for BeeSmartVideo (from ArUcoReader01.py)."
    )
    parser.add_argument("--image", help="one-shot: decode a single image and print JSON")
    parser.add_argument(
        "--benchmark", help="decode every image in a folder and print latency stats"
    )
    parser.add_argument(
        "--dict",
        default=DEFAULT_DICT,
        help=(
            "predefined dictionary name, e.g. DICT_4X4_50 (default) or DICT_4X4_1000; "
            "pass 'custom' for the 10000-marker 5x5 dictionary from ArUcoReader01.py"
        ),
    )
    parser.add_argument("--custom-markers", type=int, default=DEFAULT_CUSTOM_MARKERS)
    parser.add_argument("--marker-bits", type=int, default=DEFAULT_MARKER_BITS)
    parser.add_argument(
        "--rebuild-dict",
        action="store_true",
        help="ignore the cached custom dictionary and rebuild it (slow)",
    )
    parser.add_argument(
        "--channel",
        choices=["red", "gray", "green", "blue"],
        default=DEFAULT_CHANNEL,
        help="which channel feeds the detector (default: red, matching 01)",
    )
    parser.add_argument(
        "--upscale-min",
        type=int,
        default=DEFAULT_UPSCALE_MIN,
        help="upscale crops so the shortest side reaches this many px (0 disables)",
    )
    parser.add_argument("--threads", type=int, default=1)
    return parser.parse_args(argv)


def main(argv=None) -> int:
    args = parse_args(sys.argv[1:] if argv is None else argv)
    detector, dict_name = build_detector(args)

    if args.image:
        mode_image(args.image, detector, args)
        return 0
    if args.benchmark:
        mode_benchmark(args.benchmark, detector, args)
        return 0

    worker_loop(detector, args, dict_name)
    return 0


if __name__ == "__main__":
    sys.exit(main())
