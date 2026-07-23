import React from "react";
import {
  AbsoluteFill,
  Audio,
  Img,
  Sequence,
  Easing,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
} from "remotion";

const FPS = 30;
const W = 1920;
const H = 1080;

const palette = {
  ink: "#17211b",
  muted: "#788078",
  paper: "#f8f7f2",
  warm: "#eeeee7",
  white: "#ffffff",
  green: "#2c5139",
  greenDark: "#0d1911",
  greenSoft: "#dfead9",
  lime: "#b9d18d",
};

const asset = (name) => staticFile(name);
const clamp = (value) => Math.max(0, Math.min(1, value));
const appear = (frame, start = 0, duration = 18) =>
  interpolate(frame, [start, start + duration], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
const disappear = (frame, start, duration = 18) =>
  1 - interpolate(frame, [start, start + duration], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.in(Easing.cubic),
  });
const typeText = (value, frame, start, frames, speed = 1) => {
  const amount = Math.floor(clamp((frame - start) / frames) * value.length * speed);
  return value.slice(0, Math.min(value.length, amount));
};
const fontSans = "Inter, Helvetica Neue, Arial, sans-serif";
const fontSerif = "Iowan Old Style, Palatino Linotype, Georgia, serif";

const base = {
  boxSizing: "border-box",
  fontFamily: fontSans,
};

export function LearnloomLaunch({ sound = true }) {
  return (
    <AbsoluteFill style={{ ...base, backgroundColor: palette.paper }}>
      {sound ? <Audio src={asset("soundtrack.m4a")} volume={0.18} /> : null}
      <Sequence from={0} durationInFrames={120}>
        <OpeningQuestion />
      </Sequence>
      <Sequence from={96} durationInFrames={99}>
        <BrandReveal />
      </Sequence>
      <Sequence from={170} durationInFrames={170}>
        <PromptScene />
      </Sequence>
      <Sequence from={315} durationInFrames={270}>
        <AutonomousPathScene />
      </Sequence>
      <Sequence from={560} durationInFrames={190}>
        <DossierScene />
      </Sequence>
      <Sequence from={725} durationInFrames={175}>
        <SubdomainScene />
      </Sequence>
      <Sequence from={880} durationInFrames={200}>
        <ClosingScene />
      </Sequence>
    </AbsoluteFill>
  );
}

function OpeningQuestion() {
  const frame = useCurrentFrame();
  const first = typeText("AI can speed up your work.", frame, 8, 38);
  const second = typeText("But did it speed up your learning?", frame, 50, 42);
  const firstOpacity = disappear(frame, 55, 14);
  const secondOpacity = appear(frame, 65, 16);
  const sceneOpacity = disappear(frame, 94, 22);
  return (
    <AbsoluteFill style={{ ...centered, backgroundColor: palette.white, opacity: sceneOpacity }}>
      <div style={{ ...eyebrow, opacity: appear(frame, 0, 18) }}>A question for the age of AI</div>
      <div style={{ ...openingLine, opacity: firstOpacity }}>{first}<Cursor visible={frame < 55} /></div>
      <div style={{ ...openingLine, opacity: secondOpacity }}>{second}<Cursor visible={frame > 67 && frame < 95} /></div>
    </AbsoluteFill>
  );
}

function BrandReveal() {
  const frame = useCurrentFrame();
  const opacity = appear(frame, 2, 20) * disappear(frame, 72, 20);
  const scale = interpolate(appear(frame, 2, 25), [0, 1], [0.94, 1]);
  return (
    <AbsoluteFill style={{ ...centered, backgroundColor: palette.white, opacity }}>
      <div style={{ ...eyebrow, position: "static", marginBottom: 28 }}>Introducing</div>
      <div style={{ display: "flex", alignItems: "center", gap: 20, transform: `scale(${scale})` }}>
        <BrandMark size={70} />
        <div style={{ ...brandType, fontSize: 64 }}>Learnloom</div>
      </div>
      <div style={{ ...subtleLine, marginTop: 22 }}>A learning home that grows with you.</div>
    </AbsoluteFill>
  );
}

