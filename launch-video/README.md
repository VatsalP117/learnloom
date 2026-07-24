# Learnloom launch film

A 46-second, 1920×1080 launch film inspired by the reference video's editorial
typography, restrained product motion, white-space-led composition, and dark
immersive interludes.

The film uses original Learnloom visuals and an original generated ambient
soundtrack. It does not reuse footage, branding, or audio from the reference.

## Story

1. Introduce Learnloom as a learning home.
2. Contrast endless reading with durable understanding.
3. Choose a question and trusted sources.
4. Weave those sources into a Learning Blueprint.
5. Reveal the finished Dossier.
6. Show continuity through Learning History.
7. Deliver through a personal site and email.
8. Close with the product promise and launch CTA.

## Render

Requires Node.js, `sips` (included with macOS), and FFmpeg:

```sh
node launch-video/render.mjs
```

The renderer creates SVG motion frames, rasterizes them, synthesizes the
soundtrack, and exports `output/learnloom-launch-film.mp4`.

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

The separate `LearnloomLaunchV2` composition is a 27-second cut with compressed
typing, a nearly double-speed continuous workspace, and a kinetic closing
statement. It exports `output/learnloom-launch-v2.mp4`.
