// The list of files the TUI shows, laid out by the view and narrowed by the
// filter (filelist.go). A file is { id, name, dir, at, size, isDir, run,
// origin }: dir is where it was deleted from, written with ~, run the rm
// that deleted it, and origin what restoring it meets: '', 'taken' or 'gone'.
import { env } from './env.js';
import { dayLabel } from './time.js';

export const VIEWS = ['recent', 'places', 'batches', 'picker'];
export const SORTS = [
  ['Deleted, newest first', (a, b) => b.at - a.at],
  ['Deleted, oldest first', (a, b) => a.at - b.at],
  ['Name', (a, b) => (a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1)],
  ['Size, largest first', (a, b) => b.size - a.size],
  ['Deleted from', (a, b) => (a.dir < b.dir ? -1 : a.dir > b.dir ? 1 : 0)],
];

// newList is the TUI's state over files, the trash newest first. scope is
// ui.scope (auto, here or all) and view ui.view.
export function newList(files, { scope = 'auto', view = 'recent' } = {}) {
  const m = {
    files, view: 'recent', sort: 0, open: {}, seen: {},
    scoped: scope === 'here' || (scope === 'auto' && files.some(here)),
    filter: '', filtering: false, cursor: 0, top: 0, sel: new Set(),
    detail: false, overlay: null, sortSel: 0, vy: 0, previewed: new Set(), status: '',
  };
  if (view !== 'recent') setView(m, view);
  fixCursor(m, rows(m));
  return m;
}

// here reports whether f was deleted where gomi runs, or under it
export const here = f => f.dir === env.cwd || f.dir.startsWith(env.cwd + '/');
// fullPath is where f goes back to, as the restore prints it
export const fullPath = f => (f.dir + '/' + f.name).replace(/^~/, env.home);

// fuzzy finds q's letters in s in order, and where they are
export function fuzzy(s, q) {
  const pos = []; let j = 0; const rs = [...s];
  for (let i = 0; i < rs.length && j < q.length; i++) if (rs[i].toLowerCase() === q[j]) { pos.push(i); j++; }
  return j === q.length ? pos : null;
}
function pickScore(f, q) {
  let p = fuzzy(f.name, q); if (p) return p[p.length - 1] - p[0] - p.length + 1;
  p = fuzzy(fullPath(f), q); if (p) return 1000 + p[p.length - 1] - p[0];
  return null;
}

export const query = m => m.filter.trim().toLowerCase();
function matches(m, f) {
  if (m.scoped && !here(f)) return false;
  const q = query(m); if (!q) return true;
  if (m.view === 'picker') return pickScore(f, q) !== null;
  return f.name.toLowerCase().includes(q) || f.dir.toLowerCase().includes(q);
}
// shown is the files the scope and the filter keep
export const shown = m => m.files.filter(f => matches(m, f));

// rows lays out the files by the view
export function rows(m) {
  if (m.view === 'places' || m.view === 'batches') return foldRows(m);
  if (m.view === 'picker') return pickRows(m);
  return dayRows(m);
}

// sameRun reports whether one rm deleted a and b; a file with no run is
// a run of its own
const sameRun = (a, b) => !!a.run && a.run === b.run;

function dayRows(m) {
  const files = shown(m), out = []; let day = '', head = null;
  files.forEach((f, i) => {
    const d = dayLabel(f.at);
    if (d !== day) { head = { kind: 'day', day: d, files: [] }; out.push(head); day = d; }
    head.files.push(f);
    const first = i === 0 || !sameRun(files[i - 1], f) || dayLabel(files[i - 1].at) !== d;
    const last = i === files.length - 1 || !sameRun(files[i + 1], f) || dayLabel(files[i + 1].at) !== d;
    out.push({ kind: 'file', file: f, first, last, batch: !first || !last });
  });
  return out;
}

// foldOf is the key of the fold f is in
export const foldOf = (m, f) => (m.view === 'places' ? f.dir : f.run || 'file:' + f.id);

