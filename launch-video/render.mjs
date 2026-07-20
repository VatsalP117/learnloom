import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const ROOT = path.dirname(fileURLToPath(import.meta.url));
const FRAMES = path.join(ROOT, "frames");
const PNGS = path.join(ROOT, "png");
const OUT = path.join(ROOT, "output");
const W = 1920;
const H = 1080;
const FPS = 30;
const DURATION = 46;
const TOTAL = FPS * DURATION;

fs.rmSync(FRAMES, { recursive: true, force: true });
fs.rmSync(PNGS, { recursive: true, force: true });
fs.mkdirSync(FRAMES, { recursive: true });
fs.mkdirSync(PNGS, { recursive: true });
fs.mkdirSync(OUT, { recursive: true });

const clamp = (v, a = 0, b = 1) => Math.max(a, Math.min(b, v));
const mix = (a, b, t) => a + (b - a) * t;
const ease = (t) => {
  t = clamp(t);
  return t < .5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
};
const inOut = (t, start, end, edge = .55) =>
  ease(clamp((t - start) / edge)) * ease(clamp((end - t) / edge));
const esc = (s) => String(s).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");

const colors = {
  ink: "#17211b",
  muted: "#68736b",
  green: "#4d7253",
  lime: "#abc477",
  paper: "#f7f6f0",
  dark: "#101811",
  line: "#dfe4dd",
  white: "#ffffff",
};

function defs() {
  return `<defs>
    <filter id="shadow" x="-40%" y="-40%" width="180%" height="180%">
      <feDropShadow dx="0" dy="22" stdDeviation="28" flood-color="#132219" flood-opacity=".14"/>
    </filter>
    <filter id="soft" x="-40%" y="-40%" width="180%" height="180%">
      <feGaussianBlur stdDeviation="35"/>
    </filter>
    <linearGradient id="space" x1="0" y1="0" x2="1" y2="1">
      <stop stop-color="#07100c"/><stop offset=".55" stop-color="#142119"/><stop offset="1" stop-color="#050906"/>
    </linearGradient>
    <radialGradient id="glow">
      <stop stop-color="#b5d69a" stop-opacity=".28"/><stop offset="1" stop-color="#6a9d70" stop-opacity="0"/>
    </radialGradient>
    <linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
      <stop stop-color="#eef8fb"/><stop offset=".63" stop-color="#dcecd9"/><stop offset="1" stop-color="#8aa778"/>
    </linearGradient>
    <clipPath id="browserClip"><rect x="0" y="0" width="1240" height="720" rx="24"/></clipPath>
  </defs>`;
}

function brandMark(x, y, size = 48, dark = true) {
  const bg = dark ? colors.ink : "#eef5eb";
  const fg = dark ? "#f3f8f1" : colors.green;
  return `<g transform="translate(${x} ${y})">
    <rect width="${size}" height="${size}" rx="${size * .28}" fill="${bg}"/>
    <path d="M${size*.27} ${size*.27} C${size*.49} ${size*.17},${size*.73} ${size*.29},${size*.69} ${size*.51}
      C${size*.65} ${size*.72},${size*.39} ${size*.79},${size*.26} ${size*.61}
      C${size*.14} ${size*.44},${size*.27} ${size*.27},${size*.27} ${size*.27}Z"
      fill="none" stroke="${fg}" stroke-width="${size*.085}" stroke-linecap="round"/>
    <path d="M${size*.37} ${size*.58} C${size*.49} ${size*.38},${size*.64} ${size*.36},${size*.78} ${size*.43}"
      fill="none" stroke="${fg}" stroke-width="${size*.07}" stroke-linecap="round"/>
  </g>`;
}

function text(x, y, value, size, opts = {}) {
  const {
    fill = colors.ink, weight = 500, anchor = "start", family = "Helvetica Neue, Helvetica, Arial, sans-serif",
    opacity = 1, tracking = 0, style = "", transform = "",
  } = opts;
  return `<text x="${x}" y="${y}" text-anchor="${anchor}" fill="${fill}" opacity="${opacity}"
    font-family="${family}" font-size="${size}" font-weight="${weight}" letter-spacing="${tracking}"
    font-style="${style}" transform="${transform}">${esc(value)}</text>`;
}

function lineText(cx, cy, lines, size, gap, opts = {}) {
  return lines.map((s, i) => text(cx, cy + i * gap, s, size, { anchor: "middle", ...opts })).join("");
}

