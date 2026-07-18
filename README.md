# Learnloom

A local, source-grounded morning learning dossier that weaves fresh sources
with your learning history, powered by the DeepSeek credits in a
[Command Code](https://commandcode.ai/) subscription.

The engine retrieves configured RSS/Atom feeds, remembers recent lessons, and
uses four separate model passes:

1. **Researcher** chooses one coherent theme and extracts its important claims.
2. **Skeptic** challenges the evidence and identifies missing context.
3. **Teacher** produces a focused lesson at your level.
4. **Examiner** creates retrieval questions and an application exercise.

The result is saved to `output/YYYY-MM-DD.md`. Local learning history is kept
in `data/history.json`, so later lessons can build on earlier ones. Both
directories are ignored by Git.

## Why Command Code works here

Command Code officially supports a headless print mode:

```sh
cmd --print "prompt" --model deepseek-v4-pro
```

This project invokes that documented interface directly. It does not extract
your authentication token or use private endpoints. Each call uses plan
permissions, a one-turn limit, and an instruction not to use tools. See the
[official CLI reference](https://commandcode.ai/docs/reference/cli).

Command Code currently lists the full model ID as
`deepseek/deepseek-v4-pro`; the documented short-name form
`deepseek-v4-pro` is used in the example configuration.

## Requirements

- macOS, Linux, or Windows for manual runs
- Node.js 22 or newer
- macOS for automatic `launchd` scheduling
- Command Code subscription for live generation

There are no project runtime dependencies.

## Quick start

First verify the entire pipeline without network access or model credits:

```sh
npm test
npm run demo
```

Open the generated file under `output/`.

Then create your local configuration:

```sh
node bin/learn.mjs init
```

Edit `config.json` to set your interests, learner profile, and feeds. The file
is ignored by Git so personal source choices stay local.

Install and authenticate Command Code:

```sh
npm install -g command-code
cmd login
```

Confirm the complete setup:

```sh
npm run doctor
```

Run a live dossier:

```sh
npm start
```

## Daily 9:00 a.m. schedule

After one successful live run:

```sh
node bin/learn.mjs schedule install
node bin/learn.mjs schedule status
```

The launch agent uses the exact Node executable and project path present at
installation time. Logs are written under `data/logs/`.

To choose another local time:

```sh
node bin/learn.mjs schedule install --hour 8 --minute 30
```

To remove it:

```sh
node bin/learn.mjs schedule remove
```

The scheduled hour follows the Mac's current local time. `timeZone` in
`config.json` controls the date written in the dossier.

## Configuration

`config.example.json` contains all available settings:

```json
{
  "timeZone": "Asia/Kolkata",
  "interests": ["artificial intelligence research", "learning science"],
  "learner": {
    "level": "technically experienced",
    "goal": "build durable understanding",
    "lessonMinutes": 15
  },
  "sources": [
    {
      "name": "Hacker News",
      "url": "https://hnrss.org/frontpage",
      "limit": 12
    }
  ],
  "provider": {
    "kind": "commandcode",
    "executable": "cmd",
    "model": "deepseek-v4-pro",
    "timeoutSeconds": 600
  }
}
```

Useful source types include:

- Research category feeds such as arXiv
- Blogs with RSS or Atom feeds
- Hacker News or topic-specific news feeds
- A private feed you generate from your own bookmarks

Feed summaries are treated as untrusted data and are placed inside prompts as
reference material, not instructions. Important model claims should still be
verified against the linked source.

## Commands

```text
learn init [--config path] [--force]
learn run [--config path] [--demo]
learn doctor [--config path]
learn schedule install [--config path] [--hour 9] [--minute 0]
learn schedule status
learn schedule remove
```

You can invoke `learn` through `node bin/learn.mjs`, or install this repository
as a local command with `npm link`.

## Troubleshooting

### `Could not find "cmd"`

Install the official CLI and open a new terminal:

```sh
npm install -g command-code
```

If Node is managed by `nvm`, run schedule installation from the same shell in
which `cmd` works so its executable directory is captured in the launch agent.

### Command Code is installed but not authenticated

Run `cmd login`, complete browser approval, then run `npm run doctor` again.
Credentials remain in Command Code's own local storage and are never read by
this project.

### One feed fails

The run continues when at least one configured feed succeeds and prints a
warning for failed feeds. If all feeds fail, no model credits are spent.

### A model stage fails

The incomplete dossier is not saved. Re-run after fixing authentication,
connectivity, credit balance, or model availability.