function foldRows(m) {
  const folds = [], by = {};
  for (const f of m.files) {
    const k = foldOf(m, f);
    if (!by[k]) { by[k] = { key: k, total: 0, files: [] }; folds.push(by[k]); }
    by[k].total++;
    if (matches(m, f)) by[k].files.push(f);
  }
  const out = [];
  for (const g of folds) {
    if (!g.files.length) continue;
    const open = !!m.open[m.view + ':' + g.key] || query(m) !== '';
    out.push({ kind: 'fold', key: g.key, files: g.files, total: g.total, open });
    if (open) g.files.forEach((f, i) => out.push({ kind: 'file', file: f, depth: 1, last: i === g.files.length - 1 }));
  }
  return out;
}

function pickRows(m) {
  const q = query(m); let files = shown(m);
  if (!q) files = files.slice().sort(SORTS[m.sort][1]);
  else files = files.slice().sort((a, b) => pickScore(a, q) - pickScore(b, q));
  return files.map(f => ({ kind: 'file', file: f }));
}

export const keyOf = r => (r.kind === 'file' ? r.file.id : r.kind === 'fold' ? 'fold:' + r.key : '');

// fixCursor keeps the cursor on a row it can be on: never a day heading
export function fixCursor(m, rs) {
  if (!rs.length) { m.cursor = 0; return; }
  m.cursor = Math.max(0, Math.min(m.cursor, rs.length - 1));
  let i = m.cursor; while (i < rs.length && rs[i].kind === 'day') i++;
  if (i >= rs.length) { i = m.cursor; while (i > 0 && rs[i].kind === 'day') i--; }
  m.cursor = i;
}

// find puts the cursor on the row of key, or on the first row
export function find(m, key) {
  const rs = rows(m);
  m.cursor = 0;
  const i = rs.findIndex(r => key !== '' && keyOf(r) === key);
  if (i >= 0) m.cursor = i;
  fixCursor(m, rs);
}

export const currentRow = m => { const rs = rows(m); fixCursor(m, rs); return rs[m.cursor]; };
export const currentFile = m => { const r = currentRow(m); return r && r.kind === 'file' ? r.file : null; };

// setView lays the files out by view, keeping the cursor on its file, or
// on the fold the file is in. A view shown for the first time opens its
// first fold, and places the directories under where gomi runs.
export function setView(m, view) {
  const f = currentFile(m);
  m.view = view;
  m.filtering = view === 'picker';
  if (view === 'picker') m.filter = '';
  const rs = rows(m);
  if ((view === 'places' || view === 'batches') && rs.length && !m.seen[view]) {
    m.open[view + ':' + rs[0].key] = true;
    if (view === 'places') rs.forEach(r => { if (r.kind === 'fold' && here({ dir: r.key })) m.open[view + ':' + r.key] = true; });
    m.seen[view] = true;
  }
  if (!f) { m.cursor = 0; fixCursor(m, rows(m)); return; }
  find(m, f.id);
  const c = currentFile(m);
  if (!c || c.id !== f.id) find(m, 'fold:' + foldOf(m, f));
}

// move moves the cursor n rows, over day headings
export function move(m, n) {
  const rs = rows(m); let i = m.cursor;
  const step = n > 0 ? 1 : -1;
  for (let left = Math.abs(n); left > 0; left--) {
    let j = i + step;
    while (j >= 0 && j < rs.length && rs[j].kind === 'day') j += step;
    if (j < 0 || j >= rs.length) break;
    i = j;
  }
  m.cursor = i; m.vy = 0;
}

// moveFile moves the cursor to the next or the previous file
export function moveFile(m, n) {
  const rs = rows(m); let i = m.cursor;
  do { i += n; } while (i >= 0 && i < rs.length && rs[i].kind !== 'file');
  if (i >= 0 && i < rs.length) { m.cursor = i; m.vy = 0; }
}

// scroll keeps the cursor among the n rows shown, and the day above it
export function scroll(m, rs, n) {
  if (m.cursor < m.top) { m.top = m.cursor; if (m.top > 0 && rs[m.top - 1].kind === 'day') m.top--; }
  if (m.cursor >= m.top + n) m.top = m.cursor - n + 1;
  m.top = Math.max(0, Math.min(m.top, Math.max(0, rs.length - n)));
}

// select selects or unselects the file on the cursor, or a fold's files
export function select(m, on) {
  const r = currentRow(m); if (!r) return;
  (r.kind === 'fold' ? r.files : [r.file]).forEach(f => (on ? m.sel.add(f.id) : m.sel.delete(f.id)));
  if (on) move(m, 1);
}
