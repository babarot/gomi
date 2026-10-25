// What is in the trash when the page opens: made-up files, deleted over the
// last six weeks, some by one rm together (run), some with a mark. A
// directory's ls is what its preview lists, as ls -F does.
const TEXT = {
  go: `package ui

import (
	"testing"
	"time"
)

// probe how long each startup phase takes
func TestProbe(t *testing.T) {
	start := time.Now()
	files, err := listTrash()
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range files {
		if i > 50 {
			break
		}
		t.Logf("%s %d", f.Name, f.Size)
	}
	t.Logf("took %v", time.Since(start))
}`,
  log: `22:07:31 INFO  initialize xdg storage
22:07:31 DEBUG legacy storage historyPath=~/.gomi/history.json
22:07:31 INFO  list files storage_type=xdg len(files)=34
22:07:31 DEBUG after period filtering len(items)=34
22:07:32 WARN  slow preview path=~/Downloads/gomi_1.8.0_darwin_arm64 took=212ms
22:07:33 INFO  restore name=config.yaml to=~/.config/gomi/config.yaml`,
  build: `> site@0.4.2 build
> vite build

vite v6.3.1 building for production...
✓ 412 modules transformed.
dist/index.html                  1.12 kB │ gzip:  0.58 kB
dist/assets/index-3f9a.css      18.40 kB │ gzip:  4.31 kB
dist/assets/index-3f9a.js      211.07 kB │ gzip: 68.12 kB
✓ built in 2.41s`,
  coverage: `mode: set
github.com/babarot/gomi/internal/trash/xdg/xdg.go:41.52,43.16 2 1
github.com/babarot/gomi/internal/trash/xdg/xdg.go:43.16,45.3 1 0
github.com/babarot/gomi/internal/trash/xdg/xdg.go:46.2,47.12 2 1
github.com/babarot/gomi/internal/ui/view_list.go:36.32,39.2 3 1`,
  inbox: `# Inbox

- read the XDG trash spec again for the directorysizes cache
- reply about the Windows build
- buy coffee beans

## Someday

- a gomi sticker`,
  daily: `# 2026-10-23

Fixed the size filter for directories. It reads directorysizes now,
so a big folder no longer slows down the list.`,
  scratch: `curl -fsSL https://gomi.dev/install | PREFIX=~/.local/bin bash
gomi --prune=30d --dry-run=json | jq '.[].original'`,
  env: `DATABASE_URL=postgres://localhost:5432/api_dev
REDIS_URL=redis://localhost:6379/0
STRIPE_KEY=sk_test_xxxxxxxxxxxxxxxx
LOG_LEVEL=debug`,
  compose: `services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      LOG_LEVEL: debug
  db:
    image: postgres:16
    volumes:
      - ./tmp/pg:/var/lib/postgresql/data`,
  sql: `-- dump of api_dev, 2026-10-21
BEGIN;
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL
);
INSERT INTO users VALUES (1, 'a@example.com', '2026-09-01 10:00');
INSERT INTO users VALUES (2, 'b@example.com', '2026-09-03 18:22');
COMMIT;`,
  zshrc: `# ~/.zshrc
export EDITOR=nvim
export PATH="$HOME/.local/bin:$PATH"

alias rm=gomi
alias ll='ls -lah'

source ~/.zsh/prompt.zsh`,
  csv: `region,q1,q2,q3
north,128,141,152
south,96,104,99
east,143,150,171
west,88,93,101`,
  lock: `{
  "name": "site",
  "version": "0.4.2",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {
    "node_modules/vite": {
      "version": "6.3.1",
      "dev": true
    }
  }
}`,
  py: `import os
from fastapi import FastAPI

app = FastAPI()


@app.get("/health")
def health():
    return {"ok": True, "env": os.getenv("APP_ENV", "dev")}`,
  req: `fastapi==0.115.0
uvicorn==0.30.6
psycopg[binary]==3.2.1`,
  q3: `# 2026 Q3

## Done

- gomi 1.7: the recent view and the right preview
- moved the blog to a new host

## Not done

- the Windows installer`,
  config: `core:
  trash:
    strategy: auto
    home_fallback: true
  restore:
    confirm: false
    verbose: true
ui:
  view: recent
  theme: gomi
  preview:
    colorscheme: nord`,
  header: `import { Logo } from "./Logo";

export function Header({ title }: { title: string }) {
  return (
    <header className="top">
      <Logo />
      <h1>{title}</h1>
    </header>
  );
}`,
  footer: `export function Footer() {
  const year = new Date().getFullYear();
  return <footer>© {year} site</footer>;
}`,
  css: `/* the layout of the top bar */
.top {
  display: flex;
  gap: 12px;
  padding: 16px;
}
.top h1 {
  font-size: 20px;
  margin: 0;
}`,
  todo: `- [x] alias rm=gomi
- [ ] empty the trash once a month
- [ ] write about gomi`,
  trace: `trace 2026-09-14T10:02:11Z start
trace 2026-09-14T10:02:11Z open /tmp/trace.out
trace 2026-09-14T10:02:14Z gc 12 @3.012s 2%: 0.041+1.2+0.012 ms clock
trace 2026-09-14T10:02:19Z stop`,
  notes: `meeting notes, Oct 25

- ship the new site after the font decision
- the terminal demo should match the real TUI`,
  maingo: `package main

import "fmt"

func main() {
	fmt.Println("hello, gomi")
}`,
};

