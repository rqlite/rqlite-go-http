// Node-oriented editable pro deck builder.
// Run this after editing SLIDES, SOURCES, and layout functions.
// The init script installs a sibling node_modules/@oai/artifact-tool package link
// and package.json with type=module for shell-run eval builders. Run with the
// Node executable from Codex workspace dependencies or the platform-appropriate
// command emitted by the init script.
// Do not use pnpm exec from the repo root or any Node binary whose module
// lookup cannot resolve the builder's sibling node_modules/@oai/artifact-tool.

const fs = await import("node:fs/promises");
const path = await import("node:path");
const { Presentation, PresentationFile } = await import("@oai/artifact-tool");

const W = 1280;
const H = 720;

const DECK_ID = "rhan-final-presentation";
const OUT_DIR = "C:\\opensource\\rqlite\\rqlite-go-http\\outputs\\rhan-final-presentation";
const REF_DIR = "C:\\opensource\\rqlite\\rqlite-go-http\\tmp\\rhan_pdf\\pages";
const SCRATCH_DIR = path.resolve(process.env.PPTX_SCRATCH_DIR || path.join("tmp", "slides", DECK_ID));
const PREVIEW_DIR = path.join(SCRATCH_DIR, "preview");
const VERIFICATION_DIR = path.join(SCRATCH_DIR, "verification");
const INSPECT_PATH = path.join(SCRATCH_DIR, "inspect.ndjson");
const MAX_RENDER_VERIFY_LOOPS = 3;

const INK = "#10243A";
const GRAPHITE = "#344054";
const MUTED = "#667085";
const PAPER = "#F4F1EA";
const PAPER_96 = "#F4F1EAF2";
const WHITE = "#FFFFFF";
const ACCENT = "#1D8A68";
const ACCENT_DARK = "#0F5C46";
const GOLD = "#D89B28";
const CORAL = "#C85F4F";
const TRANSPARENT = "#00000000";

const TITLE_FACE = "Aptos Display";
const BODY_FACE = "Aptos";
const MONO_FACE = "Aptos Mono";

const FALLBACK_PLATE_DATA_URL =
  "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=";

const SOURCES = {
  primary: "Rhan final-year project PDF, NIT Warangal, Academic Year 2025-26.",
};

