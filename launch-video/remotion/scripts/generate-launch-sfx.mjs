import {spawnSync} from "node:child_process";
import {dirname, resolve} from "node:path";
import {fileURLToPath} from "node:url";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const publicDirectory = resolve(scriptDirectory, "../public");
const sourceDirectory = resolve(publicDirectory, "sfx-source");
const output = resolve(publicDirectory, "launch-sfx-v1.wav");

/*
 * Source recordings: "Keyboard Soundpack #1" by unicae_games (CC0).
 * Recorded from a Cherry KC 1000 with a Shure SM7B. We use only the
 * three softest human performances and three restrained single keys.
 *
 * The mix deliberately avoids synthetic chirps, UI chimes, and whooshes.
 * These sounds should sit under the music and be felt as physical texture.
 */
const source = {
  typing01: resolve(sourceDirectory, "human-soft-01.wav"),
  typing02: resolve(sourceDirectory, "human-soft-02.wav"),
  typing03: resolve(sourceDirectory, "human-soft-03.wav"),
  key01: resolve(sourceDirectory, "key-soft-01.wav"),
  key02: resolve(sourceDirectory, "key-soft-02.wav"),
  key03: resolve(sourceDirectory, "key-soft-03.wav"),
};

const events = [
  // Opening typography and the topic composer.
  {file: source.typing01, kind: "typing", sourceStart: 0.10, duration: 0.84, at: 0.12, gain: 0.82, pan: -0.04},
  {file: source.typing02, kind: "typing", sourceStart: 1.15, duration: 1.08, at: 1.20, gain: 0.78, pan: 0.03},
  {file: source.typing03, kind: "typing", sourceStart: 0.30, duration: 1.40, at: 2.66, gain: 0.78, pan: -0.02},
  {file: source.typing01, kind: "typing", sourceStart: 3.55, duration: 1.82, at: 4.58, gain: 0.72, pan: 0.02},
  {file: source.typing02, kind: "typing", sourceStart: 4.58, duration: 2.22, at: 7.20, gain: 0.80, pan: -0.03},

  // One tactile selection cue for "Daily".
  {file: source.key01, kind: "click", sourceStart: 0, duration: 0.22, at: 9.84, gain: 0.70, pan: 0.08},

  // A few restrained physical ticks as the research tasks resolve.
  {file: source.key01, kind: "tick", sourceStart: 0, duration: 0.18, at: 11.92, gain: 0.32, pan: -0.10},
  {file: source.key02, kind: "tick", sourceStart: 0, duration: 0.18, at: 12.86, gain: 0.26, pan: 0.08},
  {file: source.key01, kind: "tick", sourceStart: 0, duration: 0.18, at: 13.80, gain: 0.30, pan: -0.04},
  {file: source.key03, kind: "tick", sourceStart: 0, duration: 0.18, at: 14.74, gain: 0.20, pan: 0.10},
  {file: source.key02, kind: "tick", sourceStart: 0, duration: 0.18, at: 15.52, gain: 0.24, pan: 0},

  // Publish confirmation: a single muted physical response, not a chime.
  {file: source.key01, kind: "click", sourceStart: 0, duration: 0.22, at: 25.16, gain: 0.42, pan: 0.04},

  // The personal learning home and final typed close.
  {file: source.typing03, kind: "typing", sourceStart: 3.60, duration: 1.04, at: 33.58, gain: 0.66, pan: 0.02},
  {file: source.typing01, kind: "typing", sourceStart: 6.25, duration: 2.10, at: 38.82, gain: 0.70, pan: -0.02},
];

const ffmpegArguments = ["-y", "-hide_banner", "-loglevel", "error"];
for (const event of events) ffmpegArguments.push("-i", event.file);

const filters = events.map((event, index) => {
  const fadeOutStart = Math.max(0, event.duration - 0.08).toFixed(3);
  const delay = Math.round(event.at * 1000);
  const left = Math.sqrt((1 - event.pan) / 2).toFixed(4);
  const right = Math.sqrt((1 + event.pan) / 2).toFixed(4);
  const tone =
    event.kind === "typing"
      ? "highpass=f=150:p=2,lowpass=f=4200:p=2,equalizer=f=2450:t=q:w=0.85:g=-3"
      : "highpass=f=110:p=2,lowpass=f=2900:p=2,equalizer=f=1900:t=q:w=1:g=-2";
  const fadeIn = event.kind === "typing" ? 0.025 : 0.004;

  const chain = [
    `atrim=start=${event.sourceStart}:duration=${event.duration}`,
    "asetpts=PTS-STARTPTS",
    "aresample=48000",
    tone,
    `volume=${event.gain}`,
    `afade=t=in:st=0:d=${fadeIn}`,
    `afade=t=out:st=${fadeOutStart}:d=0.08`,
    `pan=stereo|c0=${left}*c0|c1=${right}*c0`,
    `adelay=${delay}|${delay}`,
  ].join(",");
  return `[${index}:a]${chain}[event${index}]`;
});

const mixInputs = events.map((_, index) => `[event${index}]`).join("");
filters.push(
  `${mixInputs}amix=inputs=${events.length}:duration=longest:dropout_transition=0:normalize=0,` +
    "apad=whole_dur=45.2,atrim=duration=45.2[out]",
);

ffmpegArguments.push(
  "-filter_complex",
  filters.join(";"),
  "-map",
  "[out]",
  "-ar",
  "48000",
  "-ac",
  "2",
  "-c:a",
  "pcm_s16le",
  output,
);

const result = spawnSync("ffmpeg", ffmpegArguments, {
  encoding: "utf8",
  stdio: ["ignore", "pipe", "pipe"],
});

if (result.status !== 0) {
  process.stderr.write(result.stderr);
  process.exit(result.status ?? 1);
}

console.log(output);
