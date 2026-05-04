let originalImg = new Image();
let currentParams = {};
let canvases = [
    document.getElementById('canvas1'),
    document.getElementById('canvas2'),
    document.getElementById('canvas3'),
    document.getElementById('canvas4')
];
let ctxs = canvases.map(c => c.getContext('2d', { willReadFrequently: true }));

const profiles = {
    default: { threshold_value: 128, morph_repair_iterations: 1, kmeans_k: 1, shape_filter_min_area: 50, target_area: 450, max_normal_area: 800, solidity_threshold: 45, aspect_ratio_threshold: 25, merge_distance: 30 },
    aggressive: { threshold_value: 110, morph_repair_iterations: 2, kmeans_k: 1, shape_filter_min_area: 30, target_area: 350, max_normal_area: 1500, solidity_threshold: 30, aspect_ratio_threshold: 35, merge_distance: 20 },
    lite: { threshold_value: 140, morph_repair_iterations: 0, kmeans_k: 1, shape_filter_min_area: 80, target_area: 500, max_normal_area: 600, solidity_threshold: 60, aspect_ratio_threshold: 20, merge_distance: 40 }
};

// Initialize
window.onload = async () => {
    await loadParams();
    refreshFrame();
    setupSliders();
};

function setupSliders() {
    document.querySelectorAll('input[type="range"]').forEach(slider => {
        const update = () => {
            const val = slider.value;
            slider.parentElement.querySelector('span').textContent = slider.id === 'solidity_threshold' || slider.id === 'aspect_ratio_threshold' ? (val / 100).toFixed(2) : val;
            process();
        };
        slider.oninput = update;
        // Initial value set in loadParams
    });
}

async function loadParams() {
    const res = await fetch('/api/params');
    const params = await res.json();
    currentParams = params;
    
    // Default threshold if not in JSON (it wasn't in the Go version)
    if (!currentParams.threshold_value) currentParams.threshold_value = 128;

    for (const key in currentParams) {
        const el = document.getElementById(key);
        if (el) {
            if (key === 'solidity_threshold' || key === 'aspect_ratio_threshold') {
                el.value = currentParams[key] * 100;
            } else {
                el.value = currentParams[key];
            }
            el.parentElement.querySelector('span').textContent = currentParams[key];
        }
    }
}

function loadProfile(p) {
    document.querySelectorAll('.profile-btn').forEach(b => b.classList.remove('active'));
    event.target.classList.add('active');
    
    const profile = profiles[p];
    for (const key in profile) {
        const el = document.getElementById(key);
        if (el) {
            el.value = key === 'solidity_threshold' || key === 'aspect_ratio_threshold' ? profile[key] : profile[key];
            el.parentElement.querySelector('span').textContent = key === 'solidity_threshold' || key === 'aspect_ratio_threshold' ? (profile[key]/100).toFixed(2) : profile[key];
        }
    }
    process();
}

async function saveParams() {
    const params = {};
    document.querySelectorAll('input[type="range"]').forEach(s => {
        const val = parseFloat(s.value);
        params[s.id] = s.id === 'solidity_threshold' || s.id === 'aspect_ratio_threshold' ? val / 100 : val;
    });
    
    const res = await fetch('/api/params', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params)
    });
    if (res.ok) alert('Settings saved to Go project!');
}

function refreshFrame() {
    const time = document.getElementById('videoTime').value;
    originalImg.src = `/api/frame?time=${time}&t=${Date.now()}`;
    originalImg.onload = () => {
        canvases.forEach(c => { c.width = originalImg.width; c.height = originalImg.height; });
        process();
    };
}

function process() {
    if (!originalImg.complete) return;

    const w = originalImg.width;
    const h = originalImg.height;
    
    // Get parameters from sliders
    const params = {};
    document.querySelectorAll('input[type="range"]').forEach(s => {
        const val = parseFloat(s.value);
        params[s.id] = s.id === 'solidity_threshold' || s.id === 'aspect_ratio_threshold' ? val / 100 : val;
    });

    // Step 1: Preprocessing (Threshold + Morph Repair)
    const ctx1 = ctxs[0];
    ctx1.drawImage(originalImg, 0, 0);
    let imageData = ctx1.getImageData(0, 0, w, h);
    let data = imageData.data;

    // Binary Threshold
    for (let i = 0; i < data.length; i += 4) {
        const avg = (data[i] + data[i+1] + data[i+2]) / 3;
        const val = avg < params.threshold_value ? 255 : 0; // Inverse like Go code
        data[i] = data[i+1] = data[i+2] = val;
    }
    
    // Morph Repair (Simple Iterative Box filter based Dilate/Erode)
    for (let it = 0; it < params.morph_repair_iterations; it++) {
        data = dilate(data, w, h);
        data = erode(data, w, h);
        data = erode(data, w, h);
        data = dilate(data, w, h);
    }
    
    imageData.data.set(data);
    ctx1.putImageData(imageData, 0, 0);

    // Step 2: Core Segmentation (Find Blobs)
    const blobs = findBlobs(data, w, h, params.shape_filter_min_area);
    const ctx2 = ctxs[1];
    ctx2.clearRect(0, 0, w, h);
    drawBlobs(ctx2, blobs);

    // Step 3: Splitting
    const splitBlobs = [];
    blobs.forEach(b => {
        let count = 1;
        if (b.area > params.max_normal_area) count = Math.max(count, Math.round(b.area / params.target_area));
        if (b.solidity < params.solidity_threshold) count = Math.max(count, 2);
        const ratio = (b.bbox.x2 - b.bbox.x1) / (b.bbox.y2 - b.bbox.y1);
        if (ratio > params.aspect_ratio_threshold || ratio < 1/params.aspect_ratio_threshold) count = Math.max(count, 2);
        
        for (let i = 0; i < count; i++) splitBlobs.push(b);
    });
    const ctx3 = ctxs[2];
    ctx3.clearRect(0, 0, w, h);
    drawBlobs(ctx3, splitBlobs);

    // Step 4: Validation (Neighbor Merge)
    const validBlobs = [];
    const merged = new Array(splitBlobs.length).fill(false);
    for (let i = 0; i < splitBlobs.length; i++) {
        if (merged[i]) continue;
        const b1 = splitBlobs[i];
        for (let j = i + 1; j < splitBlobs.length; j++) {
            if (merged[j]) continue;
            const b2 = splitBlobs[j];
            const dist = Math.sqrt((b1.cx - b2.cx)**2 + (b1.cy - b2.cy)**2);
            if (dist < params.merge_distance) merged[j] = true;
        }
        validBlobs.push(b1);
    }
    const ctx4 = ctxs[3];
    ctx4.clearRect(0, 0, w, h);
    drawBlobs(ctx4, validBlobs, true); // Show IDs

    document.getElementById('blobCount').textContent = validBlobs.length;
}

