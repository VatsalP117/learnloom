import {Easing, interpolate} from "remotion";

export const FPS = 30;

export const colors = {
  ink: "#111210",
  paper: "#fbfbfa",
  white: "#ffffff",
  muted: "#6f726d",
  line: "rgba(17,18,16,0.13)",
  lime: "#d9ff72",
  lilac: "#dcd8ff",
  blush: "#f7ddd5",
  mint: "#dceee5",
};

export const clamp = (value) => Math.max(0, Math.min(1, value));

export const enter = (frame, start = 0, duration = 14) =>
  interpolate(frame, [start, start + duration], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.16, 1, 0.3, 1),
  });

export const exit = (frame, start, duration = 12) =>
  interpolate(frame, [start, start + duration], [1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.7, 0, 0.84, 0),
  });

export const grain = {
  position: "absolute",
  inset: 0,
  pointerEvents: "none",
  opacity: 0.2,
  backgroundImage:
    "repeating-linear-gradient(0deg,rgba(17,18,16,.025) 0,rgba(17,18,16,.025) 1px,transparent 1px,transparent 5px)",
};

export const eyebrow = {
  color: colors.muted,
  fontSize: 16,
  fontWeight: 700,
  letterSpacing: 3.2,
  textTransform: "uppercase",
};

export const sceneBase = {
  overflow: "hidden",
  color: colors.ink,
  backgroundColor: colors.paper,
};

// Canonical 16:10 product stages, matching the 1440x900 clips exactly.
export const stage = {left: 620, top: 150, width: 1216, height: 760};
export const hero = {left: 96, top: 0, width: 1728, height: 1080};
