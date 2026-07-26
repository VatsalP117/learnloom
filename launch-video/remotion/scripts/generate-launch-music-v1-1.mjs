import {mkdirSync, writeFileSync} from "node:fs";
import {dirname, resolve} from "node:path";
import {fileURLToPath} from "node:url";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const output = resolve(scriptDirectory, "../public/launch-music-v1-1.wav");
const sampleRate = 48_000;
const duration = 45.2;
const totalSamples = Math.round(sampleRate * duration);
const bpm = 128;
const beat = 60 / bpm;
const eighth = beat / 2;
const sixteenth = beat / 4;
const bar = beat * 4;

const TAU = Math.PI * 2;
const clamp = (value, min = -1, max = 1) => Math.max(min, Math.min(max, value));
const smoothstep = (value) => {
  const t = clamp(value, 0, 1);
  return t * t * (3 - 2 * t);
};
const fade = (time, start, end) => smoothstep((time - start) / (end - start));
const decay = (time, amount) => (time < 0 ? 0 : Math.exp(-time / amount));
const mod = (value, length) => ((value % length) + length) % length;
const eventPhase = (time, interval, offset = 0) => {
  if (time < offset) return -1;
  return mod(time - offset, interval);
};
const sine = (frequency, time) => Math.sin(TAU * frequency * time);
const saw = (frequency, time) => 2 * (frequency * time - Math.floor(frequency * time + 0.5));
const pan = (value, amount) => {
  const angle = (amount + 1) * Math.PI / 4;
  return [value * Math.cos(angle), value * Math.sin(angle)];
};

const midi = (note) => 440 * Math.pow(2, (note - 69) / 12);
const roots = [50, 50, 45, 45, 47, 47, 43, 43];
const arpeggio = [0, 4, 7, 11, 7, 4, 12, 7, 4, 7, 11, 14, 11, 7, 4, 7];

function noteForStep(step, notes, octave = 0) {
  const index = Math.floor(step / 16) % notes.length;
  return midi(notes[index] + octave);
}

function writeWav(samples) {
  const header = Buffer.alloc(44);
  header.write("RIFF", 0);
  header.writeUInt32LE(36 + samples.length * 2, 4);
  header.write("WAVE", 8);
  header.write("fmt ", 12);
  header.writeUInt32LE(16, 16);
  header.writeUInt16LE(1, 20);
  header.writeUInt16LE(2, 22);
  header.writeUInt32LE(sampleRate, 24);
  header.writeUInt32LE(sampleRate * 4, 28);
  header.writeUInt16LE(4, 32);
  header.writeUInt16LE(16, 34);
  header.write("data", 36);
  header.writeUInt32LE(samples.length * 2, 40);
  writeFileSync(output, Buffer.concat([header, samples]));
}

mkdirSync(dirname(output), {recursive: true});
const pcm = Buffer.alloc(totalSamples * 2 * 2);
let noiseState = 0x1a2b3c4d;
const nextNoise = () => {
  noiseState = (Math.imul(noiseState, 1_664_525) + 1_013_904_223) >>> 0;
  return (noiseState / 0x1_0000_0000) * 2 - 1;
};