const SLIDES = [
  {
    "kicker": "B.TECH FINAL PROJECT",
    "title": "Multimodal Satellite Image Registration",
    "subtitle": "A comparative study of classical feature-based methods and deep learning for EO versus SAR or NIR alignment.",
    "expectedVisual": "Strong academic cover using the source document visual with a premium title treatment.",
    "moment": "NN-DISK reached 98.19% CMR and 0.87 px RMSE.",
    "notes": "Open by stating the real problem: multimodal registration fails when the same object has very different radiometric appearance across sensors.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "01  MOTIVATION",
    "title": "Why This Problem Matters",
    "subtitle": "Accurate cross-modal registration is the prerequisite for reliable remote-sensing analysis.",
    "expectedVisual": "Three concise cards over the EO/SAR reference imagery from the PDF.",
    "cards": [
      [
        "Domain context",
        "Earth observation constellations generate terabytes of EO, SAR, infrared, and multispectral imagery every day for mapping, agriculture, and disaster response."
      ],
      [
        "Why registration",
        "These downstream tasks depend on precise spatial alignment. Mis-registered inputs create false changes, noisy overlays, and poor GIS decisions."
      ],
      [
        "Why multimodal is hard",
        "The same road can appear bright in EO and dark in SAR. Classical descriptors built for same-sensor imagery break under this inversion."
      ]
    ],
    "notes": "Use this slide to motivate the project from an application viewpoint before going deep into the algorithms.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "02  PROBLEM",
    "title": "Technical Challenge and Accuracy Target",
    "subtitle": "The study compares multiple registration paradigms under one common evaluation pipeline.",
    "expectedVisual": "Metric-style framing for the key accuracy and benchmark constraints.",
    "metrics": [
      [
        "2-3 px",
        "Misalignment can already break GIS overlays",
        "At 0.5 m per pixel, even small shifts are costly."
      ],
      [
        "<= 1 px",
        "Professional photogrammetry target",
        "Sub-pixel error is the practical acceptance line."
      ],
      [
        "6 methods",
        "Unified comparison setting",
        "Same preprocessing, matching logic, and RANSAC setup."
      ]
    ],
    "notes": "State the three core challenges: radiometric inversion, lack of standardized benchmarks, and sub-pixel accuracy needs.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "03  LITERATURE",
    "title": "What Earlier Work Gives Us",
    "subtitle": "The project spans fast classical baselines, multimodal descriptors, and learned local features.",
    "expectedVisual": "Three-panel literature snapshot over the original background page.",
    "cards": [
      [
        "Classical baselines",
        "SIFT, ORB, and BRISK provide scale or speed advantages, but their handcrafted comparisons are vulnerable when modality changes invert intensities."
      ],
      [
        "Multimodal classical",
        "SASF improves robustness by capturing structural similarity through NCC-based self-convolution instead of relying only on raw pixel intensity patterns."
      ],
      [
        "Learned descriptors",
        "SuperPoint and DISK show that matchability can be learned directly, making deep methods the strongest candidates for cross-modal alignment."
      ]
    ],
    "notes": "Call out the research gap: fair cross-paradigm comparison under identical conditions is missing in much of the literature.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "04  METHODOLOGY",
    "title": "Registration Pipeline and DISK Architecture",
    "subtitle": "The PDF figures are preserved here because they capture both the generic pipeline and the deep network design.",
    "expectedVisual": "Methodology slide backed by the original figure-heavy page from the report.",
    "cards": [
      [
        "Pipeline stages",
        "Preprocess and downscale, detect keypoints, extract descriptors, match features, reject outliers with Lowe plus RANSAC, then estimate homography and measure error."
      ],
      [
        "Deep model choice",
        "DISK uses a U-Net style encoder-decoder with detector and descriptor heads trained to maximize correct correspondences instead of a proxy objective."
      ],
      [
        "Fixed settings",
        "Maximum image size is 2000 px, ratio test is 0.75 for classical methods and 0.9 for DISK, and the RANSAC reprojection threshold is 5.0 px."
      ]
    ],
    "notes": "Walk the audience through Fig. 1 first, then point to Fig. 2 to explain why the deep method can model cross-modal structure better.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "05  IMPLEMENTATION",
    "title": "Technical Stack and Main Contributions",
    "subtitle": "The implementation combines OpenCV baselines with a custom SASF pipeline and PyTorch-based DISK inference.",
    "expectedVisual": "Metric layout for the stack, custom descriptor design, and evaluation controls.",
    "metrics": [
      [
        "Python 3.9",
        "Core implementation language",
        "OpenCV, NumPy, Matplotlib, PyTorch, and Kornia."
      ],
      [
        "4 x 16",
        "SASF ring-angle descriptor structure",
        "NCC over 4 rings and 16 angles drives the custom feature."
      ],
      [
        "Top 80%",
        "Entropy-based saliency filtering",
        "Keeps the most informative keypoints for SASF."
      ]
    ],
    "notes": "This is the place to mention the from-scratch SASF implementation and unified evaluation design across all compared methods.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "06  SETUP",
    "title": "Dataset and Evaluation Protocol",
    "subtitle": "The same-scene EO and NIR or SAR pair makes the visual mismatch obvious while keeping geometry comparable.",
    "expectedVisual": "Experimental setup slide using the original paired image page.",
    "cards": [
      [
        "Image pair",
        "One EO versus NIR or SAR pair from the same scene is used to test how each method handles strong appearance change with shared spatial content."
      ],
      [
        "Compared methods",
        "SIFT, ORB, BRISK, SASF, CNN, and NN-DISK are run inside one standardized pipeline so the comparison stays fair."
      ],
      [
        "Metrics",
        "Keypoint count tracks detections, CMR measures inlier quality after RANSAC, and RMSE captures the final geometric accuracy in pixels and meters."
      ]
    ],
    "notes": "Point directly to the EO and SAR or NIR visuals to show why intuitive same-image matching assumptions do not hold.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "07  RESULTS",
    "title": "Quantitative Results Across Algorithms",
    "subtitle": "This slide preserves the original charts because they are the strongest visual evidence in the presentation.",
    "expectedVisual": "High-impact results page over the original chart panel from the report.",
    "metrics": [
      [
        "98.19%",
        "Correct matching rate for NN-DISK",
        "Highest by a very large margin."
      ],
      [
        "0.87 px",
        "RMSE for NN-DISK",
        "Only method below the 1 px threshold."
      ],
      [
        "0.44 m",
        "Ground error at 0.5 m per pixel",
        "Best practical outcome for mapping tasks."
      ]
    ],
    "notes": "Use the table and bar charts together: the bars show separation quickly, and the table gives the exact numbers the examiners may ask about.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "08  ANALYSIS",
    "title": "Why DISK Performs So Much Better",
    "subtitle": "The chart and bullets together explain the gap between handcrafted descriptors and learned matchability.",
    "expectedVisual": "Analysis slide over the original discussion chart page.",
    "cards": [
      [
        "Why ORB and BRISK fail",
        "Binary intensity comparisons flip sign under cross-modal inversion, so they produce many unstable matches even when the physical scene is the same."
      ],
      [
        "Why SASF improves",
        "SASF uses structural similarity rather than direct radiometric agreement, so it becomes the strongest classical option in the experiment."
      ],
      [
        "Why DISK wins",
        "DISK learns detector and descriptor behavior jointly, uses broader context through the network, and works with cycle-consistent matching before RANSAC."
      ]
    ],
    "notes": "Mention the standout comparison: DISK achieves 1,196 correct matches versus only 37 for ORB, which is roughly 32.95 times higher.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "09  LIMITATIONS",
    "title": "Current Constraints of the Study",
    "subtitle": "The result is strong, but the evaluation still has scope and deployment limitations that should be stated openly.",
    "expectedVisual": "Three limitations cards anchored on the original limitations page.",
    "cards": [
      [
        "Limited data breadth",
        "Only one EO versus NIR or SAR pair was evaluated, so generalization to new regions, seasons, and sensor pairs still needs validation."
      ],
      [
        "Runtime cost",
        "NN-DISK takes about 60 seconds on CPU, while OpenCV baselines finish in under 0.2 seconds and SASF takes roughly 6 seconds."
      ],
      [
        "Model assumption",
        "Homography assumes a mostly planar scene, and DISK was not fine-tuned specifically on satellite EO-SAR data."
      ]
    ],
    "notes": "Being explicit about limitations usually helps in viva discussions because it shows control over the scope of the work.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "10  CONCLUSION",
    "title": "Core Takeaways",
    "subtitle": "The deep-learning approach produces a categorical improvement rather than a small incremental gain.",
    "expectedVisual": "Conclusion slide with clean metric emphasis and strong verbal summary.",
    "metrics": [
      [
        "32.95x",
        "More correct matches than ORB",
        "1,196 versus 37 after geometric verification."
      ],
      [
        "7x",
        "Lower RMSE than the best classical method",
        "NN-DISK versus SASF in the final comparison."
      ],
      [
        "21.9%",
        "Best classical CMR from SASF",
        "Useful fallback when deep infrastructure is unavailable."
      ]
    ],
    "notes": "End with a clear statement: learned features outperform handcrafted invariances for multimodal satellite registration.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "11  FUTURE SCOPE",
    "title": "Where This Can Go Next",
    "subtitle": "The roadmap now is about better matchers, larger datasets, and deployment-ready inference.",
    "expectedVisual": "Forward-looking roadmap slide using the future-work page as a visual anchor.",
    "cards": [
      [
        "Better matching",
        "Replace soft nearest-neighbour matching with LightGlue-style transformer matching for stronger context-aware correspondences."
      ],
      [
        "Better data",
        "Fine-tune on curated Sentinel-1 and Sentinel-2 multimodal pairs to adapt the learned features to satellite-specific texture and noise."
      ],
      [
        "Better deployment",
        "Use tiling, quantization, and TensorRT or Jetson deployment to push the method toward real-time or near-real-time use."
      ]
    ],
    "notes": "If asked for future scope, mention the target direction clearly: more than 99% CMR, below 0.5 px RMSE, and faster inference.",
    "sources": [
      "primary"
    ]
  },
  {
    "kicker": "12  REFERENCES",
    "title": "Key References",
    "subtitle": "The project is grounded in foundational local features, multimodal remote-sensing registration, and robust geometric fitting.",
    "expectedVisual": "Readable academic closing slide with grouped references.",
    "cards": [
      [
        "Foundations",
        "Lowe 2004 for SIFT, Rublee et al. 2011 for ORB, and Leutenegger et al. 2011 for BRISK establish the classical feature baseline."
      ],
      [
        "Multimodal and learned features",
        "Ye et al. 2017 motivates SASF for multimodal sensing, while DeTone et al. 2018 and Tyszkiewicz et al. 2020 frame the learned feature direction."
      ],
      [
        "Robust estimation",
        "Fischler and Bolles 1981 remains the geometric backbone through RANSAC, which turns tentative matches into valid homography inliers."
      ]
    ],
    "notes": "Keep this slide visible if your department expects formal references in the main deck.",
    "sources": [
      "primary"
    ]
  }
];

