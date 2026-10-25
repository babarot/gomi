// What is drawn over the middle of the view: the keys ? shows and the
// picker's sort menu (overlay.go).
import { env } from './env.js';
import { padR, spaces, width, fg } from './text.js';
import { SORTS } from './list.js';

// boxed puts lines in a rounded frame of color, padX columns in from it
function boxed(body, color, padX = 1) {
  const inner = Math.max(...body.map(width));
  const B = fg(color), pad = spaces(padX);
  return [
    [['╭' + '─'.repeat(inner + padX * 2) + '╮', B]],
    ...body.map(l => [['│', B], [pad, {}], ...l, [spaces(inner - width(l)) + pad, {}], ['│', B]]),
    [['╰' + '─'.repeat(inner + padX * 2) + '╯', B]],
  ];
}

// helpBox is the keys that work, grouped, the group of the view open now
// first and marked, the groups in two columns
export function helpBox(m) {
  const P = env.P;
  const k = (key, d) => ({ key, d });
  const folds = m.view === 'places' || m.view === 'batches';
  const move = { name: 'Move', b: [k('↑/k', 'up'), k('↓/j', 'down'), k('g/home', 'go to start'), k('G/end', 'go to end')] };
  if (folds) move.b.push(k('l/→', 'open'), k('h/←', 'close'));
  else move.b.push(k('→/l/pgdn', 'next page'), k('←/h/pgup', 'prev page'));
  const files = { name: 'Files', b: [k('enter', 'restore'), k('space', 'detail'), k('tab', 'select'), k('s+tab', 'de-select')] };
  const views = { name: 'Views', b: [k('v', 'view'), k('/', 'filter'), k('.', 'here / all')] };
  if (m.view === 'picker') views.b.push(k('s', 'sort (picker)'));
  const preview = { name: 'Preview', b: [k('↑/k', 'preview up'), k('↓/j', 'preview down'), k('u', '½ page up'), k('d', '½ page down'), k('n', 'next'), k('p', 'prev'), k('space', 'back')] };
  const other = { name: 'Other', b: [k('esc', 'reset'), k('?', 'more'), k('q', 'quit')] };
  const groups = m.detail ? [preview, files, move, views, other] : [move, files, views, preview, other];
  const blocks = groups.map((g, i) => {
    const head = [[g.name, { fg: P.Heading, b: true }]];
    if (i === 0) head.push([' ', {}], [' here ', { fg: P.CursorRow, bg: P.Cursor, b: true }]);
    return [head, ...g.b.map(b => [['  ', {}], [padR(b.key, 11), { fg: P.Cursor, b: true }], [b.d, fg(P.Name)]])];
  });
  // each group goes to the shorter column, in order
  const cols = [[], []], hs = [0, 0];
  for (const b of blocks) {
    const c = hs[1] < hs[0] ? 1 : 0;
    if (cols[c].length) { cols[c].push([]); hs[c]++; }
    cols[c].push(...b); hs[c] += b.length;
  }
  const w0 = Math.max(...cols[0].map(width)) + 4;
  const body = [];
  for (let i = 0; i < Math.max(cols[0].length, cols[1].length); i++) {
    const a = cols[0][i] || [];
    body.push([...a, [spaces(w0 - width(a)), {}], ...(cols[1][i] || [])]);
  }
  body.push([], [['? esc q close', fg(P.Description)]]);
  return boxed(body, P.Cursor);
}

// sortBox is the picker's orders, numbered, the one in use checked
export function sortBox(m) {
  const P = env.P;
  const body = [[[' Sort', { fg: P.Heading, b: true }]]];
  SORTS.forEach(([name], i) => {
    const on = st => (i === m.sortSel ? Object.assign({}, st, { bg: P.CursorRow }) : st);
    body.push([[' ' + (i + 1) + ' ', on({ fg: P.Cursor, b: true })], [padR(name, 24), on(fg(P.Name))], [(i === m.sort ? '✓' : ' ') + ' ', on(fg(P.Selected))]]);
  });
  return boxed(body, P.Cursor, 0);
}
