// utils/ffmpeg.js
const ffmpeg = require('fluent-ffmpeg');
const path = require('path');
const fs = require('fs');

/**
 * Extract frames from a video file.
 * @param {string} videoPath - Absolute path to the source video.
 * @param {string} outputDir - Directory where extracted frames will be saved.
 * @param {number} fps - Frames per second to extract (default 1).
 * @returns {Promise<number>} Resolves with the number of frames extracted.
 */
function extractFrames(videoPath, outputDir, fps = 1) {
  return new Promise((resolve, reject) => {
    // Ensure output directory exists
    fs.mkdirSync(outputDir, { recursive: true });
    let frameCount = 0;
    ffmpeg(videoPath)
      .outputOptions([`-vf fps=${fps}`])
      .output(path.join(outputDir, '%04d.jpg'))
      .on('error', (err) => {
        reject(err);
      })
      .on('end', () => {
        // Count generated files
        fs.readdir(outputDir, (err, files) => {
          if (err) return reject(err);
          const jpgFiles = files.filter(f => f.match(/\.jpe?g$/i));
          resolve(jpgFiles.length);
        });
      })
      .run();
  });
}

module.exports = { extractFrames };