const inspectRecords = [];

async function pathExists(filePath) {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
}

async function readImageBlob(imagePath) {
  const bytes = await fs.readFile(imagePath);
  if (!bytes.byteLength) {
    throw new Error(`Image file is empty: ${imagePath}`);
  }
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
}

async function normalizeImageConfig(config) {
  if (!config.path) {
    return config;
  }
  const { path: imagePath, ...rest } = config;
  return {
    ...rest,
    blob: await readImageBlob(imagePath),
  };
}

async function ensureDirs() {
  await fs.mkdir(OUT_DIR, { recursive: true });
  const obsoleteFinalArtifacts = [
    "preview",
    "verification",
    "inspect.ndjson",
    ["presentation", "proto.json"].join("_"),
    ["quality", "report.json"].join("_"),
  ];
  for (const obsolete of obsoleteFinalArtifacts) {
    await fs.rm(path.join(OUT_DIR, obsolete), { recursive: true, force: true });
  }
  await fs.mkdir(SCRATCH_DIR, { recursive: true });
  await fs.mkdir(PREVIEW_DIR, { recursive: true });
  await fs.mkdir(VERIFICATION_DIR, { recursive: true });
}

function lineConfig(fill = TRANSPARENT, width = 0) {
  return { style: "solid", fill, width };
}