function PromptScene() {
  const frame = useCurrentFrame();
  const prompt = typeText("I want to learn about LLM inferencing.", frame, 28, 80);
  const cardIn = spring({ frame: frame - 4, fps: FPS, config: { damping: 18, stiffness: 90 } });
  const sent = appear(frame, 120, 22);
  const promptOpacity = disappear(frame, 143, 18);
  return (
    <AbsoluteFill style={{ backgroundColor: palette.paper, opacity: promptOpacity }}>
      <div style={{ ...sceneLabel, opacity: appear(frame, 0, 18) }}>Start with the thing you want to understand</div>
      <div style={{ ...promptTitle, opacity: appear(frame, 3, 20) }}>One question is enough.</div>
      <div style={{ ...promptCard, transform: `translateY(${interpolate(cardIn, [0, 1], [44, 0])}px) scale(${interpolate(cardIn, [0, 1], [0.97, 1])})`, opacity: cardIn }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, color: palette.muted, fontSize: 18 }}>
          <BrandMark size={32} />
          <span>Learnloom</span>
          <span style={{ marginLeft: "auto", fontSize: 14 }}>New learning path</span>
        </div>
        <div style={{ ...promptInput, marginTop: 28 }}>
          <span>{prompt}</span><Cursor visible={frame < 115} />
          <div style={{ ...sendButton, opacity: sent, transform: `scale(${interpolate(sent, [0, 1], [0.7, 1])})` }}>↑</div>
        </div>
        <div style={{ marginTop: 16, color: palette.muted, fontSize: 14, opacity: appear(frame, 90, 20) }}>
          Learnloom will find the signal, connect the ideas, and keep the thread.
        </div>
      </div>
      <div style={{ ...promptFoot, opacity: appear(frame, 110, 20) }}>The learner sets the direction.</div>
    </AbsoluteFill>
  );
}

function AutonomousPathScene() {
  const frame = useCurrentFrame();
  const fade = appear(frame, 0, 20) * disappear(frame, 230, 18);
  const progress = clamp((frame - 8) / 202);
  const sources = [
    { label: "Research paper", meta: "arXiv · 2024", x: 130, y: 340, delay: 0, tint: "#dce9f0" },
    { label: "Systems guide", meta: "The Batch", x: 110, y: 545, delay: 16, tint: "#e9e3c9" },
    { label: "Benchmark notes", meta: "Stanford HAI", x: 220, y: 755, delay: 32, tint: "#e3ead9" },
    { label: "Expert essay", meta: "Anthropic Research", x: 560, y: 885, delay: 48, tint: "#eadbd5" },
  ];
  const agents = [
    { label: "FIND", x: 555, y: 245, delay: 0 },
    { label: "READ", x: 760, y: 400, delay: 15 },
    { label: "COMPARE", x: 530, y: 625, delay: 30 },
    { label: "CONNECT", x: 785, y: 780, delay: 45 },
  ];
  const target = { x: 1280, y: 535 };
  return (
    <AbsoluteFill style={{ backgroundColor: palette.greenDark, color: palette.white, opacity: fade }}>
      <div style={{ ...sceneLabel, color: palette.lime, opacity: appear(frame, 0, 18) }}>Autonomous source learning</div>
      <div style={{ ...pathTitle, opacity: appear(frame, 5, 20) }}>It follows the path for you.</div>
      <div style={{ ...pathSubtitle, opacity: appear(frame, 18, 20) }}>Find the signal. Read deeply. Connect the ideas.</div>
      <svg width={W} height={H} style={{ position: "absolute", inset: 0, overflow: "visible" }}>
        {sources.map((source, index) => (
          <React.Fragment key={source.label}>
            <PathLine from={source} to={agents[index]} progress={clamp((progress - index * 0.09) * 1.55)} />
            <ScoutDot from={source} to={agents[index]} progress={clamp((progress - index * 0.09) * 1.55)} />
          </React.Fragment>
        ))}
        {agents.map((agent, index) => (
          <React.Fragment key={agent.label}>
            <PathLine from={agent} to={target} progress={clamp((progress - 0.24 - index * 0.09) * 1.55)} />
            <ScoutDot from={agent} to={target} progress={clamp((progress - 0.24 - index * 0.09) * 1.55)} />
          </React.Fragment>
        ))}
      </svg>
      {sources.map((source) => <SourceNode key={source.label} source={source} frame={frame} />)}
      {agents.map((agent) => <AgentNode key={agent.label} agent={agent} frame={frame} />)}
      <PathCore frame={frame} progress={progress} target={target} />
      <ReadyBurst frame={frame} target={target} />
      <div style={{ ...pathFoot, opacity: appear(frame, 118, 22) }}>A learning path takes shape.</div>
    </AbsoluteFill>
  );
}