// m is minutes before now
export const SEED = [
  { m: 25,    dir: '~/projects/gomi/internal/ui', name: 'zz_probe_test.go', size: 794, run: 'r1', text: TEXT.go },
  { m: 25,    dir: '~/projects/gomi/internal/ui', name: 'zz_probe2_test.go', size: 806, run: 'r1', text: TEXT.go.replace('TestProbe', 'TestProbe2') },
  { m: 70,    dir: '~/projects/site', name: 'dist', isDir: true, ls: ['assets/', 'favicon.svg', 'index.html', 'robots.txt'], size: 421000, run: 'r2' },
  { m: 70,    dir: '~/projects/site', name: 'build.log', size: 3400, run: 'r2', text: TEXT.build },
  { m: 190,   dir: '~/Downloads', name: 'invoice-2026-10.pdf', size: 186000, run: 'r3' },
  { m: 300,   dir: '~/projects/gomi', name: 'coverage.out', size: 12300, run: 'r4', text: TEXT.coverage },
  { m: 1500,  dir: '~/notes', name: 'inbox.md', size: 1900, run: 'r5', origin: 'taken', text: TEXT.inbox },
  { m: 1500,  dir: '~/notes', name: '2026-10-23.md', size: 820, run: 'r5', text: TEXT.daily },
  { m: 1500,  dir: '~/notes', name: 'scratch.txt', size: 140, run: 'r5', text: TEXT.scratch },
  { m: 1650,  dir: '~/Desktop', name: 'Screenshot 2026-10-24 at 09.12.png', size: 384000, run: 'r6', img: 'ogimage/can.png' },
  { m: 1650,  dir: '~/Desktop', name: 'Screenshot 2026-10-24 at 09.15.png', size: 401000, run: 'r6', img: 'favicon/web-app-manifest-512x512.png' },
  { m: 2900,  dir: '~/projects/api', name: '.env.local', size: 210, run: 'r7', origin: 'taken', text: TEXT.env },
  { m: 2960,  dir: '~/projects/api', name: 'docker-compose.override.yml', size: 640, run: 'r8', text: TEXT.compose },
  { m: 3100,  dir: '~/projects/api/tmp', name: 'dump.sql', size: 2400000, run: 'r9', origin: 'gone', text: TEXT.sql },
  { m: 4400,  dir: '~/src/github.com/you/dotfiles', name: '.zshrc.bak', size: 4100, run: 'r10', origin: 'taken', text: TEXT.zshrc, lang: 'sh' },
  { m: 4410,  dir: '~/src/github.com/you/dotfiles', name: 'nvim', isDir: true, ls: ['after/', 'init.lua', 'lazy-lock.json', 'lua/'], size: 98000, run: 'r11' },
  { m: 5800,  dir: '~/Documents/old-report', name: 'report-v1.docx', size: 48200, run: 'r12', origin: 'gone' },
  { m: 5800,  dir: '~/Documents/old-report', name: 'figures', isDir: true, ls: ['fig1-latency.png', 'fig2-throughput.png', 'fig3-errors.png', 'raw/'], size: 2300000, run: 'r12', origin: 'gone' },
  { m: 5800,  dir: '~/Documents/old-report', name: 'data.csv', size: 9800, run: 'r12', origin: 'gone', text: TEXT.csv },
  { m: 7300,  dir: '~/projects/gomi/docs/tapes', name: 'demo-main.gif', size: 1250000, run: 'r13', img: 'demo/main.gif' },
  { m: 7300,  dir: '~/projects/gomi/docs/tapes', name: 'demo-prune.gif', size: 880000, run: 'r13', img: 'demo/prune.gif' },
  { m: 7300,  dir: '~/projects/gomi/docs/tapes', name: 'demo-debug.gif', size: 640000, run: 'r13', img: 'demo/debug.gif' },
  { m: 8700,  dir: '/tmp', name: 'go-build4123', isDir: true, ls: ['b001/', 'b002/', 'b003/', 'b004/', 'b005/', 'importcfg', 'importcfg.link'], size: 18800000, run: 'r14' },
  { m: 8800,  dir: '~/projects/site', name: 'package-lock.json', size: 410000, run: 'r15', text: TEXT.lock },
  { m: 14500, dir: '~/Downloads', name: 'gomi_1.8.0_darwin_arm64.tar.gz', size: 3900000, run: 'r16' },
  { m: 14500, dir: '~/Downloads', name: 'gomi_1.8.0_darwin_arm64', isDir: true, ls: ['CHANGELOG.md', 'LICENSE', 'README.md', 'gomi*'], size: 9100000, run: 'r16' },
  { m: 17300, dir: '~/projects/api', name: 'main.py', size: 3300, run: 'r17', text: TEXT.py },
  { m: 17300, dir: '~/projects/api', name: 'requirements.txt', size: 120, run: 'r17', text: TEXT.req },
  { m: 21600, dir: '~/notes/archive', name: '2026-q3.md', size: 12000, run: 'r18', text: TEXT.q3 },
  { m: 27400, dir: '~/.config/gomi', name: 'config.yaml.orig', size: 4600, run: 'r19', text: TEXT.config, lang: 'yaml' },
  { m: 31700, dir: '~/projects/site/src', name: 'Header.tsx', size: 2400, run: 'r20', text: TEXT.header },
  { m: 31700, dir: '~/projects/site/src', name: 'Footer.tsx', size: 1100, run: 'r20', text: TEXT.footer },
  { m: 31700, dir: '~/projects/site/src', name: 'styles.css', size: 5600, run: 'r20', text: TEXT.css },
  { m: 43200, dir: '~/Desktop', name: 'todo.txt', size: 300, run: 'r21', origin: 'taken', text: TEXT.todo },
  { m: 57600, dir: '/tmp', name: 'trace.log', size: 88000, run: 'r22', text: TEXT.trace },
];

// the files on the desk in the hero, in ~/work
export const DESK_FILES = [
  { name: 'notes.txt', size: 1230, text: TEXT.notes },
  { name: 'main.go', size: 72, text: TEXT.maingo },
  { name: '.env', size: 210, text: TEXT.env },
  { name: 'build', isDir: true, ls: ['app*', 'app.dSYM/', 'obj/', 'version.txt'], size: 421000 },
  { name: 'photo.png', size: 2300000, img: 'favicon/web-app-manifest-512x512.png' },
  { name: 'thesis_final_v3.docx', size: 48200 },
];