function recordShape(slideNo, shape, role, shapeType, x, y, w, h) {
  if (!slideNo) return;
  inspectRecords.push({
    kind: "shape",
    slide: slideNo,
    id: shape?.id || `slide-${slideNo}-${role}-${inspectRecords.length + 1}`,
    role,
    shapeType,
    bbox: [x, y, w, h],
  });
}

function addShape(slide, geometry, x, y, w, h, fill = TRANSPARENT, line = TRANSPARENT, lineWidth = 0, meta = {}) {
  const shape = slide.shapes.add({
    geometry,
    position: { left: x, top: y, width: w, height: h },
    fill,
    line: lineConfig(line, lineWidth),
  });
  recordShape(meta.slideNo, shape, meta.role || geometry, geometry, x, y, w, h);
  return shape;
}

function normalizeText(text) {
  if (Array.isArray(text)) {
    return text.map((item) => String(item ?? "")).join("\n");
  }
  return String(text ?? "");
}

function textLineCount(text) {
  const value = normalizeText(text);
  if (!value.trim()) {
    return 0;
  }
  return Math.max(1, value.split(/\n/).length);
}

function requiredTextHeight(text, fontSize, lineHeight = 1.18, minHeight = 8) {
  const lines = textLineCount(text);
  if (lines === 0) {
    return minHeight;
  }
  return Math.max(minHeight, lines * fontSize * lineHeight);
}

function assertTextFits(text, boxHeight, fontSize, role = "text") {
  const required = requiredTextHeight(text, fontSize);
  const tolerance = Math.max(2, fontSize * 0.08);
  if (normalizeText(text).trim() && boxHeight + tolerance < required) {
    throw new Error(
      `${role} text box is too short: height=${boxHeight.toFixed(1)}, required>=${required.toFixed(1)}, ` +
        `lines=${textLineCount(text)}, fontSize=${fontSize}, text=${JSON.stringify(normalizeText(text).slice(0, 90))}`,
    );
  }
}

function wrapText(text, widthChars) {
  const words = normalizeText(text).split(/\s+/).filter(Boolean);
  const lines = [];
  let current = "";
  for (const word of words) {
    const next = current ? `${current} ${word}` : word;
    if (next.length > widthChars && current) {
      lines.push(current);
      current = word;
    } else {
      current = next;
    }
  }
  if (current) {
    lines.push(current);
  }
  return lines.join("\n");
}

function recordText(slideNo, shape, role, text, x, y, w, h) {
  const value = normalizeText(text);
  inspectRecords.push({
    kind: "textbox",
    slide: slideNo,
    id: shape?.id || `slide-${slideNo}-${role}-${inspectRecords.length + 1}`,
    role,
    text: value,
    textPreview: value.replace(/\n/g, " | ").slice(0, 180),
    textChars: value.length,
    textLines: textLineCount(value),
    bbox: [x, y, w, h],
  });
}

function recordImage(slideNo, image, role, imagePath, x, y, w, h) {
  inspectRecords.push({
    kind: "image",
    slide: slideNo,
    id: image?.id || `slide-${slideNo}-${role}-${inspectRecords.length + 1}`,
    role,
    path: imagePath,
    bbox: [x, y, w, h],
  });
}