function pill(x, y, w, label, fill = "#eef3eb", color = colors.green) {
  return `<g transform="translate(${x} ${y})"><rect width="${w}" height="42" rx="21" fill="${fill}"/>
    <circle cx="22" cy="21" r="5" fill="${colors.lime}"/>${text(38, 27, label, 14, { fill: color, weight: 700 })}</g>`;
}

function browser(x, y, w, h, content, opts = {}) {
  const { scale = 1, opacity = 1, title = "maya.learnloom.blog" } = opts;
  return `<g transform="translate(${x} ${y}) scale(${scale})" opacity="${opacity}" filter="url(#shadow)">
    <rect width="${w}" height="${h}" rx="24" fill="#fff" stroke="#dce2dc" stroke-width="2"/>
    <rect width="${w}" height="62" rx="24" fill="#f6f7f3"/>
    <rect y="38" width="${w}" height="24" fill="#f6f7f3"/>
    <circle cx="26" cy="31" r="6" fill="#c8cec8"/><circle cx="47" cy="31" r="6" fill="#c8cec8"/><circle cx="68" cy="31" r="6" fill="#c8cec8"/>
    <rect x="${w/2-180}" y="15" width="360" height="34" rx="10" fill="#fff" stroke="#e4e8e3"/>
    <circle cx="${w/2-152}" cy="32" r="7" fill="${colors.green}"/>
    ${text(w/2-135, 37, title, 13, { fill: "#748078", weight: 550 })}
    <g transform="translate(0 62)">${content}</g>
  </g>`;
}

function starfield(t) {
  let s = `<rect width="${W}" height="${H}" fill="url(#space)"/>
    <circle cx="${mix(1420, 1270, t/46)}" cy="260" r="460" fill="url(#glow)"/>`;
  for (let i = 0; i < 96; i++) {
    const x = (i * 193 + 37) % W;
    const y = (i * i * 31 + 83) % H;
    const r = .7 + (i % 4) * .45;
    const op = .18 + ((i * 7) % 10) / 18;
    s += `<circle cx="${x}" cy="${y}" r="${r}" fill="#e9f8e8" opacity="${op}"/>`;
  }
  return s;
}

function sceneLogo(t) {
  const a = inOut(t, 0, 3.7, .65);
  const z = mix(.9, 1, ease(clamp(t / 2.5)));
  return `<rect width="${W}" height="${H}" fill="#fff"/>
    <g opacity="${a}" transform="translate(960 515) scale(${z}) translate(-960 -515)">
      ${brandMark(864, 446, 72)}
      ${text(952, 498, "Learnloom", 54, { weight: 720, tracking: -2.5 })}
      ${text(960, 567, "A learning home that grows with you.", 20, { anchor: "middle", fill: colors.muted, weight: 450 })}
    </g>`;
}

function sceneManifesto(t) {
  const local = t - 3.1;
  const a1 = inOut(local, 0, 3.8, .55);
  const a2 = inOut(local, 3.25, 7.1, .55);
  const y1 = mix(572, 535, ease(clamp(local / .8)));
  const y2 = mix(572, 535, ease(clamp((local - 3.25) / .8)));
  return `<rect width="${W}" height="${H}" fill="#fff"/>
    <g opacity="${a1}">${lineText(960, y1, ["The web gives you endless", "things to read."], 76, 82, { weight: 680, tracking: -3.8 })}</g>
    <g opacity="${a2}">${lineText(960, y2, ["Learnloom gives each idea", "somewhere to live."], 76, 82, { weight: 680, tracking: -3.8 })}
      ${text(960, 732, "CURRENT SOURCES  →  DURABLE UNDERSTANDING", 15, { anchor: "middle", fill: colors.green, weight: 750, tracking: 2.4 })}
    </g>`;
}

function sourceCard(x, y, w, title, source, active = false) {
  return `<g transform="translate(${x} ${y})">
    <rect width="${w}" height="98" rx="16" fill="${active ? "#eef4eb" : "#fff"}" stroke="${active ? "#9bb391" : "#e1e5df"}" stroke-width="${active ? 2 : 1}"/>
    <rect x="18" y="18" width="62" height="62" rx="12" fill="${active ? "#dbe8d5" : "#f0f2ee"}"/>
    <path d="M34 61 L46 43 L55 52 L66 35" fill="none" stroke="${active ? colors.green : "#8b958d"}" stroke-width="4" stroke-linecap="round"/>
    ${text(98, 42, title, 17, { weight: 650 })}${text(98, 68, source, 12, { fill: colors.muted })}
    <circle cx="${w-28}" cy="49" r="11" fill="${active ? colors.green : "#fff"}" stroke="${active ? colors.green : "#cdd3cd"}"/>
    ${active ? `<path d="M${w-33} 49 l4 4 l8 -9" fill="none" stroke="#fff" stroke-width="2.5"/>` : ""}
  </g>`;
}