function PathLine({ from, to, progress }) {
  const p = clamp(progress);
  const midX = (from.x + to.x) / 2;
  const startX = from.x + 150;
  const startY = from.y + 30;
  const endX = to.x;
  const endY = to.y + 30;
  return (
    <path
      d={`M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`}
      fill="none"
      stroke="#a7c99c"
      strokeWidth="2"
      strokeOpacity={0.18 + p * 0.56}
      strokeDasharray="7 11"
      strokeDashoffset={Math.round((1 - p) * 160)}
    />
  );
}

function ScoutDot({ from, to, progress }) {
  const p = clamp(progress);
  const point = cubicPoint(p, {
    x: from.x + 150,
    y: from.y + 30,
  }, {
    x: (from.x + to.x) / 2,
    y: from.y + 30,
  }, {
    x: (from.x + to.x) / 2,
    y: to.y + 30,
  }, {
    x: to.x,
    y: to.y + 30,
  });
  return (
    <circle
      cx={point.x}
      cy={point.y}
      r="5"
      fill={palette.lime}
      opacity={p > 0.02 && p < 0.99 ? 0.9 : 0}
      style={{ filter: "drop-shadow(0 0 8px rgba(185,209,141,.9))" }}
    />
  );
}

function cubicPoint(t, p0, p1, p2, p3) {
  const u = 1 - t;
  return {
    x: u ** 3 * p0.x + 3 * u ** 2 * t * p1.x + 3 * u * t ** 2 * p2.x + t ** 3 * p3.x,
    y: u ** 3 * p0.y + 3 * u ** 2 * t * p1.y + 3 * u * t ** 2 * p2.y + t ** 3 * p3.y,
  };
}

function SourceNode({ source, frame }) {
  const opacity = appear(frame, 18 + source.delay, 22);
  const y = source.y + Math.sin((frame + source.delay) / 20) * 4;
  return (
    <div style={{ ...sourceNode, left: source.x, top: y, opacity }}>
      <div style={{ ...sourceIcon, backgroundColor: source.tint }}>↗</div>
      <div><div style={sourceName}>{source.label}</div><div style={sourceMeta}>{source.meta}</div></div>
    </div>
  );
}

function AgentNode({ agent, frame }) {
  const opacity = appear(frame, 38 + agent.delay, 20);
  const pulse = 1 + Math.sin((frame - agent.delay) / 11) * 0.035;
  return (
    <div style={{ ...agentNode, left: agent.x, top: agent.y, opacity, transform: `scale(${pulse})` }}>
      <span style={agentDot} />{agent.label}
    </div>
  );
}

function PathCore({ frame, progress, target }) {
  const reveal = appear(frame, 100, 28);
  const cardScale = interpolate(reveal, [0, 1], [0.92, 1]);
  const ready = appear(frame, 190, 18);
  return (
    <div style={{ ...pathCore, left: target.x, top: target.y, opacity: reveal, transform: `translate(-50%, -50%) scale(${cardScale})` }}>
      <div style={{ ...coreOrb, transform: `rotate(${frame * 1.8}deg)` }}><BrandMark size={48} dark /></div>
      <div style={coreEyebrow}>{ready > 0.7 ? "DOSSIER READY" : "ASSEMBLING PATH"}</div>
      <div style={coreTitle}>{ready > 0.7 ? "Ready to learn" : "LLM inferencing"}</div>
      <div style={coreBody}>{ready > 0.7 ? "A focused lesson from the path." : "Mechanism · Systems · Trade-offs"}</div>
      <div style={coreProgress}><span style={{ width: `${Math.round(progress * 100)}%` }} /></div>
    </div>
  );
}