function applyTextStyle(box, text, size, color, bold, face, align, valign, autoFit, listStyle) {
  box.text = text;
  box.text.fontSize = size;
  box.text.color = color;
  box.text.bold = Boolean(bold);
  box.text.alignment = align;
  box.text.verticalAlignment = valign;
  box.text.typeface = face;
  box.text.insets = { left: 0, right: 0, top: 0, bottom: 0 };
  if (autoFit) {
    box.text.autoFit = autoFit;
  }
  if (listStyle) {
    box.text.style = "list";
  }
}

function addText(
  slide,
  slideNo,
  text,
  x,
  y,
  w,
  h,
  {
    size = 22,
    color = INK,
    bold = false,
    face = BODY_FACE,
    align = "left",
    valign = "top",
    fill = TRANSPARENT,
    line = TRANSPARENT,
    lineWidth = 0,
    autoFit = null,
    listStyle = false,
    checkFit = true,
    role = "text",
  } = {},
) {
  if (!checkFit && textLineCount(text) > 1) {
    throw new Error("checkFit=false is only allowed for single-line headers, footers, and captions.");
  }
  if (checkFit) {
    assertTextFits(text, h, size, role);
  }
  const box = addShape(slide, "rect", x, y, w, h, fill, line, lineWidth);
  applyTextStyle(box, text, size, color, bold, face, align, valign, autoFit, listStyle);
  recordText(slideNo, box, role, text, x, y, w, h);
  return box;
}

async function addImage(slide, slideNo, config, position, role, sourcePath = null) {
  const image = slide.images.add(await normalizeImageConfig(config));
  image.position = position;
  recordImage(slideNo, image, role, sourcePath || config.path || config.uri || "inline-data-url", position.left, position.top, position.width, position.height);
  return image;
}

async function addPlate(slide, slideNo, opacityPanel = false) {
  slide.background.fill = PAPER;
  if (opacityPanel) {
    addShape(slide, "rect", 0, 0, W, H, "#FFFFFFB8", TRANSPARENT, 0, { slideNo, role: "plate readability overlay" });
  }
}

async function addSourceFigurePanel(slide, slideNo, sourcePageNo, x, y, w, h, { title = null } = {}) {
  const platePath = path.join(REF_DIR, `slide-${String(sourcePageNo).padStart(2, "0")}.png`);
  addShape(slide, "roundRect", x, y, w, h, WHITE, "#CBD5E1", 1.2, { slideNo, role: `figure panel ${sourcePageNo}` });
  if (await pathExists(platePath)) {
    await addImage(
      slide,
      slideNo,
      { path: platePath, fit: "contain", alt: `Source figure page ${sourcePageNo}` },
      { left: x + 10, top: y + 10, width: w - 20, height: h - 20 },
      `source figure ${sourcePageNo}`,
      platePath,
    );
  }
  if (title) {
    addText(slide, slideNo, title, x + 16, y + h + 6, w - 32, 18, {
      size: 11,
      color: MUTED,
      face: BODY_FACE,
      checkFit: false,
      role: "figure caption",
    });
  }
}

function addHeader(slide, slideNo, kicker, idx, total) {
  addText(slide, slideNo, String(kicker || "").toUpperCase(), 64, 34, 430, 24, {
    size: 13,
    color: ACCENT_DARK,
    bold: true,
    face: MONO_FACE,
    checkFit: false,
    role: "header",
  });
  addText(slide, slideNo, `${String(idx).padStart(2, "0")} / ${String(total).padStart(2, "0")}`, 1114, 34, 104, 24, {
    size: 13,
    color: ACCENT_DARK,
    bold: true,
    face: MONO_FACE,
    align: "right",
    checkFit: false,
    role: "header",
  });
  addShape(slide, "rect", 64, 64, 1152, 2, INK, TRANSPARENT, 0, { slideNo, role: "header rule" });
  addShape(slide, "ellipse", 57, 57, 16, 16, ACCENT, INK, 2, { slideNo, role: "header marker" });
}

function addTitleBlock(slide, slideNo, title, subtitle = null, x = 64, y = 86, w = 780, dark = false) {
  const titleColor = dark ? PAPER : INK;
  const bodyColor = dark ? PAPER : GRAPHITE;
  addText(slide, slideNo, title, x, y, w, 142, {
    size: 40,
    color: titleColor,
    bold: true,
    face: TITLE_FACE,
    role: "title",
  });
  if (subtitle) {
    addText(slide, slideNo, subtitle, x + 2, y + 148, Math.min(w, 720), 70, {
      size: 19,
      color: bodyColor,
      face: BODY_FACE,
      role: "subtitle",
    });
  }
}