function sceneSources(t) {
  const local = t - 9.4;
  const a = inOut(local, 0, 7.5, .6);
  const zoom = mix(.91, 1.02, ease(clamp(local / 7)));
  const typed = "How cities remember their rivers".slice(0, Math.max(0, Math.floor((local - .7) * 15)));
  const selected = local > 3.25;
  const panel = `<rect width="1240" height="658" fill="#fbfbf8"/>
    <rect width="238" height="658" fill="#f5f7f2"/>
    ${brandMark(27, 26, 34)}${text(73, 51, "Learnloom", 18, { weight: 720, tracking: -.8 })}
    ${text(29, 112, "YOUR LEARNING", 10, { fill: "#8c958e", weight: 750, tracking: 1.4 })}
    ${text(29, 157, "Today", 15, { weight: 650 })}${text(29, 199, "Newsletters", 15, { fill: colors.muted })}
    ${text(29, 241, "Learning History", 15, { fill: colors.muted })}
    ${text(286, 71, "Create a learning stream", 30, { weight: 680, tracking: -1.2 })}
    ${text(286, 105, "Choose a question, then add the sources you trust.", 15, { fill: colors.muted })}
    <rect x="286" y="145" width="880" height="64" rx="15" fill="#fff" stroke="${typed ? "#9eb49b" : "#dfe4de"}" stroke-width="2"/>
    ${text(312, 185, typed || "What do you want to understand?", 18, { fill: typed ? colors.ink : "#99a29b", weight: 500 })}
    ${typed && local < 3.2 ? `<rect x="${312 + typed.length * 9.2}" y="163" width="2" height="28" fill="${colors.green}"/>` : ""}
    ${text(286, 254, "Trusted sources", 13, { fill: colors.green, weight: 750, tracking: 1.2 })}
    ${sourceCard(286, 278, 420, "Urban waterways", "City research archive", selected)}
    ${sourceCard(730, 278, 420, "River restoration", "Environmental journal", local > 3.9)}
    ${sourceCard(286, 394, 420, "Street morphology", "Open planning review", local > 4.5)}
    ${sourceCard(730, 394, 420, "Flood histories", "Municipal records", local > 5.1)}
    <rect x="884" y="548" width="266" height="58" rx="29" fill="${selected ? colors.ink : "#d5d9d4"}"/>
    ${text(1017, 585, "Build my Dossier  →", 15, { anchor: "middle", fill: "#fff", weight: 700 })}`;
  return `<rect width="${W}" height="${H}" fill="#fff"/>
    <g opacity="${a}" transform="translate(960 540) scale(${zoom}) translate(-960 -540)">
      ${browser(340, 180, 1240, 720, panel, { title: "app.learnloom.blog" })}
    </g>`;
}

