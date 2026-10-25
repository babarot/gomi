// The rows of the list and the lines around it (view_list.go).
import { env } from './env.js';
import { ELL, sw, truncate, spaces, padR, padL, shortPath, width, cutSegs, fg } from './text.js';
import { hhmm, janD, dayLabel, shortAgo, compactBytes, plural } from './time.js';
import { query, fuzzy, shown } from './list.js';

// onRow gives a style the cursor row's background when the row is under it
const onRow = cur => st => (cur ? Object.assign({}, st, { bg: env.P.CursorRow }) : st);

export const sizeOf = files => compactBytes(files.reduce((a, f) => a + f.size, 0));

// rowStart is the cursor bar and the selection mark every row starts with
function rowStart(cur, mark) {
  const P = env.P, on = onRow(cur);
  return [[cur ? '▎' : ' ', on(fg(P.Cursor))], [' ', on({})], [mark || ' ', on({ fg: P.Selected, b: true })], [' ', on({})]];
}

// fillRow pads a row to w columns on the row's background and cuts it to w
function fillRow(segs, cur, w) {
  segs.push([spaces(w - width(segs)), onRow(cur)({})]);
  return cutSegs(segs, w);
}

// nameCell is the name cut to w, a directory marked with a slash, and what
// matches the filter drawn apart
export function nameCell(m, f, w, on) {
  const P = env.P;
  const name = truncate(f.name + (f.isDir ? '/' : ''), w, ELL);
  const base = on({ fg: f.isDir ? P.DirName : P.Name, b: true });
  const hit = on({ fg: P.FilterMatch, b: true, u: true });
  const q = query(m);
  if (m.view === 'picker') {
    const pos = new Set(q ? fuzzy(name, q) || [] : []);
    return [...name].map((c, i) => [c, pos.has(i) ? hit : base]);
  }
  const i = q ? name.toLowerCase().indexOf(q) : -1;
  if (i < 0) return [[name, base]];
  const r = [...name];
  return [[r.slice(0, i).join(''), base], [r.slice(i, i + q.length).join(''), hit], [r.slice(i + q.length).join(''), base]];
}

// originMark says what restoring f meets: ! its name is taken, + its folder is gone
function originMark(f, on) {
  if (f.origin === 'taken') return [['!', on({ fg: env.P.Dialog, b: true })]];
  if (f.origin === 'gone') return [['+', on(fg(env.P.Description))]];
  return [[' ', on({})]];
}

// dayLine is a rule with the day and what was deleted on it
function dayLine(r, w) {
  const P = env.P;
  const right = r.files.length + ' ' + plural(r.files.length, 'file') + ' · ' + sizeOf(r.files);
  const fill = w - 3 - r.day.length - 2 - right.length - 3;
  const rule = fg(P.HelpSeparator);
  if (fill < 1) return [['── ', rule], [r.day, { fg: P.Heading, b: true }]];
  return [['── ', rule], [r.day, { fg: P.Heading, b: true }], [' ', {}], ['─'.repeat(fill), rule], [' ', {}], [right, fg(P.Description)], [' ', {}], ['──', rule]];
}

// fileLine is a file in recent: when, the line joining one rm's files, the
// name, where from and the size
function fileLine(m, r, cur, w) {
  const P = env.P, f = r.file, on = onRow(cur), plain = on({});
  const segs = rowStart(cur, m.sel.has(f.id) ? '✓' : '');
  const clock = r.first ? hhmm(f.at) : '';
  segs.push([clock, on(fg(P.Description))], [spaces(6 - clock.length), plain]);
  const join = !r.batch ? ' ' : r.first ? '┌' : r.last ? '└' : '│';
  segs.push([join, on(fg(P.HelpSeparator))], [' ', plain]);
  const avail = Math.max(0, w - 12 - 2 - 6 - 3);
  const nameW = Math.max(10, Math.floor(avail * 45 / 100)), placeW = Math.max(0, avail - nameW);
  const name = nameCell(m, f, nameW, on);
  segs.push(...name, [spaces(nameW - width(name)), plain], ['  ', plain]);
  const place = shortPath(f.dir, placeW);
  segs.push([place, on(fg(P.Description))], [spaces(placeW - sw(place)), plain]);
  const size = compactBytes(f.size);
  segs.push([spaces(6 - size.length), plain], [size, on(fg(P.Description))], [' ', plain], ...originMark(f, on), [' ', plain]);
  return cutSegs(segs, w);
}

