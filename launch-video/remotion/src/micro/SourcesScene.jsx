import React from "react";
import {AbsoluteFill, interpolate, spring, useCurrentFrame} from "remotion";
import {colors, ease, fade, manrope} from "./theme.js";

/*
 * Layout (1920×1080 canvas):
 *   Knowledge Dossier artifact: 520×300, centered at (960, 530).
 *   MIT CSAIL: rendered left of the artifact, arriving from the left.
 *   Distill: rendered right of the artifact, arriving from the right.
 *   Model cards: rendered below the artifact, arriving from the bottom.
 * Thin lime-to-forest connectors draw from each source to the artifact edge.
 */

const DOSSIER = {left: 700, top: 380, width: 520, height: 300};

const cardBase = {
  position: "absolute",
  width: 264,
  padding: "18px 22px",
  borderRadius: 18,
  background: colors.white,
  border: `1px solid ${colors.hairline}`,
  boxShadow: "0 14px 36px rgba(16, 17, 15, 0.07)",
};

function SourceCard({appear, from, style, title, sub}) {
  const frame = useCurrentFrame();
  const s = spring({frame: frame - appear, fps: 30, config: {damping: 13, stiffness: 170, mass: 0.85}});
  return (
    <div
      style={{
        ...cardBase,
        ...style,
        transform: `translate(${(1 - s) * from[0]}px, ${(1 - s) * from[1]}px)`,
        opacity: interpolate(s, [0, 0.3], [0, 1], {
          extrapolateLeft: "clamp",
          extrapolateRight: "clamp",
        }),
      }}
    >
      <div style={{display: "flex", alignItems: "center", gap: 9}}>
        <span
          style={{
            width: 7,
            height: 7,
            borderRadius: 4,
            background: colors.lime,
            border: `1.5px solid ${colors.green}`,
            flexShrink: 0,
          }}
        />
        <span style={{fontSize: 16.5, fontWeight: 700, letterSpacing: -0.3, color: colors.ink}}>
          {title}
        </span>
      </div>
      <div style={{marginTop: 5, paddingLeft: 16, fontSize: 12.5, fontWeight: 500, color: colors.muted}}>
        {sub}
      </div>
    </div>
  );
}

function Connector({from, to, appear, drawTime = 10}) {
  const frame = useCurrentFrame();
  const [x1, y1] = from;
  const [x2, y2] = to;
  const length = Math.hypot(x2 - x1, y2 - y1);
  const angle = (Math.atan2(y2 - y1, x2 - x1) * 180) / Math.PI;
  const drawn = interpolate(frame, [appear, appear + drawTime], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: ease,
  });
  return (
    <div
      style={{
        position: "absolute",
        left: x1,
        top: y1 - 1,
        width: length,
        height: 2,
        borderRadius: 1,
        background: "linear-gradient(90deg, #d9ff72, #77a81a)",
        transform: `rotate(${angle}deg) scaleX(${drawn})`,
        transformOrigin: "0 50%",
        opacity: fade(frame, appear, appear + 3),
      }}
    />
  );
}

const dossierRows = [
  ["MIT CSAIL", "cited"],
  ["Distill", "cited"],
  ["Model cards", "cited"],
];

function DossierRow({name, status, at}) {
  const frame = useCurrentFrame();
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "11px 4px",
        opacity: fade(frame, at, at + 8),
        transform: `translateY(${fade(frame, at, at + 8, 10, 0)}px)`,
      }}
    >
      <span
        style={{
          width: 18,
          height: 18,
          borderRadius: 9,
          background: colors.lime,
          color: colors.forest,
          display: "grid",
          placeItems: "center",
          fontSize: 11,
          fontWeight: 800,
          flexShrink: 0,
        }}
      >
        ✓
      </span>
      <span style={{fontSize: 15.5, fontWeight: 600, letterSpacing: -0.2, color: colors.ink}}>
        {name}
      </span>
      <span style={{flex: 1}} />
      <span style={{fontSize: 12, fontWeight: 500, color: colors.muted}}>{status}</span>
    </div>
  );
}

export function SourcesScene() {
  const frame = useCurrentFrame();
  const dossierIn = spring({frame: frame - 40, fps: 30, config: {damping: 14, stiffness: 180, mass: 0.9}});
  const labelOpacity = fade(frame, 58, 66) * fade(frame, 70, 76, 1, 0);

  return (
    <AbsoluteFill>
      {/* Source fragments fly in from three directions. */}
      <SourceCard
        appear={0}
        from={[-620, 0]}
        style={{left: 268, top: 258}}
        title="MIT CSAIL"
        sub="research group"
      />
      <SourceCard
        appear={3}
        from={[620, 0]}
        style={{left: 1388, top: 258}}
        title="Distill"
        sub="journal"
      />
      <SourceCard
        appear={6}
        from={[0, 420]}
        style={{left: 828, top: 730}}
        title="Model cards"
        sub="documentation"
      />

      {/* Thin lime-to-forest paths into the central artifact. */}
      <Connector from={[534, 300]} to={[698, 460]} appear={24} />
      <Connector from={[1386, 300]} to={[1222, 460]} appear={27} />
      <Connector from={[962, 731]} to={[962, 685]} appear={30} />

      {/* The Knowledge Dossier assembles last, over the path ends. */}
      <div
        style={{
          position: "absolute",
          ...DOSSIER,
          padding: "24px 28px",
          borderRadius: 22,
          background: colors.white,
          border: `1px solid ${colors.line}`,
          boxShadow: "0 22px 60px rgba(16, 17, 15, 0.1)",
          opacity: fade(frame, 40, 46),
          transform: `scale(${dossierIn}) translateY(${fade(frame, 40, 48, 14, 0)}px)`,
        }}
      >
        <div style={{display: "flex", alignItems: "center", gap: 10}}>
          <span style={{width: 9, height: 9, borderRadius: 3, background: colors.lime, flexShrink: 0}} />
          <span style={{fontSize: 12, fontWeight: 800, letterSpacing: 2.4, color: colors.muted}}>
            KNOWLEDGE DOSSIER
          </span>
        </div>
        <div style={{margin: "16px 0 8px", height: 1, background: colors.hairline}} />
        {dossierRows.map(([name, status], i) => (
          <DossierRow key={name} name={name} status={status} at={46 + i * 4} />
        ))}
      </div>

      <div
        style={{
          position: "absolute",
          left: 960,
          top: 905,
          transform: "translateX(-50%)",
          display: "flex",
          alignItems: "center",
          gap: 10,
          opacity: labelOpacity,
        }}
      >
        <span style={{width: 7, height: 7, borderRadius: 4, background: colors.lime, border: `1.5px solid ${colors.green}`}} />
        <span style={{fontFamily: manrope, fontSize: 16, fontWeight: 600, letterSpacing: 0.1, color: colors.muted}}>
          Sources stay visible.
        </span>
      </div>
    </AbsoluteFill>
  );
}