function sceneWeave(t) {
  const local = t - 16.2;
  const a = inOut(local, 0, 5.8, .6);
  const progress = ease(clamp(local / 4.6));
  let beams = "";
  const left = [
    [330, 350, "Urban waterways"],
    [300, 530, "River restoration"],
    [380, 710, "Flood histories"],
  ];
  left.forEach(([x, y, label], i) => {
    const p = ease(clamp((local - i * .35) / 2.2));
    const endX = mix(x + 260, 878, p);
    const endY = mix(y, 535, p);
    beams += `<path d="M${x+210} ${y} C700 ${y},720 ${endY},${endX} ${endY}" fill="none" stroke="#8eb085" stroke-width="2.5" opacity="${.22 + p*.62}"/>`;
    beams += `<g opacity="${inOut(local, 0, 5.6, .5)}"><rect x="${x}" y="${y-27}" width="230" height="54" rx="27" fill="#f7faf5" stroke="#73916f"/>
      ${text(x+115, y+6, label, 14, { anchor: "middle", fill: "#dfeade", weight: 600 })}</g>`;
  });
  const rings = [0,1,2,3].map(i => `<circle cx="960" cy="535" r="${72+i*42}" fill="none" stroke="#a8c699" stroke-width="2" opacity="${Math.max(0, .35-i*.06) * (1-progress*.25)}"/>`).join("");
  return `${starfield(t)}<g opacity="${a}">${beams}${rings}
    <circle cx="960" cy="535" r="${66+Math.sin(local*3)*3}" fill="#edf5e9" opacity=".98"/>
    ${brandMark(929, 504, 62, false)}
    <g transform="translate(1210 345)" opacity="${ease(clamp((local-2.2)/.8))}">
      <rect width="380" height="380" rx="24" fill="#fbfcf9" filter="url(#shadow)"/>
      ${text(32, 53, "LEARNING BLUEPRINT", 11, { fill: colors.green, weight: 780, tracking: 1.6 })}
      ${text(32, 93, "Why cities remember", 26, { weight: 680, tracking: -1 })}
      ${text(32, 126, "the shape of their rivers", 26, { weight: 680, tracking: -1 })}
      ${["The mechanism", "A worked example", "Skeptical review", "Test your model"].map((s,i)=>`
        <circle cx="43" cy="${182+i*45}" r="12" fill="${i < progress*4 ? colors.green : "#e8ece6"}"/>
        ${i < progress*4 ? `<path d="M38 ${182+i*45} l4 4 l8 -9" fill="none" stroke="#fff" stroke-width="2.2"/>` : ""}
        ${text(69, 188+i*45, s, 15, { fill: i < progress*4 ? colors.ink : "#929a93", weight: 580 })}`).join("")}
    </g>
    ${text(960, 925, "Trusted sources, woven into a lesson built for understanding.", 22, { anchor: "middle", fill: "#e9f0e7", weight: 500, tracking: -.4 })}
  </g>`;
}

function dossierContent(reveal) {
  const section = (y, num, title, body) => `<g opacity="${ease(clamp((reveal-num*.16)/.24))}">
    ${text(100, y, `0${num}`, 12, { fill: colors.green, weight: 750, tracking: 1.5 })}
    ${text(150, y, title, 20, { weight: 680, tracking: -.5 })}
    ${text(150, y+34, body, 13, { fill: colors.muted })}
    <line x1="100" y1="${y+62}" x2="1000" y2="${y+62}" stroke="#e2e6e0"/>
  </g>`;
  return `<rect width="1240" height="658" fill="#fdfcf8"/>
    <rect width="1240" height="66" fill="#fff"/>
    ${brandMark(28, 17, 32)}${text(72, 41, "Maya’s Learning Garden", 17, { weight: 720 })}
    ${text(1010, 41, "Topics    Archive    About", 13, { fill: colors.muted })}
    <g transform="translate(110 110)">
      ${text(0, 0, "TODAY’S DOSSIER  ·  8 MIN READ", 11, { fill: colors.green, weight: 770, tracking: 1.6 })}
      ${text(0, 62, "Why cities remember", 47, { family: "Georgia, serif", weight: 400, tracking: -1.7 })}
      ${text(0, 116, "the shape of their rivers", 47, { family: "Georgia, serif", weight: 400, tracking: -1.7 })}
      ${text(0, 153, "Urban Systems  ·  July 19", 13, { fill: colors.muted })}
      <rect x="790" y="-8" width="230" height="150" rx="22" fill="#dde9da"/>
      <path d="M790 119 C850 64,900 159,1020 72 L1020 142 L790 142Z" fill="#88a482"/>
      <path d="M812 142 C840 103,872 87,900 113 C930 142,954 86,1010 56" fill="none" stroke="#f0f4e8" stroke-width="10" opacity=".9"/>
      ${section(222, 1, "The mechanism", "Buried waterways continue to shape streets, density, and risk.")}
      ${section(302, 2, "A worked example", "Tracing old streams beneath modern Bengaluru.")}
      ${section(382, 3, "Skeptical review", "What this model explains—and where it can mislead.")}
      ${section(462, 4, "Test your model", "Three retrieval questions and one field observation.")}
    </g>`;
}

function sceneDossier(t) {
  const local = t - 21.1;
  const a = inOut(local, 0, 7.1, .6);
  const reveal = clamp(local / 5.1);
  const zoom = mix(.88, 1.04, ease(clamp(local / 7)));
  return `<rect width="${W}" height="${H}" fill="#fff"/>
    <g opacity="${a}" transform="translate(960 540) scale(${zoom}) translate(-960 -540)">
      ${browser(340, 180, 1240, 720, dossierContent(reveal), { title: "maya.learnloom.blog" })}
    </g>
    <g opacity="${ease(clamp((local-4.4)/.5))}">
      ${pill(1238, 820, 248, "Sources attached to every claim", "#eef4eb", colors.green)}
    </g>`;
}

