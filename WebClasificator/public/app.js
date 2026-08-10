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
    updateStats();

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
        updateStats();
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

  // Modal elements
  const statsModal = document.getElementById('statsModal');
  const closeStatsModal = document.getElementById('closeStatsModal');
  const closeStatsModalBtn = document.getElementById('closeStatsModalBtn');
  const downloadStatsBtn = document.getElementById('downloadStatsBtn');
  let currentStatsData = null;

  function openStatsModal(data) {
    currentStatsData = data;
    
    document.getElementById('modalAccuracy').textContent = `${(data.accuracy * 100).toFixed(1)}%`;
    document.getElementById('modalPrecision').textContent = `${(data.precision * 100).toFixed(1)}%`;
    document.getElementById('modalRecall').textContent = `${(data.recall * 100).toFixed(1)}%`;
    document.getElementById('modalF1').textContent = data.f1.toFixed(3);
    
    document.getElementById('matrixTP').textContent = data.tp;
    document.getElementById('matrixFP').textContent = data.fp;
    document.getElementById('matrixFN').textContent = data.fn;
    document.getElementById('matrixTN').textContent = data.tn;
    
    statsModal.classList.add('show');
  }

  function hideStatsModal() {
    statsModal.classList.remove('show');
  }

  if (closeStatsModal) closeStatsModal.addEventListener('click', hideStatsModal);
  if (closeStatsModalBtn) closeStatsModalBtn.addEventListener('click', hideStatsModal);
  window.addEventListener('click', (e) => {
    if (e.target === statsModal) {
      hideStatsModal();
    }
  });

  if (downloadStatsBtn) {
    downloadStatsBtn.addEventListener('click', () => {
      if (!currentStatsData || !currentVideoId) return;
      
      const videoName = videoSelect.options[videoSelect.selectedIndex]?.text || 'unknown';
      const report = {
        video_id: currentVideoId,
        video_filename: videoName,
        date: new Date().toISOString(),
        metrics: {
          checked_frames: currentStatsData.totalChecked,
          accuracy: currentStatsData.accuracy,
          precision: currentStatsData.precision,
          recall: currentStatsData.recall,
          f1_score: currentStatsData.f1
        },
        confusion_matrix: {
          true_positives: currentStatsData.tp,
          false_positives: currentStatsData.fp,
          false_negatives: currentStatsData.fn,
          true_negatives: currentStatsData.tn
        }
      };
      
      const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(JSON.stringify(report, null, 2));
      const downloadAnchor = document.createElement('a');
      downloadAnchor.setAttribute("href", dataStr);
      downloadAnchor.setAttribute("download", `AI_Evaluation_${videoName.replace(/\.[^/.]+$/, "")}.json`);
      document.body.appendChild(downloadAnchor);
      downloadAnchor.click();
      downloadAnchor.remove();
    });
  }

  // Update AI statistics
  function updateStats() {
    if (!currentVideoId) return;
    const aiStatsCard = document.getElementById('aiStatsCard');
    const aiStatsContent = document.getElementById('aiStatsContent');
    
    fetch(`/video/${currentVideoId}/statistics`)
      .then(res => res.json())
      .then(data => {
        if (data.totalChecked === 0) {
          aiStatsContent.innerHTML = '<p class="stats-loading">Revisa al menos un frame para ver las estadísticas...</p>';
          aiStatsCard.classList.remove('completed');
          return;
        }
        
        const accuracyPct = (data.accuracy * 100).toFixed(1);
        const precisionPct = (data.precision * 100).toFixed(1);
        const recallPct = (data.recall * 100).toFixed(1);
        const f1Val = data.f1.toFixed(2);
        
        aiStatsContent.innerHTML = `
          <div class="stats-summary">
            <div class="stat-row"><span>Exactitud:</span> <span class="stat-val">${accuracyPct}%</span></div>
            <div class="stat-row"><span>Precisión:</span> <span class="stat-val">${precisionPct}%</span></div>
            <div class="stat-row"><span>Sensibilidad:</span> <span class="stat-val">${recallPct}%</span></div>
            <div class="stat-row"><span>F1-Score:</span> <span class="stat-val">${f1Val}</span></div>
            <button class="btn-view-report" id="viewReportBtn">📊 Ver Informe Detallado</button>
          </div>
        `;
        
        document.getElementById('viewReportBtn').addEventListener('click', () => {
          openStatsModal(data);
        });

        // Highlight if completed
        fetch(`/metrics/${currentVideoId}`)
          .then(mRes => mRes.json())
          .then(mData => {
            if (mData.checkedFrames > 0 && mData.checkedFrames === mData.totalFrames) {
              aiStatsCard.classList.add('completed');
            } else {
              aiStatsCard.classList.remove('completed');
            }
          })
          .catch(console.error);
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