function ReadyBurst({ frame, target }) {
  const burst = clamp((frame - 188) / 32);
  if (burst <= 0 || burst >= 1) return null;
  return (
    <div style={{ position: "absolute", left: target.x, top: target.y, width: 10, height: 10, borderRadius: "50%", border: "1px solid rgba(185,209,141,.8)", opacity: 1 - burst, transform: `translate(-50%, -50%) scale(${1 + burst * 20})` }} />
  );
}

function DossierScene() {
  const frame = useCurrentFrame();
  const imageIn = spring({ frame: frame - 12, fps: FPS, config: { damping: 17, stiffness: 80 } });
  const opacity = appear(frame, 0, 20) * disappear(frame, 145, 25);
  const imageScale = interpolate(imageIn, [0, 1], [1.08, 1]);
  return (
    <AbsoluteFill style={{ backgroundColor: palette.paper, opacity }}>
      <div style={{ ...sceneLabel, opacity: appear(frame, 0, 18) }}>The path resolves into a lesson</div>
      <div style={{ ...dossierTitle, opacity: appear(frame, 5, 20) }}>The dossier is ready.</div>
      <div style={{ ...dossierSub, opacity: appear(frame, 18, 18) }}>A focused lesson, built from the path.</div>
      <div style={{ ...browserFrame, opacity: imageIn, transform: `translate(-50%, -50%) scale(${imageScale})` }}>
        <Img src={asset("11-demo-lesson.jpg")} style={imageStyle} />
      </div>
      <div style={{ ...dossierBadge, opacity: appear(frame, 115, 20) }}><span>✓</span> Source-grounded understanding</div>
    </AbsoluteFill>
  );
}

function SubdomainScene() {
  const frame = useCurrentFrame();
  const opacity = appear(frame, 0, 20) * disappear(frame, 148, 20);
  const windowIn = spring({ frame: frame - 16, fps: FPS, config: { damping: 18, stiffness: 78 } });
  const imageScale = interpolate(windowIn, [0, 1], [1.06, 1]);
  return (
    <AbsoluteFill style={{ backgroundColor: palette.paper, opacity }}>
      <div style={{ ...sceneLabel, opacity: appear(frame, 0, 18) }}>Your own learning home</div>
      <div style={{ ...siteTitle, opacity: appear(frame, 5, 20) }}>Every dossier lands<br />somewhere that’s yours.</div>
      <div style={{ ...siteBody, opacity: appear(frame, 25, 18) }}>A personal address for the ideas you want to keep.</div>
      <div style={{ ...siteAddress, opacity: appear(frame, 45, 18) }}><span style={siteAddressDot}>●</span> maya.learnloom.blog</div>
      <div style={{ ...siteNote, opacity: appear(frame, 64, 18) }}>Public or private.<br />Your choice.</div>
      <div style={{ ...siteWindow, opacity: windowIn, transform: `translateY(${interpolate(windowIn, [0, 1], [30, 0])}px) scale(${imageScale})` }}>
        <Img src={asset("06-personal-site.jpg")} style={siteImage} />
      </div>
    </AbsoluteFill>
  );
}

function ClosingScene() {
  const frame = useCurrentFrame();
  const opacity = appear(frame, 0, 24) * disappear(frame, 245, 12);
  const y = interpolate(appear(frame, 8, 26), [0, 1], [26, 0]);
  return (
    <AbsoluteFill style={{ ...centered, backgroundColor: palette.greenDark, color: palette.white, opacity }}>
      <BrandMark size={64} dark />
      <div style={{ ...closeBrand, marginTop: 18 }}>Learnloom</div>
      <div style={{ ...closeTitle, transform: `translateY(${y}px)` }}>Make curiosity<br />a place you return to.</div>
      <div style={closeSub}>A learning home that grows with you.</div>
      <div style={closeUrl}>learnloom.blog</div>
    </AbsoluteFill>
  );
}

