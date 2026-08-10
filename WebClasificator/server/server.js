// server/server.js
const express = require('express');
const path = require('path');
const fs = require('fs');
const { execFile } = require('child_process');
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

// Ensure DB tables exist
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
CREATE TABLE IF NOT EXISTS pre_annotations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  video_id INTEGER NOT NULL,
  frame_name TEXT NOT NULL,
  x REAL NOT NULL,
  y REAL NOT NULL,
  FOREIGN KEY(video_id) REFERENCES videos(id)
);
CREATE TABLE IF NOT EXISTS checked_frames (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  video_id INTEGER NOT NULL,
  frame_name TEXT NOT NULL,
  checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(video_id, frame_name),
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

// Run Go CLI pre-annotation model for video frames
function runPreAnnotation(videoId, framesDir) {
  return new Promise((resolve) => {
    db.get('SELECT COUNT(*) as count FROM pre_annotations WHERE video_id = ?', [videoId], (err, row) => {
      if (err || (row && row.count > 0)) {
        return resolve();
      }
      console.log(`[Pre-Annotation] Running Go V1 model for video ID ${videoId}...`);
      const preannotateBin = path.join(__dirname, '../../preannotate');
      execFile(preannotateBin, ['-framesDir', framesDir], { maxBuffer: 50 * 1024 * 1024 }, (execErr, stdout, stderr) => {
        if (execErr) {
          console.error(`[Pre-Annotation] Failed for video ${videoId}:`, execErr, stderr);
          return resolve();
        }
        try {
          const results = JSON.parse(stdout);
          db.serialize(() => {
            const stmt = db.prepare('INSERT INTO pre_annotations (video_id, frame_name, x, y) VALUES (?, ?, ?, ?)');
            for (const item of results) {
              for (const pt of item.points) {
                stmt.run(videoId, item.frameName, pt.x, pt.y);
              }
            }
            stmt.finalize((finalErr) => {
              if (finalErr) console.error('[Pre-Annotation] DB insert error:', finalErr);
              else console.log(`[Pre-Annotation] Stored pre-annotations for video ID ${videoId}.`);
              resolve();
            });
          });
        } catch (parseErr) {
          console.error('[Pre-Annotation] JSON parse error:', parseErr);
          resolve();
        }
      });
    });
  });
}

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

          // Trigger Go pre-annotation pipeline
          await runPreAnnotation(videoId, framesDir);
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

// Route: get ordered list of frame names for a video
app.get('/video/:videoId/frames', (req, res) => {
  const { videoId } = req.params;
  const framesPath = path.join(__dirname, '../frames', String(videoId));
  fs.readdir(framesPath, (err, files) => {
    if (err) return res.status(404).json({ error: 'Frames not found' });
    const imageFiles = files.filter(f => f.match(/\.jpe?g$/i)).sort();
    res.json({ frames: imageFiles });
  });
});

// Serve frame image
app.get('/frame/:videoId/:frameName', (req, res) => {
  const { videoId, frameName } = req.params;
  const filePath = path.join(__dirname, '../frames', String(videoId), frameName);
  res.sendFile(filePath);
});

// Get detailed frame data (user annotations, pre-annotations, checked status)
app.get('/video/:videoId/frame-data/:frameName', (req, res) => {
  const { videoId, frameName } = req.params;

  db.get('SELECT 1 FROM checked_frames WHERE video_id = ? AND frame_name = ?', [videoId, frameName], (err, checkedRow) => {
    if (err) return res.status(500).json({ error: err.message });
    const isChecked = !!checkedRow;

    db.all('SELECT x, y FROM annotations WHERE video_id = ? AND frame_name = ?', [videoId, frameName], (err, userRows) => {
      if (err) return res.status(500).json({ error: err.message });

      db.all('SELECT x, y FROM pre_annotations WHERE video_id = ? AND frame_name = ?', [videoId, frameName], (err, preRows) => {
        if (err) return res.status(500).json({ error: err.message });

        res.json({
          frameName,
          isChecked,
          userPoints: userRows || [],
          prePoints: preRows || []
        });
      });
    });
  });
});

// Legacy support & quick query
app.get('/random-frame/:videoId', (req, res) => {
  const { videoId } = req.params;
  const framesPath = path.join(__dirname, '../frames', String(videoId));
  fs.readdir(framesPath, (err, files) => {
    if (err) return res.status(404).json({ error: 'Frames not found' });
    const imageFiles = files.filter(f => f.match(/\.jpe?g$/i)).sort();
    if (!imageFiles.length) return res.status(404).json({ error: 'No images' });
    res.json({ frameName: imageFiles[0] });
  });
});

// Submit/Check annotations for a frame
const handleSaveAnnotations = (req, res) => {
  const { videoId, frameName, points } = req.body; // points: [{x, y}]
  if (!videoId || !frameName || !Array.isArray(points)) {
    return res.status(400).json({ error: 'Invalid payload' });
  }

  db.serialize(() => {
    // 1. Clear existing user annotations for this frame
    db.run('DELETE FROM annotations WHERE video_id = ? AND frame_name = ?', [videoId, frameName]);

    // 2. Insert updated points
    const stmt = db.prepare('INSERT INTO annotations (video_id, frame_name, x, y) VALUES (?, ?, ?, ?)');
    points.forEach(p => {
      stmt.run(videoId, frameName, p.x, p.y);
    });
    stmt.finalize();

    // 3. Mark frame as checked
    db.run('INSERT OR REPLACE INTO checked_frames (video_id, frame_name) VALUES (?, ?)', [videoId, frameName], (err) => {
      if (err) return res.status(500).json({ error: err.message });
      res.json({ status: 'ok' });
    });
  });
};

app.post('/annotations', handleSaveAnnotations);
app.post('/annotations/submit', handleSaveAnnotations);

// Get annotations for overlay
app.get('/annotations/:videoId/:frameName', (req, res) => {
  const { videoId, frameName } = req.params;
  db.all('SELECT x, y FROM annotations WHERE video_id = ? AND frame_name = ?', [videoId, frameName], (err, rows) => {
    if (err) return res.status(500).json({ error: err.message });
    res.json(rows);
  });
});

// Updated metrics endpoint (Checked frames focus)
app.get('/metrics/:videoId', (req, res) => {
  const { videoId } = req.params;
  db.get('SELECT COUNT(*) AS checkedFrames FROM checked_frames WHERE video_id = ?', [videoId], (err, checkedRow) => {
    if (err) return res.status(500).json({ error: err.message });
    db.get('SELECT frame_count FROM videos WHERE id = ?', [videoId], (err, videoRow) => {
      if (err) return res.status(500).json({ error: err.message });
      db.get('SELECT COUNT(*) AS totalPoints FROM annotations WHERE video_id = ?', [videoId], (err, pointsRow) => {
        if (err) return res.status(500).json({ error: err.message });
        res.json({
          totalFrames: videoRow ? videoRow.frame_count : 0,
          checkedFrames: checkedRow ? checkedRow.checkedFrames : 0,
          totalPoints: pointsRow ? pointsRow.totalPoints : 0
        });
      });
    });
  });
});

// Helper to extract dimensions from JPEG headers natively
function getJpegSize(filePath) {
  const buffer = fs.readFileSync(filePath);
  if (buffer.length < 2 || buffer[0] !== 0xff || buffer[1] !== 0xd8) {
    throw new Error('Not a valid JPEG');
  }
  let i = 2;
  while (i < buffer.length - 8) {
    if (buffer[i] === 0xff) {
      const marker = buffer[i + 1];
      if (marker === 0xd8 || marker === 0xd9 || (marker >= 0xd0 && marker <= 0xd7)) {
        i += 2;
        continue;
      }
      if ((marker >= 0xc0 && marker <= 0xcf) && marker !== 0xc4 && marker !== 0xc8 && marker !== 0xcc) {
        const height = buffer.readUInt16BE(i + 5);
        const width = buffer.readUInt16BE(i + 7);
        return { width, height };
      }
      const len = buffer.readUInt16BE(i + 2);
      i += 2 + len;
    } else {
      i++;
    }
  }
  throw new Error('SOF marker not found');
}

// Route: Get AI statistics for checked frames of a video
app.get('/video/:videoId/statistics', (req, res) => {
  const { videoId } = req.params;
  
  db.all('SELECT frame_name FROM checked_frames WHERE video_id = ?', [videoId], (err, checkedRows) => {
    if (err) return res.status(500).json({ error: err.message });
    
    if (checkedRows.length === 0) {
      return res.json({
        totalChecked: 0,
        tp: 0,
        fp: 0,
        fn: 0,
        tn: 0,
        precision: 0,
        recall: 0,
        f1: 0,
        accuracy: 0
      });
    }
    
    const checkedFrameNames = checkedRows.map(r => r.frame_name);
    const placeholders = checkedFrameNames.map(() => '?').join(',');
    
    db.all(`SELECT frame_name, x, y FROM annotations WHERE video_id = ? AND frame_name IN (${placeholders})`, [videoId, ...checkedFrameNames], (err, annRows) => {
      if (err) return res.status(500).json({ error: err.message });
      
      db.all(`SELECT frame_name, x, y FROM pre_annotations WHERE video_id = ? AND frame_name IN (${placeholders})`, [videoId, ...checkedFrameNames], (err, preRows) => {
        if (err) return res.status(500).json({ error: err.message });
        
        const annByFrame = {};
        const preByFrame = {};
        for (const frameName of checkedFrameNames) {
          annByFrame[frameName] = [];
          preByFrame[frameName] = [];
        }
        for (const row of annRows) {
          if (annByFrame[row.frame_name]) {
            annByFrame[row.frame_name].push({ x: row.x, y: row.y });
          }
        }
        for (const row of preRows) {
          if (preByFrame[row.frame_name]) {
            preByFrame[row.frame_name].push({ x: row.x, y: row.y });
          }
        }
        
        let imgWidth = 800;
        let imgHeight = 600;
        try {
          const framesDir = path.join(__dirname, '../frames', String(videoId));
          const firstFrameName = checkedFrameNames[0];
          const firstFramePath = path.join(framesDir, firstFrameName);
          if (fs.existsSync(firstFramePath)) {
            const size = getJpegSize(firstFramePath);
            imgWidth = size.width;
            imgHeight = size.height;
          }
        } catch (e) {
          console.warn('[Stats] Failed to read JPEG dimensions, falling back to 800x600:', e);
        }
        
        const gridWidth = 80;
        const gridHeight = 80;
        const cols = Math.ceil(imgWidth / gridWidth);
        const rows = Math.ceil(imgHeight / gridHeight);
        const cellsPerFrame = cols * rows;
        
        let totalTP = 0;
        let totalFP = 0;
        let totalFN = 0;
        let totalTN = 0;
        
        const MATCH_THRESHOLD = 50; // Max distance to match a predicted point to a corrected point
        
        for (const frameName of checkedFrameNames) {
          const prePoints = preByFrame[frameName] || [];
          const userPoints = annByFrame[frameName] || [];
          
          let tp = 0;
          let fp = 0;
          let fn = 0;
          
          const matchedPre = new Set();
          const matchedUser = new Set();
          
          const pairs = [];
          for (let i = 0; i < prePoints.length; i++) {
            for (let j = 0; j < userPoints.length; j++) {
              const dx = prePoints[i].x - userPoints[j].x;
              const dy = prePoints[i].y - userPoints[j].y;
              const dist = Math.sqrt(dx * dx + dy * dy);
              pairs.push({ preIdx: i, userIdx: j, dist });
            }
          }
          
          pairs.sort((a, b) => a.dist - b.dist);
          
          for (const pair of pairs) {
            if (pair.dist > MATCH_THRESHOLD) break;
            if (!matchedPre.has(pair.preIdx) && !matchedUser.has(pair.userIdx)) {
              matchedPre.add(pair.preIdx);
              matchedUser.add(pair.userIdx);
              tp++;
            }
          }
          
          fp = prePoints.length - tp;
          fn = userPoints.length - tp;
          
          totalTP += tp;
          totalFP += fp;
          totalFN += fn;
          
          const occupiedCells = new Set();
          for (const p of prePoints) {
            const col = Math.floor(p.x / gridWidth);
            const row = Math.floor(p.y / gridHeight);
            occupiedCells.add(`${col},${row}`);
          }
          for (const u of userPoints) {
            const col = Math.floor(u.x / gridWidth);
            const row = Math.floor(u.y / gridHeight);
            occupiedCells.add(`${col},${row}`);
          }
          
          const tn = Math.max(0, cellsPerFrame - occupiedCells.size);
          totalTN += tn;
        }
        
        const precision = (totalTP + totalFP) > 0 ? (totalTP / (totalTP + totalFP)) : 0;
        const recall = (totalTP + totalFN) > 0 ? (totalTP / (totalTP + totalFN)) : 0;
        const f1 = (precision + recall) > 0 ? (2 * (precision * recall) / (precision + recall)) : 0;
        const accuracy = (totalTP + totalTN + totalFP + totalFN) > 0 ? ((totalTP + totalTN) / (totalTP + totalTN + totalFP + totalFN)) : 0;
        
        res.json({
          totalChecked: checkedFrameNames.length,
          tp: totalTP,
          fp: totalFP,
          fn: totalFN,
          tn: totalTN,
          precision,
          recall,
          f1,
          accuracy
        });
      });
    });
  });
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => console.log(`Server listening on port ${PORT}`));

