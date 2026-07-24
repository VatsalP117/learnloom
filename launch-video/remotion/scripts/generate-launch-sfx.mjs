import {mkdirSync, writeFileSync} from "node:fs";
import {dirname, resolve} from "node:path";
import {fileURLToPath} from "node:url";

const sampleRate = 48000;
const duration = 45.2;
const frames = Math.ceil(sampleRate * duration);
const left = new Float64Array(frames);
const right = new Float64Array(frames);
const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const output = resolve(scriptDirectory, "../public/launch-sfx-v1.wav");

let seed = 0x14f3a2c9;
const random = () => {
  seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0;
  return seed / 0xffffffff;
};
const clamp = (value, min, max) => Math.max(min, Math.min(max, value));

const addSound = ({start, length, gain, pan = 0, sample}) => {
  const firstFrame = Math.max(0, Math.floor(start * sampleRate));
  const soundFrames = Math.floor(length * sampleRate);
  const leftGain = Math.sqrt((1 - pan) / 2);
  const rightGain = Math.sqrt((1 + pan) / 2);
  for (let index = 0; index < soundFrames; index += 1) {
    const target = firstFrame + index;
    if (target >= frames) break;
    const time = index / sampleRate;
    const value = sample(time, index) * gain;
    left[target] += value * leftGain;
    right[target] += value * rightGain;
  }
};

const addKeyTap = (start, pan = 0, strength = 1) => {
  const pitch = 1080 + random() * 360;
  addSound({
    start,
    length: 0.055,
    gain: 0.17 * strength,
    pan,
    sample: (time) => {
      const transient = (random() * 2 - 1) * Math.exp(-time * 125);
      const body = Math.sin(Math.PI * 2 * pitch * time) * Math.exp(-time * 78);
      return transient * 0.46 + body * 0.54;
    },
  });
};

const addTypingRun = (start, end, rate = 10) => {
  let time = start;
  let index = 0;
  while (time < end) {
    addKeyTap(time, index % 2 === 0 ? -0.18 : 0.18, 0.7 + random() * 0.3);
    time += (1 / rate) * (0.78 + random() * 0.42);
    index += 1;
  }
};

const addClick = (start, pan = 0) => {
  addSound({
    start,
    length: 0.12,
    gain: 0.24,
    pan,
    sample: (time) => {
      const top = Math.sin(Math.PI * 2 * 920 * time) * Math.exp(-time * 75);
      const body = Math.sin(Math.PI * 2 * 260 * time) * Math.exp(-time * 42);
      return top * 0.72 + body * 0.28;
    },
  });
};

const addDataPulse = (start, pan = 0, pitch = 520) => {
  addSound({
    start,
    length: 0.26,
    gain: 0.15,
    pan,
    sample: (time) => {
      const envelope = (1 - Math.exp(-time * 160)) * Math.exp(-time * 15);
      const phase = Math.PI * 2 * (pitch * time + 210 * time * time);
      return (
        Math.sin(phase) * envelope +
        Math.sin(phase * 2.01) * envelope * 0.16
      );
    },
  });
};

const addWhoosh = (start, panStart = -0.35, panEnd = 0.35) => {
  const length = 0.82;
  let filteredNoise = 0;
  const firstFrame = Math.max(0, Math.floor(start * sampleRate));
  const soundFrames = Math.floor(length * sampleRate);
  for (let index = 0; index < soundFrames; index += 1) {
    const target = firstFrame + index;
    if (target >= frames) break;
    const time = index / sampleRate;
    const progress = time / length;
    const envelope = Math.sin(Math.PI * progress) ** 1.7;
    filteredNoise = filteredNoise * 0.91 + (random() * 2 - 1) * 0.09;
    const tone = Math.sin(
      Math.PI * 2 * (170 * time + 330 * time * time),
    );
    const value = (filteredNoise * 0.72 + tone * 0.28) * envelope * 0.12;
    const pan = panStart + (panEnd - panStart) * progress;
    left[target] += value * Math.sqrt((1 - pan) / 2);
    right[target] += value * Math.sqrt((1 + pan) / 2);
  }
};