function addIconBadge(slide, slideNo, x, y, accent = ACCENT, kind = "signal") {
  addShape(slide, "ellipse", x, y, 54, 54, PAPER_96, INK, 1.2, { slideNo, role: "icon badge" });
  if (kind === "flow") {
    addShape(slide, "ellipse", x + 13, y + 18, 10, 10, accent, INK, 1, { slideNo, role: "icon glyph" });
    addShape(slide, "ellipse", x + 31, y + 27, 10, 10, accent, INK, 1, { slideNo, role: "icon glyph" });
    addShape(slide, "rect", x + 22, y + 25, 19, 3, INK, TRANSPARENT, 0, { slideNo, role: "icon glyph" });
  } else if (kind === "layers") {
    addShape(slide, "roundRect", x + 13, y + 15, 26, 13, accent, INK, 1, { slideNo, role: "icon glyph" });
    addShape(slide, "roundRect", x + 18, y + 24, 26, 13, GOLD, INK, 1, { slideNo, role: "icon glyph" });
    addShape(slide, "roundRect", x + 23, y + 33, 20, 10, CORAL, INK, 1, { slideNo, role: "icon glyph" });
  } else {
    addShape(slide, "rect", x + 16, y + 29, 6, 12, accent, TRANSPARENT, 0, { slideNo, role: "icon glyph" });
    addShape(slide, "rect", x + 25, y + 21, 6, 20, accent, TRANSPARENT, 0, { slideNo, role: "icon glyph" });
    addShape(slide, "rect", x + 34, y + 14, 6, 27, accent, TRANSPARENT, 0, { slideNo, role: "icon glyph" });
  }
}

function addCard(slide, slideNo, x, y, w, h, label, body, { accent = ACCENT, fill = PAPER_96, line = INK, iconKind = "signal" } = {}) {
  if (h < 156) {
    throw new Error(`Card is too short for editable pro-deck copy: height=${h.toFixed(1)}, minimum=156.`);
  }
  addShape(slide, "roundRect", x, y, w, h, fill, line, 1.2, { slideNo, role: `card panel: ${label}` });
  addShape(slide, "rect", x, y, 8, h, accent, TRANSPARENT, 0, { slideNo, role: `card accent: ${label}` });
  addIconBadge(slide, slideNo, x + 22, y + 24, accent, iconKind);
  addText(slide, slideNo, label, x + 88, y + 22, w - 108, 28, {
    size: 15,
    color: ACCENT_DARK,
    bold: true,
    face: MONO_FACE,
    role: "card label",
  });
  const wrapped = wrapText(body, Math.max(28, Math.floor(w / 13)));
  const bodyY = y + 74;
  const bodyH = h - (bodyY - y) - 20;
  if (bodyH < 54) {
    throw new Error(`Card body area is too short: height=${bodyH.toFixed(1)}, cardHeight=${h.toFixed(1)}, label=${JSON.stringify(label)}.`);
  }
  addText(slide, slideNo, wrapped, x + 24, bodyY, w - 48, bodyH, {
    size: 15,
    color: INK,
    face: BODY_FACE,
    role: `card body: ${label}`,
  });
}

function addMetricCard(slide, slideNo, x, y, w, h, metric, label, note = null, accent = ACCENT) {
  if (h < 132) {
    throw new Error(`Metric card is too short for editable pro-deck copy: height=${h.toFixed(1)}, minimum=132.`);
  }
  addShape(slide, "roundRect", x, y, w, h, PAPER_96, INK, 1.2, { slideNo, role: `metric panel: ${label}` });
  addShape(slide, "rect", x, y, w, 7, accent, TRANSPARENT, 0, { slideNo, role: `metric accent: ${label}` });
  addText(slide, slideNo, metric, x + 22, y + 24, w - 44, 54, {
    size: 34,
    color: INK,
    bold: true,
    face: TITLE_FACE,
    role: "metric value",
  });
  addText(slide, slideNo, label, x + 24, y + 82, w - 48, 38, {
    size: 16,
    color: GRAPHITE,
    face: BODY_FACE,
    role: "metric label",
  });
  if (note) {
    addText(slide, slideNo, note, x + 24, y + h - 42, w - 48, 22, {
      size: 10,
      color: MUTED,
      face: BODY_FACE,
      role: "metric note",
    });
  }
}

function addNotes(slide, body, sourceKeys) {
  const sourceLines = (sourceKeys || []).map((key) => `- ${SOURCES[key] || key}`).join("\n");
  slide.speakerNotes.setText(`${body || ""}\n\n[Sources]\n${sourceLines}`);
}