// pickLine is a file in the picker
function pickLine(m, r, cur, w) {
  const P = env.P, f = r.file, on = onRow(cur), plain = on({});
  const right = padL(shortAgo(f.at), 7) + '  ' + padL(compactBytes(f.size), 6);
  const segs = rowStart(cur, m.sel.has(f.id) ? '✓' : '');
  const room = Math.max(0, w - 3 - width(segs) - right.length - 2);
  const nameW = Math.max(10, Math.floor(room * 45 / 100));
  const name = nameCell(m, f, nameW, on);
  segs.push(...name, [spaces(Math.max(2, nameW + 2 - width(name))), plain]);
  segs.push([shortPath(f.dir, Math.max(0, room - nameW - 2)), on(fg(P.Description))]);
  segs.push([spaces(Math.max(1, w - 3 - width(segs) - right.length)), plain], [right, on(fg(P.Description))], [' ', plain], ...originMark(f, on));
  return fillRow(segs, cur, w);
}

// foldLine is a directory in places or an rm in batches
function foldLine(m, r, cur, w) {
  const P = env.P, on = onRow(cur), plain = on({});
  const nsel = r.files.filter(f => m.sel.has(f.id)).length;
  const mark = nsel && nsel === r.files.length ? '✓' : nsel ? '•' : '';
  let count = r.files.length + ' ' + plural(r.files.length, 'file');
  if (r.files.length < r.total) count = r.files.length + ' of ' + r.total + ' files';
  let right = count + ' · ' + sizeOf(r.files);
  if (m.view === 'places') right += '  ' + padL(shortAgo(r.files[0].at), 7);
  const room = Math.max(0, w - 6 - right.length - 2);
  const segs = rowStart(cur, mark);
  segs.push([r.open ? '▾' : '▸', on(fg(P.Cursor))], [' ', plain]);
  if (m.view === 'places') segs.push([shortPath(r.key, room), on({ fg: P.Name, b: true })]);
  else segs.push(...runLabel(r, room, on));
  segs.push([spaces(Math.max(1, w - 1 - width(segs) - right.length)), plain], [right, on(fg(P.Description))]);
  return fillRow(segs, cur, w);
}

// runLabel is an rm in batches: when, and "rm" with what it removed and from where
function runLabel(r, room, on) {
  const P = env.P, t = r.files[0].at;
  const when = dayLabel(t) !== 'Today' ? janD(t) + ' ' + hhmm(t) : hhmm(t);
  const places = new Set(r.files.map(f => f.dir));
  const where = places.size === 1 ? r.files[0].dir : places.size + ' places';
  const head = padR(when, 12) + ' ';
  const rest = Math.max(0, room - head.length - 3);
  const whereW = Math.min(sw(where), Math.floor(rest / 3));
  const nameW = Math.max(0, rest - whereW - 2);
  const names = r.files.map(f => f.name + (f.isDir ? '/' : '')).join(' ');
  const segs = [[head, on(fg(P.Description))], ['rm ', on(fg(P.HelpSeparator))], [truncate(names, nameW, ELL), on({ fg: P.Name, b: true })]];
  if (whereW > 0) segs.push(['  ', on({})], [shortPath(where, whereW), on(fg(P.Description))]);
  return segs;
}

// foldFileLine is a file in an open fold
function foldFileLine(m, r, cur, w) {
  const P = env.P, f = r.file, on = onRow(cur), plain = on({});
  const size = compactBytes(f.size);
  const right = m.view === 'batches' ? padL(size, 6) : padL(shortAgo(f.at), 7) + '  ' + padL(size, 6);
  const segs = rowStart(cur, m.sel.has(f.id) ? '✓' : '');
  segs.push(['  ', plain], [r.last ? '└' : '├', on(fg(P.HelpSeparator))], [' ', plain]);
  const room = Math.max(0, w - 3 - width(segs) - right.length - 2);
  if (m.view === 'batches') {
    const nameW = Math.floor(room / 2);
    const name = nameCell(m, f, nameW, on);
    segs.push(...name, [spaces(Math.max(2, nameW + 2 - width(name))), plain], [shortPath(f.dir, Math.max(0, room - nameW - 2)), on(fg(P.Description))]);
  } else segs.push(...nameCell(m, f, room, on));
  segs.push([spaces(Math.max(1, w - 3 - width(segs) - right.length)), plain], [right, on(fg(P.Description))], [' ', plain], ...originMark(f, on));
  return fillRow(segs, cur, w);
}

