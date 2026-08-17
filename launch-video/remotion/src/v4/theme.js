import {Easing, interpolate} from "remotion";

export const colors = {
  ink: "#10110f",
  white: "#ffffff",
  paper: "#f7f7f4",
  muted: "#8b8e88",
  line: "rgba(16,17,15,.14)",
  lime: "#d9ff72",
};

export const fade = (frame, start, end, from = 0, to = 1) =>
  interpolate(frame, [start, end], [from, to], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.22, 1, 0.36, 1),
  });

export const hold = (frame, inStart, inEnd, outStart, outEnd) =>
  interpolate(frame, [inStart, inEnd, outStart, outEnd], [0, 1, 1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.22, 1, 0.36, 1),
  });