const addChime = (start, warm = false) => {
  const notes = warm ? [587.33, 739.99, 880] : [523.25, 659.25, 783.99];
  notes.forEach((frequency, index) => {
    addSound({
      start: start + index * 0.055,
      length: 1.15,
      gain: 0.075,
      pan: (index - 1) * 0.22,
      sample: (time) => {
        const envelope =
          (1 - Math.exp(-time * 90)) * Math.exp(-time * (3.4 + index * 0.3));
        return (
          Math.sin(Math.PI * 2 * frequency * time) * envelope +
          Math.sin(Math.PI * 4 * frequency * time) * envelope * 0.12
        );
      },
    });
  });
};

[
  [0.13, 0.97, 10],
  [1.23, 2.3, 10],
  [2.7, 4.1, 10.5],
  [4.63, 6.43, 10.5],
  [7.27, 9.47, 11],
  [33.63, 34.63, 10],
  [38.87, 40.93, 10.5],
].forEach(([start, end, rate]) => addTypingRun(start, end, rate));

addClick(9.87, 0.18);

[
  [11.5, -0.35, 480],
  [11.97, 0.3, 540],
  [12.43, -0.08, 600],
  [12.6, -0.28, 540],
  [13.4, -0.08, 580],
  [14.2, 0.12, 620],
  [15.0, 0.3, 680],
  [17.2, -0.24, 560],
  [17.8, -0.08, 600],
  [18.4, 0.1, 640],
  [19.0, 0.26, 690],
  [28.0, -0.2, 560],
  [28.5, 0, 620],
  [29.0, 0.2, 680],
  [34.43, -0.18, 590],
  [34.73, 0, 640],
  [35.03, 0.18, 690],
].forEach(([start, pan, pitch]) => addDataPulse(start, pan, pitch));

addWhoosh(16.13);
addWhoosh(20.88, 0.25, -0.2);
addWhoosh(27.45, -0.18, 0.22);
addWhoosh(33.12, 0.22, -0.16);
addWhoosh(37.76, -0.12, 0.12);

addChime(25.05);
addChime(33.25, true);

let peak = 0;
for (let index = 0; index < frames; index += 1) {
  peak = Math.max(peak, Math.abs(left[index]), Math.abs(right[index]));
}
const targetPeak = 10 ** (-6 / 20);
const normalization = peak > 0 ? targetPeak / peak : 1;

const bytesPerSample = 2;
const channels = 2;
const dataSize = frames * channels * bytesPerSample;
const wav = Buffer.alloc(44 + dataSize);
wav.write("RIFF", 0);
wav.writeUInt32LE(36 + dataSize, 4);
wav.write("WAVE", 8);
wav.write("fmt ", 12);
wav.writeUInt32LE(16, 16);
wav.writeUInt16LE(1, 20);
wav.writeUInt16LE(channels, 22);
wav.writeUInt32LE(sampleRate, 24);
wav.writeUInt32LE(sampleRate * channels * bytesPerSample, 28);
wav.writeUInt16LE(channels * bytesPerSample, 32);
wav.writeUInt16LE(bytesPerSample * 8, 34);
wav.write("data", 36);
wav.writeUInt32LE(dataSize, 40);

for (let index = 0; index < frames; index += 1) {
  const offset = 44 + index * channels * bytesPerSample;
  const leftSample = clamp(left[index] * normalization, -1, 1);
  const rightSample = clamp(right[index] * normalization, -1, 1);
  wav.writeInt16LE(Math.round(leftSample * 32767), offset);
  wav.writeInt16LE(Math.round(rightSample * 32767), offset + 2);
}

mkdirSync(dirname(output), {recursive: true});
writeFileSync(output, wav);
console.log(output);