// rowLine draws a day heading, a fold or a file, w columns wide
export function rowLine(m, r, cur, w) {
  if (r.kind === 'day') return dayLine(r, w);
  if (r.kind === 'fold') return foldLine(m, r, cur, w);
  if (r.depth > 0) return foldFileLine(m, r, cur, w);
  if (m.view === 'picker') return pickLine(m, r, cur, w);
  return fileLine(m, r, cur, w);
}

// headerLine is what is in the trash on the left, and the scope, the
// selection and the view on the right
export function headerLine(m, W) {
  const P = env.P;
  const left = [['gomi', { fg: P.Cursor, b: true }], ['  ' + m.files.length + ' in the trash · ' + sizeOf(m.files), fg(P.Description)]];
  let right = [['view: ' + m.view, fg(P.Description)]];
  if (m.scoped) right = [['in ' + shortPath(env.cwd, Math.max(8, Math.floor(W / 3))), fg(P.FilterPrompt)], [' · ', fg(P.Description)], ...right];
  if (m.sel.size) right = [[m.sel.size + ' selected', fg(P.Selected)], [' · ', fg(P.Description)], ...right];
  const gap = W - 2 - width(left) - width(right);
  if (gap < 1) return cutSegs([[' ', {}], ...left], W);
  return [[' ', {}], ...left, [spaces(gap), {}], ...right];
}

// filterLine is the filter, with its cursor while it has the focus
export function filterLine(m) {
  const segs = [[' ', {}], [m.view === 'picker' ? '❯ ' : '/ ', { fg: env.P.FilterPrompt, b: true }], [m.filter, {}]];
  if (m.filtering) segs.push([' ', { r: true }]);
  return segs;
}

export function emptyText(m) {
  if (m.filter) return 'No files match "' + m.filter + '"';
  return m.scoped ? 'Nothing was deleted here; . shows every file' : 'The trash is empty';
}

// countLine is how many files are shown, how many more are below, and
// what the marks mean when there are any
export function countLine(m, rs, n, W) {
  const files = shown(m);
  let s = files.length + ' ' + plural(files.length, 'file');
  if (m.filter !== '') s = files.length + ' of ' + m.files.length + ' files';
  const below = rs.slice(Math.min(rs.length, m.top + n)).filter(r => r.kind === 'file').length;
  if (below > 0) s += ' · ↓ ' + below + ' more';
  if (files.some(f => f.origin)) s += '   ! a file has its name now   + its folder is gone';
  return [['  ' + truncate(s, W - 2, ELL), fg(env.P.Description)]];
}

// helpLine is bubbles' help.ShortHelpView at the width the UI gives it
export function helpLine(m, W) {
  const P = env.P;
  const list = [['↑/k', 'up'], ['↓/j', 'down'], ['enter', 'restore'], ['space', 'detail'], ['tab', 'select'], ['/', 'filter'], ['v', 'view'], ['?', 'more']];
  const detail = [['space', 'back'], ['n', 'next'], ['p', 'prev'], ['?', 'more']];
  const max = W - 2; let total = 0; const segs = [['  ', {}]];
  for (const [k, d] of m.detail ? detail : list) {
    const sep = total > 0 ? ' • ' : '';
    const w = sep.length + k.length + 1 + d.length;
    if (total + w > max) { if (total + 2 < max) segs.push([' …', fg(P.HelpEllipsis)]); break; }
    if (sep) segs.push([sep, fg(P.HelpSeparator)]);
    segs.push([k, fg(P.HelpKey)], [' ' + d, fg(P.HelpDesc)]);
    total += w;
  }
  return segs;
}
