const express = require('express');
const ffmpeg = require('fluent-ffmpeg');
const path = require('path');
const fs = require('fs');

const app = express();
const port = 3001;

app.use(express.static('public'));
app.use(express.json());

const VIDEO_PATH = path.join(__dirname, '../videoSamples/1.5minBeesCrossing.mp4');
const PARAMS_PATH = path.join(__dirname, '../4steps_v1_params.json');

// Get a frame from the video at a specific time
app.get('/api/frame', (req, res) => {
    const time = req.query.time || 10; // default to 10 seconds
    const framePath = path.join(__dirname, 'temp_frame.jpg');

    ffmpeg(VIDEO_PATH)
        .screenshots({
            timestamps: [time],
            filename: 'temp_frame.jpg',
            folder: __dirname,
            size: '426x240'
        })
        .on('end', () => {
            res.sendFile(framePath);
        })
        .on('error', (err) => {
            console.error('FFmpeg error:', err);
            res.status(500).send('Error extracting frame');
        });
});

// Load parameters
app.get('/api/params', (req, res) => {
    fs.readFile(PARAMS_PATH, 'utf8', (err, data) => {
        if (err) return res.status(500).send('Error reading params');
        // Simple regex to strip comments for the web tool (it will save clean JSON)
        const cleanData = data.replace(/\/\/.*$/gm, '').replace(/\/\*[\s\S]*?\*\//g, '');
        try {
            res.json(JSON.parse(cleanData));
        } catch (e) {
            res.status(500).send('Error parsing params: ' + e.message);
        }
    });
});

// Save parameters
app.post('/api/params', (req, res) => {
    const params = req.body;
    fs.writeFile(PARAMS_PATH, JSON.stringify(params, null, 2), (err) => {
        if (err) return res.status(500).send('Error saving params');
        res.send('Params saved successfully');
    });
});

app.listen(port, () => {
    console.log(`Tuning tool listening at http://localhost:${port}`);
});
