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
