// site-golden checks that the terminal on gomi.dev (docs/tui) draws what
// gomi draws: it sets up the trash of internal/ui/golden_test.go, presses
// the keys of its cases, and compares the screen with the text of the
// golden files. Colors are not compared, since the site has its own.
//
// Usage: node scripts/site-golden.mjs [CASE ...]   (mise run site:golden)
process.env.TZ = 'UTC';

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { createTerminal } from '../docs/tui/terminal.js';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const goldenDir = join(root, 'internal/ui/testdata/golden');

// goldenNow and fixtures are those of golden_test.go
const goldenNow = Date.parse('2025-06-15T12:00:00Z');
const H = 3600 * 1000;
const longText = Array.from({ length: 40 }, (_, i) => 'line ' + String(i + 1).padStart(2, '0') + ': lorem ipsum dolor sit amet\n').join('');
const fixtures = [
  { name: 'main.go', ago: 2 * H, content: 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("hello, gomi")\n}\n' },
  { name: 'notes', ago: 5 * H, dir: '/home/user/docs', entries: { 'a.txt': 'first note\n', 'b.md': '# second note\n\nsome text\n', 'c.log': '2025-06-15 started\n2025-06-15 stopped\n', 'todo.md': '- [ ] restore files\n' } },
  { name: 'archive.bin', ago: 26 * H, dir: '/home/user/Downloads', content: '\x00\x01\x02\x03\xff\xfe\xfd\xfc\x00\x00\x10\x20', mime: 'application/octet-stream' },
  { name: 'long.txt', ago: 50 * H, content: longText },
  { name: 'README.md', ago: 74 * H, run: 'run-1', content: '# gomi\n\nReplacement for rm with rich trash.\n' },
  { name: 'config.yaml', ago: 74 * H + 30000, run: 'run-1', content: 'core:\n  trash:\n    strategy: auto\n' },
  { name: 'a-very-long-file-name-for-testing.txt', ago: 122 * H, content: 'a\n' },
  { name: 'another-quite-long-file-name.txt', ago: 146 * H, content: 'b\n' },
  { name: 'yet-another-long-file-name-here.txt', ago: 170 * H, content: 'c\n' },
  { name: 'todo.txt', ago: 240 * H, content: 'buy milk\n' },
  { name: 'report.csv', ago: 480 * H, dir: '/tmp', content: 'name,size\ngomi,1\n' },
  { name: 'old.log', ago: 960 * H, dir: '/tmp', content: 'old\n' },
];

const tilde = p => p.replace(/^\/home\/user(?=\/|$)/, '~');
const bytes = s => Buffer.byteLength(s, 'latin1');

// the builtin listing of a directory in the preview (file.go):
// mode, humanize.Bytes of the size, name
const listing = entries => Object.keys(entries).sort().map(n => '-rw-r--r-- ' + (bytes(entries[n]) + ' B').padStart(7) + '  ' + n);

function trashFiles() {
  return fixtures.map(f => {
    const dir = f.dir || '/home/user/projects/gomi';
    const isDir = !!f.entries;
    // what restoring meets, as TestGolden's checkOrigin has it
    const origin = f.name === 'config.yaml' ? 'taken' : dir.startsWith('/tmp') ? 'gone' : '';
    return {
      name: f.name, dir: tilde(dir), at: new Date(goldenNow - f.ago), run: f.run, isDir, origin,
      size: isDir ? Object.values(f.entries).reduce((a, c) => a + bytes(c), 0) : bytes(f.content),
      text: isDir ? undefined : f.content, mime: f.mime, listing: isDir ? listing(f.entries) : undefined,
    };
  });
}