function sceneHistory(t) {
  const local = t - 27.5;
  const a = inOut(local, 0, 6.6, .55);
  let cards = "";
  const data = [
    ["Cities & Rivers", "TODAY", "#dbe8d8"],
    ["The Logic of Street Grids", "YESTERDAY", "#e7e1c8"],
    ["How Water Shapes Density", "JUL 17", "#d6e3e8"],
    ["Reading Urban Risk", "JUL 16", "#ead8ce"],
  ];
  data.forEach(([title, date, fill], i) => {
    const p = ease(clamp((local - i*.32)/.8));
    cards += `<g transform="translate(${430+i*220} ${mix(740, 410+i*15, p)}) rotate(${(i-1.5)*2.2})" opacity="${p}">
      <rect width="400" height="360" rx="24" fill="#fff" stroke="#dfe3dd" filter="url(#shadow)"/>
      <rect x="22" y="22" width="356" height="155" rx="16" fill="${fill}"/>
      <path d="M45 153 C110 83,165 151,235 86 C276 48,318 86,356 60" fill="none" stroke="${colors.green}" stroke-width="5" opacity=".62"/>
      ${text(28, 214, date, 11, { fill: colors.green, weight: 770, tracking: 1.3 })}
      ${text(28, 257, title, 25, { weight: 670, tracking: -1 })}
      ${text(28, 298, "Mechanism · Example · Practice", 13, { fill: colors.muted })}
      <rect x="28" y="322" width="155" height="4" rx="2" fill="${colors.lime}"/>
    </g>`;
  });
  return `<rect width="${W}" height="${H}" fill="${colors.paper}"/>
    <g opacity="${a}">
      ${text(960, 174, "Every lesson builds on the last.", 68, { anchor: "middle", weight: 680, tracking: -3.2 })}
      ${text(960, 226, "Your Learning History creates continuity—not repetition.", 20, { anchor: "middle", fill: colors.muted })}
      ${cards}
    </g>`;
}

function sceneEverywhere(t) {
  const local = t - 33.3;
  const a = inOut(local, 0, 6.4, .55);
  const p1 = ease(clamp((local-.5)/.8));
  const p2 = ease(clamp((local-1.4)/.8));
  return `${starfield(t)}<g opacity="${a}">
    ${text(960, 147, "A home on the web. A nudge in your inbox.", 58, { anchor: "middle", fill: "#f0f5ed", weight: 670, tracking: -2.6 })}
    <g transform="translate(${mix(160, 245, p1)} 260) scale(.72)" opacity="${p1}">
      ${browser(0, 0, 1100, 650, dossierContent(1), { title: "maya.learnloom.blog" })}
    </g>
    <g transform="translate(${mix(1410, 1320, p2)} 310)" opacity="${p2}" filter="url(#shadow)">
      <rect width="390" height="570" rx="54" fill="#fbfcfa" stroke="#d8ded8" stroke-width="3"/>
      <rect x="134" y="17" width="122" height="26" rx="13" fill="#101510"/>
      ${text(31, 83, "9:41", 13, { weight: 700 })}${text(195, 121, "Today’s Dossier", 17, { anchor: "middle", weight: 700 })}
      <line x1="24" y1="145" x2="366" y2="145" stroke="#e2e6e1"/>
      ${brandMark(30, 178, 44)}${text(87, 204, "Learnloom", 18, { weight: 700 })}
      ${text(30, 275, "Why cities remember", 28, { weight: 680, tracking: -1.1 })}
      ${text(30, 310, "the shape of their rivers", 28, { weight: 680, tracking: -1.1 })}
      ${text(30, 351, "Your 8-minute Dossier is ready.", 14, { fill: colors.muted })}
      <rect x="30" y="390" width="330" height="54" rx="27" fill="${colors.ink}"/>
      ${text(195, 424, "Continue reading  →", 15, { anchor: "middle", fill: "#fff", weight: 700 })}
      ${text(195, 501, "maya.learnloom.blog", 13, { anchor: "middle", fill: colors.green, weight: 650 })}
    </g>
  </g>`;
}

