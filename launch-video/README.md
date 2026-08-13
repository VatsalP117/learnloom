# Learnloom launch film

> **Not approved for publication.** Existing renders and source scenes predate
> the AI/software launch wedge, topic-first discovery, private/draft/published
> semantics, and current product UI. They include legacy Maya/urban examples.
> Regenerate from reviewed starter lessons and current captures before use.

A 46-second, 1920×1080 launch film inspired by the reference video's editorial
typography, restrained product motion, white-space-led composition, and dark
immersive interludes.

The film uses original Learnloom visuals and an original generated ambient
soundtrack. It does not reuse footage, branding, or audio from the reference.

## Story

1. Introduce Learnloom as a learning home.
2. Contrast endless reading with durable understanding.
3. Start with a technical question and inspect Learnloom's explainable source
   portfolio (or supply specific sources).
4. Weave those sources into a Learning Blueprint.
5. Reveal the finished Dossier.
6. Show continuity through Learning History.
7. Keep it private by default; optionally deliver by email or publish later.
8. Close with the product promise and launch CTA.

## Render

Requires Node.js, `sips` (included with macOS), and FFmpeg:

```sh
node launch-video/render.mjs
```

The renderer creates the legacy storyboard only. Its output is not launch
evidence until the scenes and captures are replaced and reviewed.

## Remotion cut

The editable launch-film composition now lives in `remotion/` and is the
preferred path for the revised cut:

```sh
cd launch-video/remotion
npm install
npm run studio
npm run render
npm run render:v2
```

It exports a 45-second `output/learnloom-launch-remotion.mp4`. The current cut
uses a continuous animated product workspace, with the captured product states
in `remotion/public/captures/` retained as visual references. Every scene stays
within the same white, warm-paper, black-type visual language.

The V1.1 audio experiment keeps the original film and visuals intact while
using a separate original electronic score with a faster 128 BPM pulse. It
exports to `output/learnloom-launch-remotion-v1.1.mp4`:

```sh
cd launch-video/remotion
npm run music:v1.1
npm run render:v1.1
```

The separate `LearnloomLaunchV2` composition is a 27-second cut with compressed
typing, a nearly double-speed continuous workspace, and a kinetic closing
statement. It exports `output/learnloom-launch-v2.mp4`.
