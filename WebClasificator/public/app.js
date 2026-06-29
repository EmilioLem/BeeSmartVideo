// public/app.js - AI Pre-Annotated Sequential Verification Workflow
document.addEventListener('DOMContentLoaded', () => {
  const videoSelect = document.getElementById('videoSelect');
  const frameCanvas = document.getElementById('frameCanvas');
  const ctx = frameCanvas.getContext('2d');
  
  const prevBtn = document.getElementById('prevBtn');
  const nextBtn = document.getElementById('nextBtn');
  const submitBtn = document.getElementById('submitBtn');

  const progressBar = document.getElementById('progressBar');
  const currentFrameNumSpan = document.getElementById('currentFrameNum');
  const totalFrameNumSpan = document.getElementById('totalFrameNum');
  const frameStatusBadge = document.getElementById('frameStatusBadge');

  const totalFramesSpan = document.getElementById('totalFrames');
  const checkedFramesSpan = document.getElementById('checkedFrames');
  const totalPointsSpan = document.getElementById('totalPoints');

  const frameNameDisplay = document.getElementById('frameNameDisplay');
  const pointCountDisplay = document.getElementById('pointCountDisplay');

  const dotSize = 25; // Visual circle radius for bee annotations

  let currentVideoId = null;
  let frames = [];
  let currentFrameIndex = 0;
  let points = [];
  let isCurrentFrameChecked = false;
  let currentImg = null;

  // Fetch list of videos
  function fetchVideos() {
    fetch('/videos')
      .then(res => res.json())
      .then(videoList => {
        videoSelect.innerHTML = '';
        if (videoList.length === 0) {
          const opt = document.createElement('option');
          opt.text = 'No hay videos cargados';
          opt.disabled = true;
          videoSelect.add(opt);
        } else {
          videoList.forEach(v => {
            const opt = document.createElement('option');
            opt.value = v.id;
            opt.text = v.filename;
            videoSelect.add(opt);
          });
          videoSelect.selectedIndex = 0;
          currentVideoId = videoSelect.value;
          loadVideo(currentVideoId);
        }
      })
      .catch(console.error);
  }

  // Load video metadata and frame list
  function loadVideo(videoId) {
    currentVideoId = videoId;
    updateMetrics();

    fetch(`/video/${videoId}/frames`)
      .then(res => res.json())
      .then(data => {
        frames = data.frames || [];
        currentFrameIndex = 0;
        if (frames.length > 0) {
          loadFrameByIndex(0);
        } else {
          frameNameDisplay.textContent = 'Sin frames disponibles';
        }
      })
      .catch(console.error);
  }

  // Load specific frame by 0-based index
  function loadFrameByIndex(index) {
    if (!currentVideoId || frames.length === 0) return;
    if (index < 0) index = 0;
    if (index >= frames.length) index = frames.length - 1;

    currentFrameIndex = index;
    const frameName = frames[currentFrameIndex];

    // Update progress elements
    currentFrameNumSpan.textContent = currentFrameIndex + 1;
    totalFrameNumSpan.textContent = frames.length;
    frameNameDisplay.textContent = frameName;
    
    const progressPercent = ((currentFrameIndex + 1) / frames.length) * 100;
    progressBar.style.width = `${progressPercent}%`;

    // Disable / enable navigation buttons
    prevBtn.disabled = (currentFrameIndex === 0);
    nextBtn.disabled = (currentFrameIndex === frames.length - 1);

    // Fetch frame metadata (user annotations, pre-annotations, checked status)
    fetch(`/video/${currentVideoId}/frame-data/${frameName}`)
      .then(res => res.json())
      .then(data => {
        isCurrentFrameChecked = data.isChecked;
        if (isCurrentFrameChecked) {
          frameStatusBadge.textContent = '✅ Revisado';
          frameStatusBadge.className = 'status-badge checked';
        } else {
          frameStatusBadge.textContent = '⏳ Pendiente de revisión';
          frameStatusBadge.className = 'status-badge pending';
        }

        // If user has saved points or frame is checked, use user points. Else use AI pre-annotations!
        if (isCurrentFrameChecked || (data.userPoints && data.userPoints.length > 0)) {
          points = [...data.userPoints];
        } else {
          points = [...data.prePoints];
        }

        // Load image onto canvas
        const img = new Image();
        img.onload = () => {
          currentImg = img;
          frameCanvas.width = img.width;
          frameCanvas.height = img.height;
          redrawCanvas();
        };
        img.src = `/frame/${currentVideoId}/${frameName}`;
      })
      .catch(console.error);
  }

  // Redraw image and annotations on canvas
  function redrawCanvas() {
    if (!currentImg) return;
    ctx.clearRect(0, 0, frameCanvas.width, frameCanvas.height);
    ctx.drawImage(currentImg, 0, 0);

    pointCountDisplay.textContent = `${points.length} abejas detectadas`;

    // Draw active points with high-visibility target styling
    points.forEach((p, i) => {
      // Outer glow circle
      ctx.fillStyle = 'rgba(255, 200, 0, 0.4)';
      ctx.beginPath();
      ctx.arc(p.x, p.y, dotSize * 1.3, 0, 2 * Math.PI);
      ctx.fill();

      // Main fill dot
      ctx.fillStyle = 'rgba(239, 68, 68, 0.85)'; // Red dot
      ctx.beginPath();
      ctx.arc(p.x, p.y, dotSize, 0, 2 * Math.PI);
      ctx.fill();

      // Center bright point
      ctx.fillStyle = '#ffffff';
      ctx.beginPath();
      ctx.arc(p.x, p.y, 4, 0, 2 * Math.PI);
      ctx.fill();
    });
  }

  // Handle interactive clicks on canvas to add or remove points
  frameCanvas.addEventListener('click', (e) => {
    if (!currentImg) return;
    const rect = frameCanvas.getBoundingClientRect();
    const scaleX = frameCanvas.width / rect.width;
    const scaleY = frameCanvas.height / rect.height;

    const x = (e.clientX - rect.left) * scaleX;
    const y = (e.clientY - rect.top) * scaleY;

    const clickMargin = 60; // Margin around existing dot to detect deletion
    const existingIndex = points.findIndex(
      p => Math.abs(p.x - x) <= clickMargin && Math.abs(p.y - y) <= clickMargin
    );

    if (existingIndex !== -1) {
      // Unclick / remove miscounted bee
      points.splice(existingIndex, 1);
    } else {
      // Click to add missing bee
      points.push({ x, y });
    }

    redrawCanvas();
  });

  // Save current frame annotations, mark checked, and advance to next frame
  function submitCurrentFrame() {
    if (!currentVideoId || frames.length === 0) return;
    const frameName = frames[currentFrameIndex];

    fetch('/annotations/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        videoId: currentVideoId,
        frameName: frameName,
        points: points
      })
    })
      .then(res => res.json())
      .then(() => {
        updateMetrics();
        if (currentFrameIndex < frames.length - 1) {
          loadFrameByIndex(currentFrameIndex + 1);
        } else {
          loadFrameByIndex(currentFrameIndex); // refresh current
        }
      })
      .catch(console.error);
  }

  // Update project metrics
  function updateMetrics() {
    if (!currentVideoId) return;
    fetch(`/metrics/${currentVideoId}`)
      .then(res => res.json())
      .then(data => {
        totalFramesSpan.textContent = data.totalFrames || 0;
        checkedFramesSpan.textContent = data.checkedFrames || 0;
        totalPointsSpan.textContent = data.totalPoints || 0;
      })
      .catch(console.error);
  }

  // Event Listeners
  submitBtn.addEventListener('click', submitCurrentFrame);
  prevBtn.addEventListener('click', () => loadFrameByIndex(currentFrameIndex - 1));
  nextBtn.addEventListener('click', () => loadFrameByIndex(currentFrameIndex + 1));

  videoSelect.addEventListener('change', () => {
    loadVideo(videoSelect.value);
  });

  // Keyboard navigation shortcuts
  document.addEventListener('keydown', (e) => {
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'SELECT') return;
    if (e.key === 'ArrowLeft') {
      loadFrameByIndex(currentFrameIndex - 1);
    } else if (e.key === 'ArrowRight') {
      loadFrameByIndex(currentFrameIndex + 1);
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      submitCurrentFrame();
    }
  });

  // Initial startup
  fetchVideos();
});
