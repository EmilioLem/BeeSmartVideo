# Implementation Plan for Bee Annotation Web Application

## Goal Description

Create a web platform that allows crowdsourced annotation of bee video frames. The workflow:
1. Upload a video (or select from a pre‑uploaded set of 4 videos).
2. Server extracts frames (e.g., 1 fps or configurable) using ffmpeg.
3. Random frames are presented to the user.
4. User clicks the centre of each bee in the image; multiple clicks per image are allowed.
5. When finished, the user presses **"Submit Annotations"** which sends the list of `{frameId, x, y}` to the backend.
6. Backend stores annotations in a relational database.
7. UI shows metrics: total frames, annotated frames, total bee points, per‑video progress, and a toggle to display existing community annotations overlayed on the current frame.
8. A live “dataset growth” visualization (e.g., a small sparkline or counter) updates as more annotations are added.

The system should be extensible for future automatic validation and merging of overlapping points.

---

## User Review Required

> **[!IMPORTANT]**
> The following design decisions need your confirmation before any code is written:
> 
> - **Database**: Use SQLite (file‑based, easy for a prototype) or PostgreSQL (more robust, supports concurrent writes). Which do you prefer?
> - **Backend framework**: Simple Node.js with Express, or a full‑stack like Next.js (React) for richer UI? The requirement mentions “maybe Node.js”. Please confirm.
> - **Frame extraction frequency**: Extract one frame per second, every Nth frame, or a fixed number of random frames per video? Define the default.
> - **Authentication**: Do you need user accounts or can annotations be anonymous? If authentication is required, should we use simple email‑based login or OAuth?
> - **Deployment environment**: Local development only, or containerised (Docker) for later deployment?
> 
> Confirm or adjust these points so we can finalize the implementation plan.

---

## Open Questions

> **[!WARNING]** Clarify the following before implementation:
> 
> 1. **Storage of video files** – Should the original videos be stored on the server filesystem, or referenced via URLs?
> 2. **Annotation format** – Apart from `(x, y)`, do you need timestamp, confidence, or tag type? Currently only centre points.
> 3. **Overlap handling** – Do you want a visual radius around each point to indicate possible overlap? (For future validation.)
> 4. **Metrics to display** – Besides total frames, annotated frames, and bee points, any other stats (e.g., average annotations per user, per‑video progress bar)?
> 5. **Toggle for past annotations** – Should the overlay be a heat‑map, individual dots, or both?
> 6. **Styling/Theme** – Any brand colours or design preferences? The system should be visually premium.

---

## Proposed Changes

### Backend (`server/`)

- **[NEW]** `package.json` – Node.js project with Express, Multer (file uploads), `fluent-ffmpeg`, and a DB driver (sqlite3 or pg).
- **[NEW]** `server.js` – Main entry point. Sets up routes:
  - `POST /upload` – Receive video, store, spawn ffmpeg job to extract frames to `frames/<videoId>/`.
  - `GET /videos` – List available videos with metadata (frame count).
  - `GET /frame/:videoId/:frameName` – Serve a specific frame image.
  - `GET /random-frame/:videoId` – Return a random frame name.
  - `POST /annotations` – Accept JSON payload of annotations and insert into DB.
  - `GET /metrics/:videoId` – Return JSON with counts for UI.
  - `GET /annotations/:videoId/:frameName` – Return existing community points for overlay.
- **[NEW]** `db/` – Contains migration scripts to create tables:
  - `videos(id, filename, frame_count)`
  - `frames(id, video_id, filename)`
  - `annotations(id, video_id, frame_name, x, y, created_at)`
- **[NEW]** `utils/ffmpeg.js` – Wrapper to extract frames (`ffmpeg -i input -vf fps=1 frames/%04d.jpg`).

### Frontend (`public/`)

- **index.html** – Main UI with a clean dark‑mode glassmorphism design.
- **styles.css** – Custom CSS using HSL palette, smooth gradients, and micro‑animations.
- **app.js** – Vanilla JavaScript handling:
  - Load video list, let user pick a video.
  - Fetch a random frame image and display in a canvas.
  - Capture click coordinates relative to image, draw a small marker.
  - Maintain an array of points for the current frame.
  - "Submit" button sends payload `{videoId, frameName, points:[{x,y}]}` via `fetch`.
  - Metrics panel queries `/metrics/:videoId` and updates counters live.
  - Toggle switch fetches `/annotations/:videoId/:frameName` and draws existing points (different colour).
- **metrics.js** – Helper to periodically refresh counts.

### UI / Design

- Dark background with a subtle gradient.
- Central card with glass effect for the frame canvas.
- Floating action button for **Submit**.
- Sidebar with video selector, metrics cards, and toggle for existing points.
- Animations: fade‑in of new frames, hover effects on buttons, subtle pulse on the submit button after a click.

### Verification Plan

- **Automated tests**: Use `jest` + `supertest` to verify API endpoints (upload, annotation storage, metrics).
- **Manual QA**: Steps to upload a sample video, annotate a few frames, confirm DB entries, and see metrics update.
- **Performance**: Ensure frame extraction runs asynchronously and does not block the server; use background worker (Node child process).

---

## Verification Plan

### Automated Tests
- `npm test` runs Jest suites covering:
  - Video upload route returns 200 and triggers frame extraction.
  - Random frame endpoint returns an existing image.
  - Annotation POST correctly stores points.
  - Metrics endpoint reflects inserted data.

### Manual Verification
1. Start server: `npm run dev`.
2. Open browser, upload one of the four videos.
3. Annotate a few frames, submit.
4. Refresh page – metrics counters should increase.
5. Toggle “Show community points” – previously submitted points appear as red dots.

---

*Once you approve the plan (including the open questions), I will create the task list and begin implementation.*
