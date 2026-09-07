import {Easing, interpolate} from "remotion";
import {loadFont as loadManrope} from "@remotion/google-fonts/Manrope";
import {loadFont as loadBricolage} from "@remotion/google-fonts/BricolageGrotesque";

export const manrope = loadManrope("normal", {
  weights: ["400", "500", "600", "700", "800"],
  subsets: ["latin"],
}).fontFamily;

export const bricolage = loadBricolage("normal", {
  weights: ["500", "600", "700", "800"],
  subsets: ["latin"],
}).fontFamily;

export const colors = {
  white: "#ffffff",
  ink: "#10110f",
  paper: "#f7f7f4",
  muted: "#72766f",
  forest: "#1f4533",
  lime: "#d9ff72",
  green: "#77a81a",
  line: "rgba(16, 17, 15, 0.14)",
  hairline: "rgba(16, 17, 15, 0.1)",
};

export const ease = Easing.bezier(0.22, 1, 0.36, 1);

/** Clamped opacity helper on the shared ease curve. */
export const fade = (frame, start, end, from = 0, to = 1) =>
  interpolate(frame, [start, end], [from, to], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: ease,
  });

/** Clamped rise helper; returns pixels to use inside a translate(). */
export const rise = (frame, start, end, distance) =>
  interpolate(frame, [start, end], [distance, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: ease,
  });