// the cases of golden_test.go the site's terminal can show. The others set
// what the site does not have: ui.preview.position, ui.height, permanent
// deletion, ui.style, ui.theme, the prompts outside the TUI and a preview
// still being read.
const cases = [
  { name: 'list' },
  { name: 'list-wide', width: 120 },
  { name: 'list-wide-down', width: 120, keys: ['down'] },
  { name: 'list-preview-right', width: 100, keys: ['down', 'down', 'down'] },
  { name: 'view-places', view: 'places' },
  { name: 'view-places-close', view: 'places', keys: ['h', 'j', 'l'] },
  { name: 'view-batches', view: 'batches', keys: ['j', 'j', 'j', 'l', 'j'] },
  { name: 'view-switch', keys: ['j', 'v'] },
  { name: 'view-fold-select', view: 'places', keys: ['tab'] },
  { name: 'view-fold-preview', width: 120, view: 'batches', keys: ['j', 'j', 'j'] },
  { name: 'picker', view: 'picker' },
  { name: 'picker-typing', view: 'picker', keys: ['n', 't', 's'] },
  { name: 'picker-path', view: 'picker', keys: ['t', 'm', 'p'] },
  { name: 'picker-esc-view', view: 'picker', keys: ['r', 'e', 'esc', 'v'] },
  { name: 'picker-sort-menu', view: 'picker', keys: ['esc', 's', 'j', 'j'] },
  { name: 'picker-sorted', view: 'picker', keys: ['esc', 's', '3'] },
  { name: 'view-to-picker', keys: ['v', 'v', 'v'] },
  { name: 'scope-here', cwd: '/home/user/projects/gomi' },
  { name: 'scope-all', cwd: '/home/user/projects/gomi', keys: ['.'] },
  { name: 'scope-config-here', cwd: '/srv', scope: 'here' },
  { name: 'scope-config-all', cwd: '/home/user/projects/gomi', scope: 'all' },
  { name: 'scope-places', cwd: '/home/user/docs', view: 'places' },
  { name: 'scope-nothing-here', cwd: '/home/user/projects/gomi', keys: ['/', 'z', 'z', 'z'] },
  { name: 'help-overlay', keys: ['?'] },
  { name: 'help-overlay-detail', width: 120, keys: ['space', '?'] },
  { name: 'list-end', keys: ['G'] },
  { name: 'list-selected', keys: ['tab', 'tab'] },
  { name: 'list-selected-cursor', keys: ['tab', 'tab', 'shift+tab'] },
  { name: 'list-filter-empty', keys: ['/'] },
  { name: 'list-filter-typing', keys: ['/', 'r', 'e'] },
  { name: 'list-filter-applied', keys: ['/', 'r', 'e', 'enter'] },
  { name: 'list-help', keys: ['?'] },
  { name: 'detail-text', keys: ['space'] },
  { name: 'detail-dir', keys: ['down', 'space'] },
  { name: 'detail-binary', keys: ['down', 'down', 'space'] },
  { name: 'detail-scroll', keys: ['down', 'down', 'down', 'space', 'd'] },
  { name: 'detail-selected', keys: ['tab', 'up', 'space'] },
  { name: 'detail-help', keys: ['space', '?'] },
  { name: 'detail-taken', keys: ['down', 'down', 'down', 'down', 'down', 'space'] },
  { name: 'view-places-marks', width: 120, view: 'places', keys: ['G'] },
];

// key names of golden_test.go as the terminal takes them
const KEY = { down: 'ArrowDown', up: 'ArrowUp', right: 'ArrowRight', space: ' ', enter: 'Enter', esc: 'Escape', tab: 'Tab', 'shift+tab': 'BackTab', backspace: 'Backspace' };

// screen is what a case draws, as text
function screen(c) {
  const term = createTerminal({
    now: () => new Date(goldenNow), home: '/home/user', previewDelay: 0,
    cwd: c.cwd ? tilde(c.cwd) : '/nowhere', scope: c.scope || 'auto', view: c.view || 'recent',
  });
  trashFiles().forEach(f => term.add(f));
  // the TUI alone, without the shell above it
  term.size(c.width || 80, 30, 30);
  term.run();
  for (const k of c.keys || []) term.key(KEY[k] || k);
  return term.text();
}

// want is the screen of a golden file, as Grid.text writes it
function want(name) {
  const g = readFileSync(join(goldenDir, name + '.dark.golden'), 'utf8');
  const s = g.slice(g.indexOf('-- screen --\n') + 13, g.indexOf('-- styles --'));
  const rows = s.split('\n').map(l => l.replace(/\s+$/, ''));
  while (rows.length && rows[rows.length - 1] === '') rows.pop();
  return rows.join('\n');
}

const only = process.argv.slice(2);
let failed = 0;
for (const c of cases) {
  if (only.length && !only.includes(c.name)) continue;
  const got = screen(c), exp = want(c.name);
  if (got === exp) { console.log('ok   ' + c.name); continue; }
  failed++;
  console.log('FAIL ' + c.name);
  const g = got.split('\n'), w = exp.split('\n');
  for (let i = 0; i < Math.max(g.length, w.length); i++) {
    if (g[i] === w[i]) continue;
    console.log('  row ' + (i + 1));
    console.log('    want |' + (w[i] ?? '') + '|');
    console.log('    got  |' + (g[i] ?? '') + '|');
  }
}
console.log(failed ? failed + ' of ' + (only.length || cases.length) + ' cases differ' : 'all cases match');
process.exit(failed ? 1 : 0);