function addReferenceCaption(slide, slideNo) {
  addText(
    slide,
    slideNo,
    "Slide rebuilt with a clean layout for presentation clarity.",
    64,
    674,
    1080,
    22,
    {
      size: 10,
      color: MUTED,
      face: BODY_FACE,
      checkFit: false,
      role: "caption",
    },
  );
}

async function slideCover(presentation) {
  const slideNo = 1;
  const data = SLIDES[0];
  const slide = presentation.slides.add();
  await addPlate(slide, slideNo);
  await addSourceFigurePanel(slide, slideNo, 1, 792, 120, 430, 430, { title: "Project cover from source report" });
  addShape(slide, "rect", 64, 86, 7, 455, ACCENT, TRANSPARENT, 0, { slideNo, role: "cover accent rule" });
  addText(slide, slideNo, data.kicker, 86, 88, 520, 26, {
    size: 13,
    color: ACCENT_DARK,
    bold: true,
    face: MONO_FACE,
    role: "kicker",
  });
  addText(slide, slideNo, data.title, 82, 130, 785, 184, {
    size: 48,
    color: INK,
    bold: true,
    face: TITLE_FACE,
    role: "cover title",
  });
  addText(slide, slideNo, data.subtitle, 86, 326, 610, 86, {
    size: 20,
    color: GRAPHITE,
    face: BODY_FACE,
    role: "cover subtitle",
  });
  addShape(slide, "roundRect", 86, 456, 390, 92, PAPER_96, INK, 1.2, { slideNo, role: "cover moment panel" });
  addText(slide, slideNo, data.moment || "Replace with core idea", 112, 478, 336, 40, {
    size: 23,
    color: INK,
    bold: true,
    face: TITLE_FACE,
    role: "cover moment",
  });
  addReferenceCaption(slide, slideNo);
  addNotes(slide, data.notes, data.sources);
}

async function slideCards(presentation, idx) {
  const data = SLIDES[idx - 1];
  const slide = presentation.slides.add();
  await addPlate(slide, idx);
  addHeader(slide, idx, data.kicker, idx, SLIDES.length);
  addTitleBlock(slide, idx, data.title, data.subtitle, 64, 86, 760);
  if (idx === 5) {
    await addSourceFigurePanel(slide, idx, 5, 64, 150, 1152, 248, { title: "Source report visuals: registration pipeline and DISK architecture" });
  }
  if (idx === 7) {
    await addSourceFigurePanel(slide, idx, 7, 780, 136, 436, 240, { title: "Source report visual: EO and NIR or SAR example pair" });
  }
  if (idx === 9) {
    await addSourceFigurePanel(slide, idx, 9, 64, 150, 500, 300, { title: "Source report visual: keypoints detected and correct matches" });
  }
  const cards = data.cards?.length
    ? data.cards
    : [
        ["Replace", "Add a specific, sourced point for this slide."],
        ["Author", "Use native PowerPoint chart objects for charts; use deterministic geometry for cards and callouts."],
        ["Verify", "Render previews, inspect them at readable size, and fix actionable layout issues within 3 total render loops."],
      ];
  const cols = Math.min(3, cards.length);
  const cardW = (1114 - (cols - 1) * 24) / cols;
  const iconKinds = ["signal", "flow", "layers"];
  for (let cardIdx = 0; cardIdx < cols; cardIdx += 1) {
    const [label, body] = cards[cardIdx];
    const x = 84 + cardIdx * (cardW + 24);
    const y = idx === 5 ? 430 : idx === 7 ? 400 : idx === 9 ? 430 : 350;
    addCard(slide, idx, x, y, cardW, 240, label, body, { iconKind: iconKinds[cardIdx % iconKinds.length] });
  }
  addReferenceCaption(slide, idx);
  addNotes(slide, data.notes, data.sources);
}

async function slideMetrics(presentation, idx) {
  const data = SLIDES[idx - 1];
  const slide = presentation.slides.add();
  await addPlate(slide, idx);
  addHeader(slide, idx, data.kicker, idx, SLIDES.length);
  addTitleBlock(slide, idx, data.title, data.subtitle, 64, 86, 700);
  if (idx === 8) {
    await addSourceFigurePanel(slide, idx, 8, 64, 150, 1152, 278, { title: "Source report visuals: CMR and RMSE comparison charts with result table" });
  }
  const metrics = data.metrics || [
    ["00", "Replace metric", "Source"],
    ["00", "Replace metric", "Source"],
    ["00", "Replace metric", "Source"],
  ];
  const accents = [ACCENT, GOLD, CORAL];
  for (let metricIdx = 0; metricIdx < Math.min(3, metrics.length); metricIdx += 1) {
    const [metric, label, note] = metrics[metricIdx];
    const y = idx === 8 ? 470 : 404;
    addMetricCard(slide, idx, 92 + metricIdx * 370, y, 330, 174, metric, label, note, accents[metricIdx % accents.length]);
  }
  addReferenceCaption(slide, idx);
  addNotes(slide, data.notes, data.sources);
}

