// The preview pane: of a file, or of a fold (view_detail.go).
import { env } from './env.js';
import { ELL, sw, truncate, spaces, padR, shortPath, width, cutSegs, ellipsize, fg } from './text.js';
import { hhmm, ymdhm, relTime, compactBytes } from './time.js';
import { previewOf } from './preview.js';
import { sizeOf } from './view_list.js';

// the rows above a file's contents: From, Deleted, Size, Restore, a rule
const INFO_ROWS = 5;

function frameRow(segs, inner) {
  const B = fg(env.P.Border);
  return [['│', B], [' ', {}], ...cutSegs(segs, inner), [spaces(inner - Math.min(inner, width(segs))), {}], [' ', {}], ['│', B]];
}
const label = k => [padR(k, 9), fg(env.P.Description)];

function restoreText(f) {
  const P = env.P;
  if (f.origin === 'taken') return [['! a file has its name there now; gomi asks for a new name', fg(P.Dialog)]];
  if (f.origin === 'gone') return [['+ its folder is gone; gomi makes it again', fg(P.Description)]];
  return [['back to where it was', fg(P.Selected)]];
}

// pane draws f in a frame w by h: its name on the frame, where and when
// it was deleted and its size, then its contents, and how far they are
// scrolled on the bottom of the frame
export function pane(m, f, w, h) {
  const P = env.P, B = fg(P.Border);
  const inner = Math.max(0, w - 4), bodyH = Math.max(0, h - 2 - INFO_ROWS);
  const title = [[truncate(f.name + (f.isDir ? '/' : ''), Math.max(0, w - 8), ELL), { fg: f.isDir ? P.DirName : P.Name, b: true }]];
  if (m.sel.has(f.id)) title.push([' ', {}], ['✓', { fg: P.Selected, b: true }]);
  const top = [['╭─ ', B], ...title, [' ', {}]];
  const lines = [[...top, ['─'.repeat(Math.max(0, w - 1 - width(top))) + '╮', B]]];
  lines.push(frameRow([label('From'), [shortPath(f.dir, Math.max(0, inner - 9)), fg(P.Name)]], inner));
  lines.push(frameRow(ellipsize([label('Deleted'), [ymdhm(f.at), fg(P.Name)], ['  ' + relTime(f.at), fg(P.Description)]], inner), inner));
  lines.push(frameRow([label('Size'), [compactBytes(f.size), fg(P.Name)], ['  ' + (f.isDir ? 'directory' : 'file'), fg(P.Description)]], inner));
  lines.push(frameRow(ellipsize([label('Restore'), ...restoreText(f)], inner), inner));
  lines.push(frameRow([['╌'.repeat(inner), fg(P.PreviewBorder)]], inner));

  // the contents are read once the cursor rests on the file
  let body, total = 0;
  if (env.previewDelay > 0 && !m.previewed.has(f.id)) body = [[['Loading…', fg(P.PreviewError)]]];
  else {
    const pv = previewOf(f, inner, bodyH);
    if (pv.error) body = [[['cannot preview (' + pv.error + ')', fg(P.PreviewError)]]];
    else {
      total = pv.length;
      m.vy = Math.max(0, Math.min(m.vy, Math.max(0, total - bodyH)));
      body = pv.slice(m.vy, m.vy + bodyH);
    }
  }
  for (let i = 0; i < bodyH; i++) lines.push(frameRow(body[i] || [], inner));
  const pos = total > bodyH ? ' ' + (m.vy + 1) + '–' + Math.min(m.vy + bodyH, total) + '/' + total + ' ' : '';
  lines.push([['╰' + '─'.repeat(Math.max(0, w - 3 - sw(pos))) + pos + '─╯', B]]);
  return lines;
}

// foldPane is the preview of a fold: what it is, how many files it holds,
// their size and when they were deleted, then the files
export function foldPane(m, r, w, h) {
  const P = env.P, B = fg(P.Border);
  const inner = Math.max(0, w - 4);
  const row = segs => frameRow(ellipsize(segs, inner), inner);
  const title = m.view === 'places' ? shortPath(r.key, Math.max(0, w - 8)) : 'rm at ' + ymdhm(r.files[0].at);
  const top = [['╭─ ', B], [title, { fg: P.Name, b: true }], [' ', {}]];
  const lines = [[...top, ['─'.repeat(Math.max(0, w - 1 - width(top))) + '╮', B]]];
  const count = r.files.length < r.total ? r.files.length + ' of ' + r.total : String(r.files.length);
  lines.push(row([label('Files'), [count, fg(P.Name)], ['  ' + sizeOf(r.files), fg(P.Description)]]));
  const newest = r.files[0].at, oldest = r.files[r.files.length - 1].at;
  let when = ymdhm(newest);
  if (ymdhm(oldest) !== when) {
    when = ymdhm(oldest) + ' – ' + hhmm(newest);
    if (oldest.toDateString() !== newest.toDateString()) when = ymdhm(oldest).slice(0, 10) + ' – ' + ymdhm(newest).slice(0, 10);
  }
  lines.push(row([label('Deleted'), [when, fg(P.Name)]]));
  lines.push(row([label('Restore'), [m.view === 'batches' ? 'enter puts this rm back' : 'enter restores them all', fg(P.Selected)]]));
  const taken = r.files.filter(f => f.origin === 'taken').length, gone = r.files.filter(f => f.origin === 'gone').length;
  const marks = [];
  if (taken) marks.push(['! ' + taken + ' have a file in their place', fg(P.Dialog)]);
  if (gone) { if (marks.length) marks.push([' · ', fg(P.Description)]); marks.push(['+ ' + gone + ' have no folder', fg(P.Description)]); }
  lines.push(row([label('Marks'), ...(marks.length ? marks : [['none', fg(P.Description)]])]));
  lines.push(row([['╌'.repeat(inner), fg(P.PreviewBorder)]]));
  const body = Math.max(0, h - 2 - INFO_ROWS);
  for (let i = 0; i < body; i++) {
    const f = r.files[i];
    if (!f) { lines.push(row([])); continue; }
    const size = compactBytes(f.size);
    const name = truncate(f.name + (f.isDir ? '/' : ''), Math.max(0, inner - 8), ELL);
    lines.push(row([[name, fg(f.isDir ? P.DirName : P.Name)], [spaces(Math.max(1, inner - sw(name) - size.length)), {}], [size, fg(P.Description)]]));
  }
  lines.push([['╰' + '─'.repeat(Math.max(0, w - 2)) + '╯', B]]);
  return lines;
}