for (let sample = 0; sample < totalSamples; sample++) {
  const time = sample / sampleRate;
  const beatNumber = Math.floor(time / beat);
  const eighthStep = Math.floor(time / eighth);
  const sixteenthStep = Math.floor(time / sixteenth);
  const barIndex = Math.floor(time / bar);
  const beatInBar = beatNumber % 4;
  const intro = fade(time, 0, 1.1);
  const outro = 1 - fade(time, duration - 1.7, duration);
  const fullEnergy = fade(time, 1.0, 2.2) * outro;

  let left = 0;
  let right = 0;

  // A soft, detuned chord bed keeps the score warm under the product story.
  const chordRoots = [50, 55, 57, 52];
  const chordRoot = chordRoots[barIndex % chordRoots.length];
  const chord = [chordRoot, chordRoot + 4, chordRoot + 7];
  let pad = 0;
  for (const note of chord) {
    const frequency = midi(note);
    pad += sine(frequency, time) + 0.22 * sine(frequency * 1.006, time);
  }
  const padEnvelope = (0.20 + 0.06 * Math.sin(TAU * time / 8.4)) * intro * outro;
  [left, right] = [left + pad * padEnvelope * 0.06, right + pad * padEnvelope * 0.06];

  // Pulsing bass gives the cut a confident, modern launch cadence.
  const bassPhase = mod(time, eighth);
  const bassStep = eighthStep % 16;
  const bassOffsets = [0, 0, 12, 0, 7, 0, 12, 0, 0, 0, 7, 0, 12, 0, 7, 0];
  const bassNote = roots[barIndex % roots.length] + bassOffsets[bassStep];
  const bassFrequency = midi(bassNote - 12);
  const bassEnvelope = Math.exp(-bassPhase / 0.18) * (0.72 + 0.28 * (bassStep % 2 === 0));
  const bass = (sine(bassFrequency, time) + 0.14 * saw(bassFrequency * 2, time)) * bassEnvelope * fullEnergy * 0.18;
  left += bass;
  right += bass;

  // Bright, short arpeggio notes add the fast-paced SF product-film feel.
  const arpPhase = eventPhase(time, sixteenth);
  const arpNote = roots[barIndex % roots.length] + arpeggio[sixteenthStep % arpeggio.length];
  const arpFrequency = midi(arpNote + 12);
  const arpEnvelope = arpPhase < 0 ? 0 : Math.exp(-arpPhase / 0.105) * fullEnergy;
  const arp = (sine(arpFrequency, time) + 0.28 * sine(arpFrequency * 2.01, time)) * arpEnvelope * 0.075;
  const [arpLeft, arpRight] = pan(arp, Math.sin(sixteenthStep * 1.7) * 0.42);
  left += arpLeft;
  right += arpRight;

  // Four-on-the-floor kick with a short click for definition on small speakers.
  const kickPhase = eventPhase(time, beat);
  if (kickPhase >= 0 && kickPhase < 0.34 && time > 0.7) {
    const kickEnvelope = Math.exp(-kickPhase / 0.115) * fullEnergy;
    const kickFrequency = 48 + 122 * Math.exp(-kickPhase / 0.028);
    const kick = (sine(kickFrequency, kickPhase) + 0.16 * sine(2 * kickFrequency, kickPhase)) * kickEnvelope * 0.48;
    const click = sine(1_900, kickPhase) * Math.exp(-kickPhase / 0.012) * 0.045;
    left += kick + click;
    right += kick + click;
  }

  // Tight clap on beats two and four; the high-frequency noise is deliberately restrained.
  const clapPhase = eventPhase(time, beat * 2, beat);
  if (clapPhase >= 0 && clapPhase < 0.20) {
    const clapEnvelope = Math.exp(-clapPhase / 0.038) * fullEnergy;
    const clap = nextNoise() * clapEnvelope * 0.055 + sine(190, clapPhase) * clapEnvelope * 0.035;
    const [clapLeft, clapRight] = pan(clap, -0.06);
    left += clapLeft;
    right += clapRight;
  }

  // Sixteenth-note hats are the main source of forward motion.
  const hatPhase = eventPhase(time, sixteenth);
  if (hatPhase >= 0 && hatPhase < 0.075) {
    const hatEnvelope = Math.exp(-hatPhase / (sixteenthStep % 4 === 3 ? 0.020 : 0.012)) * fullEnergy;
    const hat = nextNoise() * hatEnvelope * (sixteenthStep % 4 === 3 ? 0.032 : 0.021);
    const [hatLeft, hatRight] = pan(hat, sixteenthStep % 2 ? 0.28 : -0.28);
    left += hatLeft;
    right += hatRight;
  }

  // Small transition impacts land with the major visual chapters.
  const transitionTimes = [2.5, 4.5, 6.7, 11.1, 16.7, 25.0, 33.2, 37.8, 42.6];
  for (const transition of transitionTimes) {
    const phase = time - transition;
    if (phase >= 0 && phase < 0.42) {
      const impactEnvelope = Math.exp(-phase / 0.14) * 0.11;
      const impact = sine(78 - 35 * Math.min(phase / 0.42, 1), phase) * impactEnvelope;
      const air = nextNoise() * Math.exp(-phase / 0.035) * 0.018;
      left += impact + air;
      right += impact + air;
    }
  }

  // Sidechain-style ducking lets each kick read without making the mix aggressive.
  const kickDuck = kickPhase >= 0 && kickPhase < 0.16 ? 1 - 0.16 * Math.exp(-kickPhase / 0.08) : 1;
  left *= kickDuck;
  right *= kickDuck;

  const master = 0.86 * outro;
  const outLeft = Math.tanh(clamp(left * master) * 1.18);
  const outRight = Math.tanh(clamp(right * master) * 1.18);
  const offset = sample * 4;
  pcm.writeInt16LE(Math.round(clamp(outLeft) * 32_000), offset);
  pcm.writeInt16LE(Math.round(clamp(outRight) * 32_000), offset + 2);
}

writeWav(pcm);
console.log(output);