async function createDeck() {
  await ensureDirs();
  if (!SLIDES.length) {
    throw new Error("SLIDES must contain at least one slide.");
  }
  const presentation = Presentation.create({ slideSize: { width: W, height: H } });
  presentation.theme.colorScheme = {
    name: "Rhan Viva Theme",
    themeColors: {
      accent1: ACCENT,
      accent2: GOLD,
      accent3: CORAL,
      bg1: PAPER,
      bg2: WHITE,
      tx1: INK,
      tx2: GRAPHITE,
    },
  };
  await slideCover(presentation);
  for (let idx = 2; idx <= SLIDES.length; idx += 1) {
    const data = SLIDES[idx - 1];
    if (data.metrics) {
      await slideMetrics(presentation, idx);
    } else {
      await slideCards(presentation, idx);
    }
  }
  return presentation;
}

async function saveBlobToFile(blob, filePath) {
  const bytes = new Uint8Array(await blob.arrayBuffer());
  await fs.writeFile(filePath, bytes);
}

async function writeInspectArtifact(presentation) {
  inspectRecords.unshift({
    kind: "deck",
    id: DECK_ID,
    slideCount: presentation.slides.count,
    slideSize: { width: W, height: H },
  });
  presentation.slides.items.forEach((slide, index) => {
    inspectRecords.splice(index + 1, 0, {
      kind: "slide",
      slide: index + 1,
      id: slide?.id || `slide-${index + 1}`,
    });
  });
  const lines = inspectRecords.map((record) => JSON.stringify(record)).join("\n") + "\n";
  await fs.writeFile(INSPECT_PATH, lines, "utf8");
}

async function currentRenderLoopCount() {
  const logPath = path.join(VERIFICATION_DIR, "render_verify_loops.ndjson");
  if (!(await pathExists(logPath))) return 0;
  const previous = await fs.readFile(logPath, "utf8");
  return previous.split(/\r?\n/).filter((line) => line.trim()).length;
}

async function nextRenderLoopNumber() {
  return (await currentRenderLoopCount()) + 1;
}

async function appendRenderVerifyLoop(presentation, previewPaths, pptxPath) {
  const logPath = path.join(VERIFICATION_DIR, "render_verify_loops.ndjson");
  const priorCount = await currentRenderLoopCount();
  const record = {
    kind: "render_verify_loop",
    deckId: DECK_ID,
    loop: priorCount + 1,
    maxLoops: MAX_RENDER_VERIFY_LOOPS,
    capReached: priorCount + 1 >= MAX_RENDER_VERIFY_LOOPS,
    timestamp: new Date().toISOString(),
    slideCount: presentation.slides.count,
    previewCount: previewPaths.length,
    previewDir: PREVIEW_DIR,
    inspectPath: INSPECT_PATH,
    pptxPath,
  };
  await fs.appendFile(logPath, JSON.stringify(record) + "\n", "utf8");
  return record;
}

async function verifyAndExport(presentation) {
  await ensureDirs();
  const nextLoop = await nextRenderLoopNumber();
  if (nextLoop > MAX_RENDER_VERIFY_LOOPS) {
    throw new Error(
      `Render/verify/fix loop cap reached: ${MAX_RENDER_VERIFY_LOOPS} total renders are allowed. ` +
        "Do not rerender; note any remaining visual issues in the final response.",
    );
  }
  await writeInspectArtifact(presentation);
  const previewPaths = [];
  for (let idx = 0; idx < presentation.slides.items.length; idx += 1) {
    const slide = presentation.slides.items[idx];
    const preview = await presentation.export({ slide, format: "png", scale: 1 });
    const previewPath = path.join(PREVIEW_DIR, `slide-${String(idx + 1).padStart(2, "0")}.png`);
    await saveBlobToFile(preview, previewPath);
    previewPaths.push(previewPath);
  }
  const pptxBlob = await PresentationFile.exportPptx(presentation);
  const pptxPath = path.join(OUT_DIR, "output.pptx");
  await pptxBlob.save(pptxPath);
  const loopRecord = await appendRenderVerifyLoop(presentation, previewPaths, pptxPath);
  return { pptxPath, loopRecord };
}

const presentation = await createDeck();
const result = await verifyAndExport(presentation);
console.log(result.pptxPath);