function Cursor({ visible }) {
  return <span style={{ display: "inline-block", width: 3, height: "0.92em", marginLeft: 8, verticalAlign: "-0.08em", backgroundColor: palette.green, opacity: visible ? 1 : 0 }} />;
}

function BrandMark({ size = 40, dark = false }) {
  return (
    <div style={{ width: size, height: size, display: "grid", placeItems: "center", borderRadius: size * 0.27, color: dark ? palette.greenDark : palette.white, backgroundColor: dark ? palette.greenSoft : palette.ink, fontSize: size * 0.48, fontWeight: 700, boxShadow: dark ? "none" : "0 12px 28px rgba(23,33,27,.18)" }}>
      ✦
    </div>
  );
}

const centered = { display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center" };
const eyebrow = { position: "absolute", top: 350, color: palette.green, fontSize: 15, fontWeight: 750, letterSpacing: 3.2, textTransform: "uppercase" };
const openingLine = { position: "absolute", top: 462, width: 1500, color: palette.ink, fontSize: 72, fontWeight: 680, lineHeight: 1.04, letterSpacing: -3.8, textAlign: "center" };
const subtleLine = { color: palette.muted, fontSize: 20, letterSpacing: 0.2 };
const brandType = { color: palette.ink, fontWeight: 720, letterSpacing: -2.8 };
const sceneLabel = { position: "absolute", top: 105, left: 150, color: palette.green, fontSize: 15, fontWeight: 750, letterSpacing: 2.2, textTransform: "uppercase" };
const promptTitle = { position: "absolute", top: 165, left: 150, color: palette.ink, fontFamily: fontSerif, fontSize: 60, fontWeight: 600, letterSpacing: -2.6 };
const promptCard = { position: "absolute", top: 390, left: 280, width: 1360, minHeight: 310, padding: "32px 42px", border: "1px solid #dfe3dc", borderRadius: 26, backgroundColor: palette.white, boxShadow: "0 30px 80px rgba(40,54,43,.12)" };
const promptInput = { display: "flex", alignItems: "center", minHeight: 102, padding: "0 23px", border: "2px solid #9bb292", borderRadius: 18, color: palette.ink, fontFamily: fontSerif, fontSize: 34, letterSpacing: -1.1 };
const sendButton = { display: "grid", placeItems: "center", width: 54, height: 54, marginLeft: "auto", borderRadius: "50%", color: palette.white, backgroundColor: palette.ink, fontFamily: fontSans, fontSize: 28, fontWeight: 500 };
const promptFoot = { position: "absolute", bottom: 132, left: 150, color: palette.muted, fontSize: 17 };
const pathTitle = { position: "absolute", top: 160, left: 150, maxWidth: 1080, color: palette.white, fontFamily: fontSerif, fontSize: 58, fontWeight: 600, letterSpacing: -2.3, lineHeight: 1.02 };
const pathSubtitle = { position: "absolute", top: 255, left: 152, color: "#b8c5b8", fontSize: 18, letterSpacing: 0.1 };
const sourceNode = { position: "absolute", display: "flex", alignItems: "center", gap: 13, width: 260, padding: "12px 15px", border: "1px solid rgba(188,220,183,.18)", borderRadius: 15, backgroundColor: "rgba(242,249,240,.08)" };
const sourceIcon = { display: "grid", placeItems: "center", width: 34, height: 34, borderRadius: 10, color: palette.ink, fontSize: 17, fontWeight: 750 };
const sourceName = { color: palette.white, fontSize: 14, fontWeight: 650 };
const sourceMeta = { marginTop: 3, color: "#9caf9c", fontSize: 11 };
const agentNode = { position: "absolute", display: "flex", alignItems: "center", gap: 8, padding: "9px 13px", border: "1px solid rgba(185,209,141,.5)", borderRadius: 99, color: palette.lime, backgroundColor: "rgba(185,209,141,.08)", fontSize: 11, fontWeight: 800, letterSpacing: 1.2 };
const agentDot = { width: 7, height: 7, borderRadius: "50%", backgroundColor: palette.lime, boxShadow: "0 0 0 5px rgba(185,209,141,.12)" };
const pathCore = { position: "absolute", width: 420, padding: "28px 30px", border: "1px solid rgba(193,221,186,.45)", borderRadius: 24, color: palette.ink, backgroundColor: "#f8fbf6", boxShadow: "0 25px 70px rgba(0,0,0,.24)" };
const coreOrb = { display: "grid", placeItems: "center", width: 58, height: 58, marginBottom: 22, borderRadius: 17, backgroundColor: "#deecda" };
const coreEyebrow = { color: palette.green, fontSize: 10, fontWeight: 800, letterSpacing: 1.8 };
const coreTitle = { marginTop: 12, color: palette.ink, fontFamily: fontSerif, fontSize: 31, fontWeight: 600, letterSpacing: -1.1 };
const coreBody = { marginTop: 7, color: palette.muted, fontSize: 13 };
const coreProgress = { height: 5, marginTop: 25, overflow: "hidden", borderRadius: 99, backgroundColor: "#e3e9e0" };
const pathFoot = { position: "absolute", right: 150, bottom: 100, color: "#b8c5b8", fontFamily: fontSerif, fontSize: 23 };
const dossierTitle = { position: "absolute", top: 158, left: 150, color: palette.ink, fontFamily: fontSerif, fontSize: 58, fontWeight: 600, letterSpacing: -2.4 };
const dossierSub = { position: "absolute", top: 255, left: 153, color: palette.muted, fontSize: 18 };
const browserFrame = { position: "absolute", top: 700, left: 960, width: 1420, height: 800, overflow: "hidden", border: "12px solid #eef0eb", borderRadius: 24, backgroundColor: palette.white, boxShadow: "0 35px 100px rgba(31,47,36,.18)" };
const imageStyle = { display: "block", width: "100%", height: "100%", objectFit: "cover", objectPosition: "center top" };
const dossierBadge = { position: "absolute", right: 150, bottom: 100, display: "flex", alignItems: "center", gap: 9, padding: "13px 18px", borderRadius: 99, color: palette.green, backgroundColor: palette.greenSoft, fontSize: 13, fontWeight: 750 };
const siteTitle = { position: "absolute", top: 205, left: 150, color: palette.ink, fontFamily: fontSerif, fontSize: 61, fontWeight: 600, lineHeight: 1.01, letterSpacing: -2.7 };
const siteBody = { position: "absolute", top: 390, left: 154, maxWidth: 430, color: palette.muted, fontSize: 18, lineHeight: 1.5 };
const siteAddress = { position: "absolute", top: 495, left: 150, display: "flex", alignItems: "center", gap: 11, padding: "12px 16px", border: "1px solid #d8e3d4", borderRadius: 99, color: palette.green, backgroundColor: "#eef3eb", fontSize: 15, fontWeight: 700 };
const siteAddressDot = { color: palette.green, fontSize: 12 };
const siteNote = { position: "absolute", left: 150, bottom: 125, color: palette.green, fontFamily: fontSerif, fontSize: 25, lineHeight: 1.12 };
const siteWindow = { position: "absolute", top: 190, left: 800, width: 1060, height: 700, overflow: "hidden", border: "12px solid #eef0eb", borderRadius: 24, backgroundColor: palette.white, boxShadow: "0 30px 90px rgba(31,47,36,.17)" };
const siteImage = { display: "block", width: "100%", height: "100%", objectFit: "cover", objectPosition: "center top" };
const closeBrand = { color: palette.white, fontSize: 38, fontWeight: 720, letterSpacing: -1.4 };
const closeTitle = { marginTop: 110, color: palette.white, fontSize: 78, fontWeight: 660, lineHeight: 1.01, letterSpacing: -4.1, textAlign: "center" };
const closeSub = { marginTop: 26, color: "#b7c8b9", fontFamily: fontSerif, fontSize: 22 };
const closeUrl = { position: "absolute", bottom: 95, color: palette.lime, fontSize: 13, fontWeight: 750, letterSpacing: 2.1, textTransform: "uppercase" };
