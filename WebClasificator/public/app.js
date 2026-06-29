// public/app.js
document.addEventListener('DOMContentLoaded', () => {
  const videoSelect = document.getElementById('videoSelect');
  const overlayToggle = document.getElementById('overlayToggle');
  const canvas = document.getElementById('frameCanvas');
  const ctx = canvas.getContext('2d');
  const nextBtn = document.getElementById('nextBtn');
  const submitBtn = document.getElementById('submitBtn');
  const totalFramesSpan = document.getElementById('totalFrames');
  const annotatedFramesSpan = document.getElementById('annotatedFrames');
  const totalPointsSpan = document.getElementById('totalPoints');


  const dotSize = 25; // 5x the original size of 5

  let currentVideoId = null;
  let currentFrameName = null;
  let points = [];
  let communityPoints = []; // Cached community annotations
  let currentImg = null;    // Stored current image for instant redrawing

  // Helper to load image onto canvas
  function loadFrame() {
    if (!currentVideoId) return;
    fetch(`/random-frame/${currentVideoId}`)
      .then(res => res.json())
      .then(data => {
        currentFrameName = data.frameName;
        // Fetch community annotations for this frame and cache them
        fetch(`/annotations/${currentVideoId}/${currentFrameName}`)
          .then(res => res.json())
          .then(commData => {
            communityPoints = commData || [];

            const img = new Image();
            img.onload = () => {
              currentImg = img;
              // Set canvas dimensions to match the physical image resolution
              canvas.width = img.width;
              canvas.height = img.height;
              points = [];
              redrawCanvas();
            };
            img.src = `/frame/${currentVideoId}/${currentFrameName}`;
          })
          .catch(console.error);
      })
      .catch(console.error);
  }

  // Redraw the canvas content (image, community points, and active user points)
  function redrawCanvas() {
    if (!currentImg) return;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(currentImg, 0, 0);

    // Draw community points (blue dots) if overlay is toggled
    if (overlayToggle.checked) {
      ctx.fillStyle = 'rgba(0,0,255,0.5)';
      communityPoints.forEach(p => {
        ctx.beginPath();
        ctx.arc(p.x, p.y, dotSize, 0, 2 * Math.PI);
        ctx.fill();
      });
    }

    // Draw active user points (red dots)
    ctx.fillStyle = 'rgba(255,0,0,0.7)';
    points.forEach(p => {
      ctx.beginPath();
      ctx.arc(p.x, p.y, dotSize, 0, 2 * Math.PI);
      ctx.fill();
    });
  }

  // Canvas click handler (handles scaling for CSS styling and adding/removing points)
  canvas.addEventListener('click', (e) => {
    const rect = canvas.getBoundingClientRect();

    // Scale client/display coordinates to match the canvas's physical/buffer resolution
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;
    const x = (e.clientX - rect.left) * scaleX;
    const y = (e.clientY - rect.top) * scaleY;

    // Check if we clicked within a 60px bounding box around an existing point to remove it
    const clickMargin = 60;
    const existingIndex = points.findIndex(
      p => Math.abs(p.x - x) <= clickMargin && Math.abs(p.y - y) <= clickMargin
    );

    if (existingIndex !== -1) {
      // Remove point if clicked again
      points.splice(existingIndex, 1);
    } else {
      // Otherwise, place a new point
      points.push({ x, y });
    }

    redrawCanvas();
  });

  // Submit annotations
  submitBtn.addEventListener('click', () => {
    if (!currentVideoId || !currentFrameName || points.length === 0) return;
    fetch('/annotations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ videoId: currentVideoId, frameName: currentFrameName, points })
    })
      .then(res => res.json())
      .then(() => {
        loadFrame();
        updateMetrics();
      })
      .catch(console.error);
  });

  // Next button (skip without saving)
  nextBtn.addEventListener('click', loadFrame);

  // Overlay toggle
  overlayToggle.addEventListener('change', redrawCanvas);


  // Fetch list of videos and populate selector
  function fetchVideos() {
    fetch('/videos')
      .then(res => res.json())
      .then(videos => {
        videoSelect.innerHTML = '';
        if (videos.length === 0) {
          const opt = document.createElement('option');
          opt.text = 'No videos uploaded';
          opt.disabled = true;
          videoSelect.add(opt);
        } else {
          videos.forEach(v => {
            const opt = document.createElement('option');
            opt.value = v.id;
            opt.text = v.filename;
            videoSelect.add(opt);
          });
          // auto-select first video
          videoSelect.selectedIndex = 0;
          currentVideoId = videoSelect.value;
          updateMetrics();
          loadFrame();
        }
      })
      .catch(console.error);
  }

  // Update metrics display
  function updateMetrics() {
    if (!currentVideoId) return;
    fetch(`/metrics/${currentVideoId}`)
      .then(res => res.json())
      .then(data => {
        totalFramesSpan.textContent = data.totalFrames || 0;
        annotatedFramesSpan.textContent = data.annotatedFrames || 0;
        totalPointsSpan.textContent = data.totalPoints || 0;
      })
      .catch(console.error);
  }

  // When video selection changes
  videoSelect.addEventListener('change', () => {
    currentVideoId = videoSelect.value;
    updateMetrics();
    loadFrame();
  });

  // Initial load
  fetchVideos();
});
