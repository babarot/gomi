// The TUI, W columns by H rows: the list with the preview beside it, or the
// preview over the whole of it, with the keys or the sort menu over the
// middle (view.go, view_list.go, view_detail.go).
import { env } from './env.js';
import { spaces, width, fg } from './text.js';
import { Grid } from './grid.js';
import { rows, fixCursor, scroll, currentFile } from './list.js';
import { rowLine, headerLine, filterLine, countLine, helpLine, emptyText } from './view_list.js';
import { pane, foldPane } from './view_detail.js';
import { layout } from './layout.js';
import { helpBox, sortBox } from './overlay.js';

const statusLine = m => (m.status ? [['  ' + m.status, fg(env.P.Selected)]] : []);

// view draws the TUI into a grid
export function view(m, W, H) {
  const lines = m.detail ? detailView(m, W, H) : listView(m, W, H);
  const g = new Grid(W, H);
  lines.forEach((l, y) => g.put(0, y, l));
  const box = m.overlay === 'help' ? helpBox(m) : m.overlay === 'sort' ? sortBox(m) : null;
  if (box) {
    const bw = width(box[0]);
    const x = Math.max(0, Math.floor((W - bw) / 2)), top = Math.max(0, Math.floor((lines.length - box.length) / 2));
    box.forEach((l, i) => { g.put(x, top + i, [[spaces(bw), {}]]); g.put(x, top + i, l); });
  }
  return g;
}

function listView(m, W, H) {
  const out = [headerLine(m, W)];
  if (m.filtering || m.filter !== '') out.push(filterLine(m));
  const l = layout(m, W, H), n = l.rows, rs = rows(m);
  fixCursor(m, rs);
  scroll(m, rs, n);
  const shown = [];
  for (let i = 0; i < n; i++) {
    const j = m.top + i;
    shown.push(j < rs.length ? rowLine(m, rs[j], j === m.cursor, l.listW) : []);
  }
  if (!rs.length) shown[0] = [['  ' + emptyText(m), fg(env.P.Description)]];
  if (l.side === 'right') {
    const r = rs[m.cursor];
    const p = !r ? [] : r.kind === 'fold' ? foldPane(m, r, l.paneW, l.paneH) : pane(m, r.file, l.paneW, l.paneH);
    shown.forEach((s, i) => { shown[i] = [...s, [spaces(l.listW - width(s)), {}], [' ', {}], ...(p[i] || [])]; });
  }
  out.push(...shown, countLine(m, rs, n, W), statusLine(m), helpLine(m, W));
  return out;
}

function detailView(m, W, H) {
  const f = currentFile(m);
  return [headerLine(m, W), ...(f ? pane(m, f, W, Math.max(8, H - 3)) : []), statusLine(m), helpLine(m, W)];
}
