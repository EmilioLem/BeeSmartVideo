// server/server.js
const express = require('express');
const path = require('path');
const fs = require('fs');
const { extractFrames } = require('../utils/ffmpeg');
const sqlite3 = require('sqlite3').verbose();
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname, '../public')));
app.use('/videoSamples', express.static(path.join(__dirname, '../videoSamples')));

// SQLite DB (file based)
const DB_PATH = path.join(__dirname, '../db/annotations.db');
const db = new sqlite3.Database(DB_PATH);

// Ensure DB tables exist (run init script if needed)
const initSql = `
CREATE TABLE IF NOT EXISTS videos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  filename TEXT NOT NULL,
  frame_count INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS annotations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  video_id INTEGER NOT NULL,
  frame_name TEXT NOT NULL,
  x REAL NOT NULL,
  y REAL NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(video_id) REFERENCES videos(id)
);
`;
db.exec(initSql, (err) => {
  if (err) {
    console.error('DB init error:', err);
  } else {
    initializeVideos();
  }
});

// Automatically scan and discover videos on startup
async function initializeVideos() {
  const videoSamplesDir = path.join(__dirname, '../videoSamples');
  
  try {
    if (!fs.existsSync(videoSamplesDir)) {
      fs.mkdirSync(videoSamplesDir, { recursive: true });
    }
    const files = fs.readdirSync(videoSamplesDir);
    const videoFiles = files.filter(f => f.match(/\.(mp4|mkv|avi|mov)$/i));
    
    console.log(`[Auto-Discovery] Found ${videoFiles.length} video files in videoSamples/`);
    
    for (const file of videoFiles) {
      await new Promise((resolve) => {
        getOrCreateVideoId(file, async (err, videoId) => {
          if (err) {
            console.error(`[Auto-Discovery] Error getting/creating ID for ${file}:`, err);
            return resolve();
          }
          
          const framesDir = path.join(__dirname, '../frames', String(videoId));
          let needsExtraction = true;
          
          if (fs.existsSync(framesDir)) {
            const frameFiles = fs.readdirSync(framesDir).filter(f => f.match(/\.jpe?g$/i));
            if (frameFiles.length > 0) {
              needsExtraction = false;
              // Ensure database has correct frame count
              db.run('UPDATE videos SET frame_count = ? WHERE id = ?', [frameFiles.length, videoId]);
              console.log(`[Auto-Discovery] Video "${file}" already indexed (ID: ${videoId}) with ${frameFiles.length} frames.`);
            }
          }
          
          if (needsExtraction) {
            console.log(`[Auto-Discovery] Extracting frames for video: "${file}" (ID: ${videoId})...`);
            const videoPath = path.join(videoSamplesDir, file);
            fs.mkdirSync(framesDir, { recursive: true });
            try {
              const count = await extractFrames(videoPath, framesDir, 1);
              db.run('UPDATE videos SET frame_count = ? WHERE id = ?', [count, videoId]);
              console.log(`[Auto-Discovery] Successfully extracted ${count} frames for video: "${file}"`);
            } catch (e) {
              console.error(`[Auto-Discovery] Frame extraction failed for ${file}:`, e);
            }
          }
          resolve();
        });
      });
    }
    console.log('[Auto-Discovery] Video initialization completed.');
  } catch (err) {
    console.error('[Auto-Discovery] Error scanning video samples:', err);
  }
}



// Helper to get video ID (create if new)
function getOrCreateVideoId(filename, callback) {
  db.get('SELECT id FROM videos WHERE filename = ?', [filename], (err, row) => {
    if (err) return callback(err);
    if (row) return callback(null, row.id);
    // Insert placeholder, frame_count will be updated after extraction
    db.run('INSERT INTO videos (filename, frame_count) VALUES (?, ?)', [filename, 0], function (err) {
      if (err) return callback(err);
      callback(null, this.lastID);
    });
  });
}



// Route: list videos
app.get('/videos', (req, res) => {
  db.all('SELECT id, filename, frame_count FROM videos', [], (err, rows) => {
    if (err) return res.status(500).json({ error: err.message });
    res.json(rows);
  });
});

// Route: get random frame name for a video
app.get('/random-frame/:videoId', (req, res) => {
  const { videoId } = req.params;
  const framesPath = path.join(__dirname, '../frames', String(videoId));
  fs.readdir(framesPath, (err, files) => {
    if (err) return res.status(404).json({ error: 'Frames not found' });
    const imageFiles = files.filter(f => f.match(/\.jpe?g$/i));
    if (!imageFiles.length) return res.status(404).json({ error: 'No images' });
    const random = imageFiles[Math.floor(Math.random() * imageFiles.length)];
    res.json({ frameName: random });
  });
});

// Serve frame image
app.get('/frame/:videoId/:frameName', (req, res) => {
  const { videoId, frameName } = req.params;
  const filePath = path.join(__dirname, '../frames', String(videoId), frameName);
  res.sendFile(filePath);
});

// Submit annotations
app.post('/annotations', (req, res) => {
  const { videoId, frameName, points } = req.body; // points: [{x, y}]
  if (!videoId || !frameName || !Array.isArray(points)) {
    return res.status(400).json({ error: 'Invalid payload' });
  }
  const stmt = db.prepare('INSERT INTO annotations (video_id, frame_name, x, y) VALUES (?, ?, ?, ?)');
  db.serialize(() => {
    points.forEach(p => {
      stmt.run(videoId, frameName, p.x, p.y);
    });
    stmt.finalize(err => {
      if (err) return res.status(500).json({ error: err.message });
      res.json({ status: 'ok' });
    });
  });
});

// Get annotations for a specific frame (for overlay)
app.get('/annotations/:videoId/:frameName', (req, res) => {
  const { videoId, frameName } = req.params;
  db.all('SELECT x, y FROM annotations WHERE video_id = ? AND frame_name = ?', [videoId, frameName], (err, rows) => {
    if (err) return res.status(500).json({ error: err.message });
    res.json(rows);
  });
});

// Metrics endpoint
app.get('/metrics/:videoId', (req, res) => {
  const { videoId } = req.params;
  db.get('SELECT COUNT(*) AS annotatedFrames FROM (SELECT DISTINCT frame_name FROM annotations WHERE video_id = ?) ', [videoId], (err, annotatedRow) => {
    if (err) return res.status(500).json({ error: err.message });
    db.get('SELECT frame_count FROM videos WHERE id = ?', [videoId], (err, videoRow) => {
      if (err) return res.status(500).json({ error: err.message });
      db.get('SELECT COUNT(*) AS totalPoints FROM annotations WHERE video_id = ?', [videoId], (err, pointsRow) => {
        if (err) return res.status(500).json({ error: err.message });
        res.json({
          totalFrames: videoRow ? videoRow.frame_count : 0,
          annotatedFrames: annotatedRow.annotatedFrames,
          totalPoints: pointsRow.totalPoints
        });
      });
    });
  });
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => console.log(`Server listening on port ${PORT}`));
