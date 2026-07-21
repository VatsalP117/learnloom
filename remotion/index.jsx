import React from "react";
import { AbsoluteFill, Sequence, interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { Composition } from "remotion";
import "./style.css";

const sources = [
  { label: "Field notes", detail: "urban systems", color: "#a8c67f", x: 17, y: 32 },
  { label: "Research", detail: "primary sources", color: "#8fb5c9", x: 17, y: 57 },
  { label: "Signal", detail: "trusted feed", color: "#d6b77d", x: 17, y: 78 },
];

const fade = (frame, start, end) => interpolate(frame, [start, end], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
const rise = (frame, start, end, distance = 30) => interpolate(frame, [start, end], [distance, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });

function Brand({ light = false }) {
  return <div className="brand"><span className={light ? "brand-mark light" : "brand-mark"}>✦</span><span>Learnloom</span></div>;
}

function Intro() {
  const frame = useCurrentFrame();
  return <AbsoluteFill className="scene intro">
    <div className="grain" />
    <div className="intro-orb orb-one" /><div className="intro-orb orb-two" />
    <div className="intro-content" style={{ opacity: fade(frame, 0, 18), transform: `translateY(${rise(frame, 0, 18)}px)` }}>
      <Brand />
      <p className="eyebrow">A learning home that grows with you</p>
      <h1>Make curiosity<br /><em>a place you return to.</em></h1>
      <p className="lede">Learnloom turns the sources you trust into thoughtful, living Dossiers.</p>
    </div>
    <div className="intro-handoff" style={{ opacity: fade(frame, 45, 64) }}><span /> Your sources are waiting</div>
  </AbsoluteFill>;
}

function Network() {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const pulse = spring({ frame: frame - 34, fps, config: { damping: 16, stiffness: 80 } });
  const reveal = fade(frame, 8, 24);
  return <AbsoluteFill className="scene network-scene">
    <div className="network-header" style={{ opacity: reveal }}><Brand light /><span>Autonomous source intelligence</span></div>
    <div className="network-copy" style={{ opacity: fade(frame, 4, 18), transform: `translateY(${rise(frame, 4, 18)}px)` }}>
      <p className="eyebrow mint">One idea. Many signals.</p><h2>Learnloom follows<br /><em>the thread.</em></h2>
      <p>Agents search, compare, and connect the sources that matter—then shape the signal into something you can keep.</p>
    </div>
    <div className="network-field">
      <div className="orbit orbit-a" /><div className="orbit orbit-b" />
      <svg viewBox="0 0 900 700" className="connections" aria-hidden="true">
        {sources.map((source, i) => <line key={source.label} x1={`${source.x}%`} y1={`${source.y}%`} x2="58%" y2="51%" style={{ opacity: fade(frame, 18 + i * 5, 32 + i * 5), strokeDashoffset: interpolate(frame, [18 + i * 5, 36 + i * 5], [260, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }) }} />)}
        <line x1="58%" y1="51%" x2="84%" y2="51%" style={{ opacity: fade(frame, 55, 72) }} />
      </svg>
      {sources.map((source, i) => <div key={source.label} className="source-node" style={{ left: `${source.x}%`, top: `${source.y}%`, opacity: fade(frame, 12 + i * 5, 25 + i * 5), transform: `scale(${.8 + fade(frame, 12 + i * 5, 25 + i * 5) * .2})` }}><span className="node-dot" style={{ background: source.color }} /><strong>{source.label}</strong><small>{source.detail}</small><i>agent connected</i></div>)}
      <div className="agent-core" style={{ transform: `scale(${.75 + pulse * .25})`, opacity: fade(frame, 27, 43) }}><span>✦</span><strong>Learnloom<br />agent</strong><small>weaving context</small></div>
      <div className="dossier-bloom" style={{ opacity: fade(frame, 65, 82), transform: `translateX(${interpolate(frame, [65, 82], [30, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" })}px) scale(${.8 + fade(frame, 65, 82) * .2})` }}><span>✦</span><strong>Dossier<br />ready</strong><small>8 min read · 14 sources</small></div>
    </div>
    <div className="network-footer" style={{ opacity: fade(frame, 70, 88) }}><span className="status-dot" /> Signal gathered <i /> Understanding, ready to return to</div>
  </AbsoluteFill>;
}

function Home() {
  const frame = useCurrentFrame();
  const progress = fade(frame, 10, 32);
  return <AbsoluteFill className="scene home-scene">
    <div className="home-glow" />
    <div className="home-copy" style={{ opacity: progress, transform: `translateY(${rise(frame, 10, 32)}px)` }}><p className="eyebrow">Your knowledge, with somewhere to live</p><h2>A personal address<br /><em>for your becoming.</em></h2><p>Every Dossier lands in your own searchable learning home—beautiful, lasting, and yours.</p></div>
    <div className="browser" style={{ opacity: fade(frame, 22, 40), transform: `translateY(${rise(frame, 22, 40, 50)}px) rotate(-1.5deg)` }}>
      <div className="browser-bar"><span>● ● ●</span><b>⌁ maya.learnloom.blog</b><i>PUBLIC</i></div>
      <div className="site-nav"><Brand /><span>Topics &nbsp;&nbsp; Archive &nbsp;&nbsp; About</span></div>
      <div className="site-body"><p className="eyebrow">Today’s Dossier · 8 min read</p><h3>Why cities remember<br />the shape of their rivers</h3><p>Urban systems · July 19</p><div className="site-lines"><span /><span /><span /></div></div>
      <div className="site-card"><small>LEARNLOOM DOSSIER</small><strong>14</strong><span>issues gathered<br />in your garden</span></div>
    </div>
    <div className="address-pill" style={{ opacity: fade(frame, 43, 58) }}><span>●</span> maya.learnloom.blog <b>LIVE</b></div>
    <div className="final-lockup" style={{ opacity: fade(frame, 67, 84), transform: `translateY(${rise(frame, 67, 84)}px)` }}><Brand /><p>Current sources, woven into durable understanding.</p></div>
  </AbsoluteFill>;
}

export function LearnloomLaunch() {
  return <AbsoluteFill className="video"><Sequence from={0} durationInFrames={75}><Intro /></Sequence><Sequence from={60} durationInFrames={120}><Network /></Sequence><Sequence from={168} durationInFrames={132}><Home /></Sequence></AbsoluteFill>;
}

export const RemotionRoot = () => <Composition id="LearnloomLaunch" component={LearnloomLaunch} durationInFrames={300} fps={30} width={1920} height={1080} />;