// --- Image Processing Helpers ---

function dilate(data, w, h) {
    const out = new Uint8ClampedArray(data.length);
    for (let y = 1; y < h - 1; y++) {
        for (let x = 1; x < w - 1; x++) {
            const idx = (y * w + x) * 4;
            let max = 0;
            for (let dy = -1; dy <= 1; dy++) {
                for (let dx = -1; dx <= 1; dx++) {
                    max = Math.max(max, data[((y + dy) * w + (x + dx)) * 4]);
                }
            }
            out[idx] = out[idx+1] = out[idx+2] = max;
            out[idx+3] = 255;
        }
    }
    return out;
}

function erode(data, w, h) {
    const out = new Uint8ClampedArray(data.length);
    for (let y = 1; y < h - 1; y++) {
        for (let x = 1; x < w - 1; x++) {
            const idx = (y * w + x) * 4;
            let min = 255;
            for (let dy = -1; dy <= 1; dy++) {
                for (let dx = -1; dx <= 1; dx++) {
                    min = Math.min(min, data[((y + dy) * w + (x + dx)) * 4]);
                }
            }
            out[idx] = out[idx+1] = out[idx+2] = min;
            out[idx+3] = 255;
        }
    }
    return out;
}

function findBlobs(data, w, h, minArea) {
    const visited = new Uint8Array(w * h);
    const blobs = [];
    
    for (let y = 0; y < h; y++) {
        for (let x = 0; x < w; x++) {
            const idx = y * w + x;
            if (data[idx * 4] === 255 && !visited[idx]) {
                const points = [];
                const stack = [[x, y]];
                visited[idx] = 1;
                
                let minX = x, maxX = x, minY = y, maxY = y;
                let sumX = 0, sumY = 0;

                while (stack.length > 0) {
                    const [px, py] = stack.pop();
                    points.push([px, py]);
                    sumX += px; sumY += py;
                    if (px < minX) minX = px; if (px > maxX) maxX = px;
                    if (py < minY) minY = py; if (py > maxY) maxY = py;

                    const neighbors = [[px+1, py], [px-1, py], [px, py+1], [px, py-1]];
                    for (const [nx, ny] of neighbors) {
                        if (nx >= 0 && nx < w && ny >= 0 && ny < h) {
                            const nidx = ny * w + nx;
                            if (data[nidx * 4] === 255 && !visited[nidx]) {
                                visited[nidx] = 1;
                                stack.push([nx, ny]);
                            }
                        }
                    }
                }
                
                if (points.length >= minArea) {
                    const bboxArea = (maxX - minX + 1) * (maxY - minY + 1);
                    blobs.push({
                        points,
                        area: points.length,
                        cx: sumX / points.length,
                        cy: sumY / points.length,
                        bbox: { x1: minX, y1: minY, x2: maxX, y2: maxY },
                        solidity: points.length / bboxArea
                    });
                }
            }
        }
    }
    return blobs;
}

function drawBlobs(ctx, blobs, showIDs = false) {
    blobs.forEach((b, i) => {
        const hue = (i * 137.5) % 360;
        ctx.fillStyle = `hsla(${hue}, 70%, 50%, 0.8)`;
        b.points.forEach(p => {
            ctx.fillRect(p[0], p[1], 1, 1);
        });
        
        if (showIDs) {
            ctx.strokeStyle = 'white';
            ctx.lineWidth = 1;
            ctx.strokeRect(b.bbox.x1, b.bbox.y1, b.bbox.x2 - b.bbox.x1, b.bbox.y2 - b.bbox.y1);
            ctx.fillStyle = 'white';
            ctx.font = '10px Outfit';
            ctx.fillText(i+1, b.cx, b.cy);
        }
    });
}