function sceneFinal(t) {
  const local = t - 39.0;
  const a = ease(clamp(local/.7));
  const fade = ease(clamp((46-t)/.55));
  const y = mix(570, 525, ease(clamp(local/1.2)));
  return `<rect width="${W}" height="${H}" fill="url(#sky)"/>
    <circle cx="240" cy="940" r="560" fill="#688968" opacity=".33"/>
    <circle cx="1660" cy="990" r="610" fill="#55775c" opacity=".42"/>
    <path d="M0 860 C340 710,530 865,800 755 C1110 630,1310 770,1920 590 L1920 1080 L0 1080Z" fill="#668166" opacity=".55"/>
    <path d="M0 940 C380 765,640 930,950 820 C1320 690,1580 840,1920 720 L1920 1080 L0 1080Z" fill="#344f3b" opacity=".64"/>
    <g opacity="${a*fade}">
      ${brandMark(909, 210, 62)}${text(984, 253, "Learnloom", 42, { weight: 720, tracking: -1.8 })}
      ${lineText(960, y, ["Make curiosity", "a place you return to."], 82, 88, { weight: 660, tracking: -4.2 })}
      ${text(960, 742, "Claim your learning home and publish your first Dossier.", 21, { anchor: "middle", fill: "#435648", weight: 500 })}
      <g transform="translate(770 790)"><rect width="380" height="68" rx="34" fill="${colors.ink}" filter="url(#shadow)"/>
        ${text(190, 43, "Start learning at learnloom.blog  →", 16, { anchor: "middle", fill: "#fff", weight: 720 })}</g>
    </g>`;
}

function render(t) {
  let body;
  if (t < 3.5) body = sceneLogo(t);
  else if (t < 10.0) body = sceneManifesto(t);
  else if (t < 16.8) body = sceneSources(t);
  else if (t < 21.7) body = sceneWeave(t);
  else if (t < 28.0) body = sceneDossier(t);
  else if (t < 33.9) body = sceneHistory(t);
  else if (t < 39.5) body = sceneEverywhere(t);
  else body = sceneFinal(t);
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">
    ${defs()}${body}
  </svg>`;
}

for (let i = 0; i < TOTAL; i++) {
  const file = path.join(FRAMES, `frame-${String(i).padStart(5, "0")}.svg`);
  fs.writeFileSync(file, render(i / FPS));
  if (i % 150 === 0) process.stdout.write(`Generated ${i}/${TOTAL}\n`);
}

const svgFiles = fs.readdirSync(FRAMES).map((name) => path.join(FRAMES, name));
for (let i = 0; i < svgFiles.length; i += 60) {
  execFileSync("sips", ["-s", "format", "png", ...svgFiles.slice(i, i + 60), "--out", PNGS], {
    stdio: "ignore",
  });
  process.stdout.write(`Rasterized ${Math.min(i + 60, TOTAL)}/${TOTAL}\n`);
}

const audio = path.join(OUT, "soundtrack.m4a");
execFileSync("ffmpeg", [
  "-y",
  "-f", "lavfi", "-i", `sine=frequency=110:sample_rate=48000:duration=${DURATION}`,
  "-f", "lavfi", "-i", `sine=frequency=164.81:sample_rate=48000:duration=${DURATION}`,
  "-f", "lavfi", "-i", `sine=frequency=220:sample_rate=48000:duration=${DURATION}`,
  "-f", "lavfi", "-i", `anoisesrc=color=pink:sample_rate=48000:duration=${DURATION}`,
  "-filter_complex",
  `[0:a]volume=.055,tremolo=f=0.10:d=.28,lowpass=f=420[a0];` +
  `[1:a]volume=.035,tremolo=f=0.12:d=.35,lowpass=f=520[a1];` +
  `[2:a]volume=.018,tremolo=f=0.11:d=.4,lowpass=f=720[a2];` +
  `[3:a]volume=.008,lowpass=f=1100[a3];` +
  `[a0][a1][a2][a3]amix=inputs=4,afade=t=in:st=0:d=2,afade=t=out:st=43:d=3,` +
  `aecho=0.8:0.55:700|1200:0.14|0.08,alimiter=limit=.7[a]`,
  "-map", "[a]", "-c:a", "aac", "-b:a", "192k", audio,
], { stdio: "inherit" });

const video = path.join(OUT, "learnloom-launch-film.mp4");
execFileSync("ffmpeg", [
  "-y", "-framerate", String(FPS), "-i", path.join(PNGS, "frame-%05d.png"),
  "-i", audio, "-c:v", "libx264", "-preset", "medium", "-crf", "18",
  "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "192k",
  "-movflags", "+faststart", "-shortest", video,
], { stdio: "inherit" });

process.stdout.write(`\nCreated ${video}\n`);
