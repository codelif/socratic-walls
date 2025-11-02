(function () {
  const $ = (s) => document.querySelector(s);

  const API_BASE = "http://localhost:8080";

  const img = $('#fractalImage');
  const imageContainer = document.querySelector('.image');
  const previewLayer = $('#previewLayer');
  const resetBtn = $('.reset_nav');
  const zoomInBtn = $('.zoomin');
  const zoomOutBtn = $('.zoomout');
  const randomBtn = $('.random_view');
  const overlay = $('#infoOverlay');
  const renderModeSelect = $('#select_cm');
  const backendPalettePanel = $('#backendPalettePanel');
  const backendPaletteList = $('#backendPaletteList');
  const addRandomPaletteBtn = $('#addRandomPalette');

  const saveToggle = document.querySelector('.save-toggle');
  const saveMenu = $('#saveMenu');

  const view = {
    x: -0.5,
    y: 0.0,
    scale: 3.5,
  };

  let currentColorMode = 4; // histogram
  let backendPalettes = [];
  let selectedPaletteIndex = 0;

  let requestCounter = 0;
  let lastObjectURL = null;

  function getImgRect() {
    return img.getBoundingClientRect();
  }
  function getImageContainerRect() {
    return imageContainer.getBoundingClientRect();
  }

  function paletteModeUsesPalette(mode) {
    return mode === 1 || mode === 4;
  }

  // preview follows main mode but histogram -> long gradient
  function effectivePreviewMode() {
    if (currentColorMode === 4) return 1;
    return currentColorMode;
  }

  function buildMandelURL({ cx, cy, scale, width, height, mode, palette }) {
    const u = new URL(API_BASE + "/api/mandel");
    u.searchParams.set("width", width);
    u.searchParams.set("height", height);
    u.searchParams.set("cx", cx);
    u.searchParams.set("cy", cy);
    u.searchParams.set("scale", scale);
    u.searchParams.set("mode", mode);
    if (typeof palette === "number") {
      u.searchParams.set("palette", palette);
    }
    return u.toString();
  }

  function updateOverlay() {
    if (!overlay) return;
    overlay.textContent = `x:${view.x.toFixed(6)} y:${view.y.toFixed(6)} scale:${view.scale.toExponential(2)}`;
  }

  function clientToComplex(clientX, clientY) {
    const rect = getImgRect();
    const cx = clientX - rect.left;
    const cy = clientY - rect.top;
    const w = rect.width;
    const h = rect.height;
    const aspect = h / w;
    const left = view.x - view.scale / 2;
    const top = view.y - (view.scale * aspect) / 2;
    const dx = view.scale / w;
    const dy = (view.scale * aspect) / h;
    return {
      x: left + cx * dx,
      y: top + cy * dy,
    };
  }

  async function fetchAndShowImage(opts = {}) {
    const { hidePreviewWhenDone = false } = opts;
    const width = Math.floor(img.clientWidth || 1080);
    const height = Math.floor(img.clientHeight || 660);
    const id = ++requestCounter;

    const url = buildMandelURL({
      cx: view.x,
      cy: view.y,
      scale: view.scale,
      width,
      height,
      mode: currentColorMode,
      palette: paletteModeUsesPalette(currentColorMode) ? selectedPaletteIndex : undefined,
    });

    try {
      const res = await fetch(url);
      if (!res.ok) return;
      const blob = await res.blob();
      if (id !== requestCounter) return;

      const objectURL = URL.createObjectURL(blob);
      if (lastObjectURL) URL.revokeObjectURL(lastObjectURL);
      lastObjectURL = objectURL;
      img.src = objectURL;
      updateOverlay();

      if (hidePreviewWhenDone && previewLayer) {
        previewLayer.style.display = "none";
        previewLayer.innerHTML = "";
      }
    } catch (err) {
      console.error(err);
      if (hidePreviewWhenDone && previewLayer) {
        previewLayer.style.display = "none";
        previewLayer.innerHTML = "";
      }
    }
  }

  async function downloadAtResolution(w, h) {
    // build JSON body so we can send a custom palette later if needed
    const body = {
      width: w,
      height: h,
      cx: view.x,
      cy: view.y,
      scale: view.scale,
      mode: currentColorMode,
    };

    // if current mode uses palette, send that palette by index as an actual palette
    // the backend POST handler expects a full gradient; normally we only have the index
    // so: if we have the actual palette object in backendPalettes, send it
    if (paletteModeUsesPalette(currentColorMode)) {
      const p = backendPalettes[selectedPaletteIndex];
      if (p) {
        body.palette = p;
      }
    }

    const res = await fetch(API_BASE + "/api/mandel", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      console.warn("download failed");
      return;
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `mandelbrot_${w}x${h}.png`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 2000);
  }

  async function loadRandomView() {
    try {
      const res = await fetch(API_BASE + "/api/view/random");
      if (!res.ok) throw new Error("random failed");
      const v = await res.json();
      view.x = v.cx;
      view.y = v.cy;
      view.scale = v.scale;
      updateOverlay();
      fetchAndShowImage();
    } catch (e) {
      console.warn("random view fallback:", e);
      fetchAndShowImage();
    }
  }

  async function loadPalettesFromBackend() {
    try {
      const res = await fetch(API_BASE + "/api/palettes");
      if (!res.ok) return;
      const data = await res.json();
      backendPalettes = data || [];
      renderBackendPalettes();
    } catch (e) {
      console.warn("cannot load palettes", e);
    }
  }

  function renderBackendPalettes() {
    if (!backendPalettePanel || !backendPaletteList) return;

    if (!paletteModeUsesPalette(currentColorMode)) {
      backendPalettePanel.style.display = "none";
      return;
    }

    backendPalettePanel.style.display = "block";
    backendPaletteList.innerHTML = "";

    backendPalettes.forEach((p, idx) => {
      const item = document.createElement("div");
      item.className = "palette-item";
      if (idx === selectedPaletteIndex) item.classList.add("selected");

      const sw = document.createElement("div");
      sw.className = "palette-swatch";

      if (p.stops && p.stops.length > 0) {
        const parts = [];
        const n = p.stops.length;
        for (let i = 0; i < n; i++) {
          const s = p.stops[i];
          const col = s.c || s.C || s.color || s;
          const r = col.R ?? col.r ?? 0;
          const g = col.G ?? col.g ?? 0;
          const b = col.B ?? col.b ?? 0;
          const pctStart = (i / n) * 100;
          const pctEnd = ((i + 1) / n) * 100;
          parts.push(`rgb(${r},${g},${b}) ${pctStart}% ${pctEnd}%`);
        }
        sw.style.background = `linear-gradient(to right, ${parts.join(",")})`;
      }

      const nm = document.createElement("div");
      nm.className = "palette-name";
      nm.textContent = p.name || `palette ${idx + 1}`;

      item.appendChild(sw);
      item.appendChild(nm);

      item.addEventListener("click", () => {
        selectedPaletteIndex = idx;
        renderBackendPalettes();
        fetchAndShowImage();
      });

      backendPaletteList.appendChild(item);
    });
  }

  // ---------------- preview/pan ----------------
  let dragging = false;
  let dragStartX = 0;
  let dragStartY = 0;
  let currentRect = null;
  let previewInner = null;

  function createPreviewGrid() {
    const imgRect = getImgRect();
    const containerRect = getImageContainerRect();
    const offsetLeft = imgRect.left - containerRect.left;
    const offsetTop = imgRect.top - containerRect.top;

    previewLayer.style.display = "block";
    previewLayer.style.left = offsetLeft + "px";
    previewLayer.style.top = offsetTop + "px";
    previewLayer.style.width = imgRect.width + "px";
    previewLayer.style.height = imgRect.height + "px";
    previewLayer.style.borderRadius = window.getComputedStyle(img).borderRadius || "20px";
    previewLayer.style.overflow = "hidden";
    previewLayer.style.background = "#060612";

    previewLayer.innerHTML = "";

    previewInner = document.createElement("div");
    previewInner.className = "preview-layer-inner";
    previewLayer.appendChild(previewInner);

    const rect = { width: imgRect.width, height: imgRect.height };
    currentRect = rect;

    const w = Math.floor(rect.width / 2);
    const h = Math.floor(rect.height / 2);
    const aspect = rect.height / rect.width;

    const cols = [-1, 0, 1];
    const rows = [-1, 0, 1];

    const dxComplex = view.scale;
    const dyComplex = view.scale * aspect;

    const previewMode = effectivePreviewMode();
    const previewNeedsPalette = paletteModeUsesPalette(previewMode);

    rows.forEach((ry) => {
      cols.forEach((cx) => {
        const tile = document.createElement("img");
        tile.className = "preview-tile";
        tile.style.width = rect.width + "px";
        tile.style.height = rect.height + "px";
        tile.style.left = (cx + 1) * rect.width + "px";
        tile.style.top = (ry + 1) * rect.height + "px";
        tile.style.position = "absolute";

        const tileCenterX = view.x + cx * dxComplex;
        const tileCenterY = view.y + ry * dyComplex;

        const url = buildMandelURL({
          cx: tileCenterX,
          cy: tileCenterY,
          scale: view.scale,
          width: w,
          height: h,
          mode: previewMode,
          palette: previewNeedsPalette ? selectedPaletteIndex : undefined,
        });

        tile.src = url;

        previewInner.appendChild(tile);
      });
    });

    previewInner.style.width = rect.width * 3 + "px";
    previewInner.style.height = rect.height * 3 + "px";
    previewInner.style.transform = `translate(${-rect.width}px, ${-rect.height}px)`;
  }

  img.addEventListener('dragstart', (e) => e.preventDefault());

  img.addEventListener("mousedown", (e) => {
    e.preventDefault();
    dragging = true;
    dragStartX = e.clientX;
    dragStartY = e.clientY;
    createPreviewGrid();
  });

  window.addEventListener("mousemove", (e) => {
    if (!dragging || !previewInner || !currentRect) return;
    const dx = e.clientX - dragStartX;
    const dy = e.clientY - dragStartY;

    const baseX = -currentRect.width;
    const baseY = -currentRect.height;
    previewInner.style.transform = `translate(${baseX + dx}px, ${baseY + dy}px)`;
  });

  window.addEventListener("mouseup", (e) => {
    if (!dragging) return;
    dragging = false;

    if (currentRect) {
      const dx = e.clientX - dragStartX;
      const dy = e.clientY - dragStartY;
      const aspect = currentRect.height / currentRect.width;
      view.x -= dx * (view.scale / currentRect.width);
      view.y -= dy * (view.scale * aspect / currentRect.height);
    }

    updateOverlay();
    fetchAndShowImage({ hidePreviewWhenDone: true });
  });

  // ---------------- zoom ----------------
  img.addEventListener("wheel", (e) => {
    e.preventDefault();
    const before = clientToComplex(e.clientX, e.clientY);
    const zoomFactor = Math.exp(e.deltaY * 0.0012);
    const newScale = Math.max(1e-12, view.scale * zoomFactor);

    if (window.gsap) {
      const startScale = view.scale;
      const diff = newScale - startScale;
      const tmp = { t: 0 };
      gsap.to(tmp, {
        t: 1,
        duration: 0.35,
        ease: "power2.out",
        onUpdate: () => {
          view.scale = startScale + diff * tmp.t;
          const after = clientToComplex(e.clientX, e.clientY);
          view.x += before.x - after.x;
          view.y += before.y - after.y;
          updateOverlay();
        },
        onComplete: () => fetchAndShowImage(),
      });
    } else {
      view.scale = newScale;
      const after = clientToComplex(e.clientX, e.clientY);
      view.x += before.x - after.x;
      view.y += before.y - after.y;
      updateOverlay();
      fetchAndShowImage();
    }
  }, { passive: false });

  img.addEventListener("dblclick", (e) => {
    const pos = clientToComplex(e.clientX, e.clientY);
    if (window.gsap) {
      gsap.to(view, {
        x: pos.x,
        y: pos.y,
        scale: view.scale * 0.45,
        duration: 0.8,
        ease: "power2.out",
        onUpdate: updateOverlay,
        onComplete: fetchAndShowImage,
      });
    } else {
      view.x = pos.x;
      view.y = pos.y;
      view.scale *= 0.45;
      updateOverlay();
      fetchAndShowImage();
    }
  });

  // ---------------- buttons ----------------
  zoomInBtn?.addEventListener("click", () => {
    const newScale = view.scale * 0.6;
    if (window.gsap) {
      gsap.to(view, {
        scale: newScale,
        duration: 0.5,
        ease: "power2.out",
        onUpdate: updateOverlay,
        onComplete: fetchAndShowImage,
      });
    } else {
      view.scale = newScale;
      updateOverlay();
      fetchAndShowImage();
    }
  });

  zoomOutBtn?.addEventListener("click", () => {
    const newScale = view.scale * 1.6;
    if (window.gsap) {
      gsap.to(view, {
        scale: newScale,
        duration: 0.5,
        ease: "power2.out",
        onUpdate: updateOverlay,
        onComplete: fetchAndShowImage,
      });
    } else {
      view.scale = newScale;
      updateOverlay();
      fetchAndShowImage();
    }
  });

  resetBtn?.addEventListener("click", () => {
    if (window.gsap) {
      gsap.to(view, {
        x: -0.5,
        y: 0,
        scale: 3.5,
        duration: 0.8,
        ease: "power2.out",
        onUpdate: updateOverlay,
        onComplete: fetchAndShowImage,
      });
    } else {
      view.x = -0.5;
      view.y = 0;
      view.scale = 3.5;
      updateOverlay();
      fetchAndShowImage();
    }
  });

  randomBtn?.addEventListener("click", () => {
    loadRandomView();
  });

  // download menu
  saveToggle?.addEventListener("click", () => {
    if (!saveMenu) return;
    const open = saveMenu.classList.contains("open");
    if (open) {
      saveMenu.classList.remove("open");
    } else {
      saveMenu.classList.add("open");
    }
  });
  saveMenu?.addEventListener("click", (e) => {
    const btn = e.target.closest(".save-item");
    if (!btn) return;
    const w = Number(btn.dataset.w);
    const h = Number(btn.dataset.h);
    saveMenu.classList.remove("open");
    downloadAtResolution(w, h);
  });
  document.addEventListener("click", (e) => {
    if (!saveMenu) return;
    if (e.target === saveToggle || saveToggle.contains(e.target)) return;
    if (!saveMenu.contains(e.target)) {
      saveMenu.classList.remove("open");
    }
  });

  // add random palette
  addRandomPaletteBtn?.addEventListener("click", async () => {
    try {
      const res = await fetch(API_BASE + "/api/palettes/random");
      if (!res.ok) return;
      const p = await res.json();
      // append to local list and select it
      backendPalettes.push(p);
      selectedPaletteIndex = backendPalettes.length - 1;
      renderBackendPalettes();
      // only refetch if current mode uses palettes
      if (paletteModeUsesPalette(currentColorMode)) {
        fetchAndShowImage();
      }
    } catch (e) {
      console.warn("cannot fetch random palette", e);
    }
  });

  // color mode
  renderModeSelect?.addEventListener("change", (e) => {
    currentColorMode = Number(e.target.value) || 0;
    renderBackendPalettes();
    fetchAndShowImage();
  });

  // startup
  window.addEventListener("load", () => {
    updateOverlay();
    loadPalettesFromBackend();
    loadRandomView();
  });

  window.mandel = {
    view,
    reload: fetchAndShowImage,
  };
})();
