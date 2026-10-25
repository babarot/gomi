// What the preview pane shows of a file (file.go): text with colors, a
// directory's listing, an image in half blocks, or why it cannot be shown.
import { env } from './env.js';

const MIME = {
  pdf: 'application/pdf',
  docx: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  gz: 'application/gzip', zip: 'application/zip', bin: 'application/octet-stream',
};
const LEX = { go: 'go', py: 'py', ts: 'ts', tsx: 'ts', js: 'ts', json: 'json', yml: 'yaml', yaml: 'yaml', md: 'md', css: 'css', sql: 'sql', sh: 'sh', zsh: 'sh', toml: 'yaml', lua: 'ts' };
const ext = n => { const m = n.match(/\.([A-Za-z0-9]+)$/); return m ? m[1].toLowerCase() : ''; };

const KW = {
  go: 'package import func return if else for range var const type struct interface go defer nil true false map chan select case switch break continue',
  py: 'import from def return if elif else for in while class with as try except None True False and or not pass lambda',
  ts: 'import export from const let var function return if else for of in while class extends new async await default type interface true false null undefined',
  sh: 'export alias if then fi for do done function local return source',
  sql: 'INSERT INTO VALUES CREATE TABLE SELECT FROM WHERE PRIMARY KEY NOT NULL BEGIN COMMIT SET DROP IF EXISTS',
  css: '',
};

// highlight colors one line of lang, roughly as chroma's lexers do
function highlight(line, lang) {
  const N = env.P.syn;
  if (lang === 'md') {
    if (/^#{1,6} /.test(line)) return [[line, { fg: N.fn, b: true }]];
    if (/^\s*[-*] /.test(line)) { const m = line.match(/^(\s*[-*] )(.*)$/); return [[m[1], { fg: N.kw }], [m[2], { fg: N.text }]]; }
    return [[line, { fg: N.text }]];
  }
  if (lang === 'yaml') {
    const m = line.match(/^(\s*-?\s*)([\w.-]+)(:)(.*)$/);
    if (/^\s*#/.test(line)) return [[line, { fg: N.com, i: true }]];
    if (m) return [[m[1], { fg: N.punct }], [m[2], { fg: N.kw }], [m[3], { fg: N.punct }], [m[4], { fg: /^\s*\d+$/.test(m[4]) ? N.num : N.str }]];
    return [[line, { fg: N.str }]];
  }
  if (lang === 'json') {
    const out = []; const re = /("(?:[^"\\]|\\.)*")(\s*:)?|(-?\d+(?:\.\d+)?)|(true|false|null)|(\s+|.)/g; let m;
    while ((m = re.exec(line))) {
      if (m[1]) out.push([m[1], { fg: m[2] ? N.kw : N.str }], ...(m[2] ? [[m[2], { fg: N.punct }]] : []));
      else if (m[3]) out.push([m[3], { fg: N.num }]);
      else if (m[4]) out.push([m[4], { fg: N.kw, b: true }]);
      else out.push([m[5], { fg: N.punct }]);
    }
    return out;
  }
  if (lang === 'css') {
    const m = line.match(/^(\s*)([\w-]+)(:\s*)(.*)$/);
    if (m) return [[m[1], {}], [m[2], { fg: N.cls }], [m[3], { fg: N.punct }], [m[4], { fg: /^\d/.test(m[4]) ? N.num : N.str }]];
    if (/^\s*\/\*/.test(line)) return [[line, { fg: N.com, i: true }]];
    return [[line, { fg: N.kw }]];
  }
  const kws = new Set((KW[lang] || '').split(' '));
  const comment = lang === 'py' || lang === 'sh' ? /^#.*$/ : lang === 'sql' ? /^--.*$/ : /^\/\/.*$/;
  const out = []; let m;
  const re = /(\/\/.*$|#.*$|--.*$)|("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`[^`]*`)|(\b\d+(?:\.\d+)?\b)|([A-Za-z_][A-Za-z0-9_]*)(\s*\()?|(\s+|.)/g;
  while ((m = re.exec(line))) {
    if (m[1] && comment.test(m[1])) out.push([m[1], { fg: N.com, i: true }]);
    else if (m[1]) { out.push([m[1][0], { fg: N.punct }]); re.lastIndex = m.index + 1; }
    else if (m[2]) out.push([m[2], { fg: N.str }]);
    else if (m[3]) out.push([m[3], { fg: N.num }]);
    else if (m[4]) {
      if (kws.has(m[4])) out.push([m[4], { fg: N.kw, b: true }]);
      else if (m[5]) out.push([m[4], { fg: N.fn }]);
      else if (/^[A-Z]/.test(m[4]) && lang !== 'sql') out.push([m[4], { fg: N.cls }]);
      else out.push([m[4], { fg: N.text }]);
      if (m[5]) out.push([m[5], { fg: N.punct }]);
    } else out.push([m[6], { fg: /[{}()[\];,.]/.test(m[6]) ? N.punct : /[=+\-*/<>!&|:]/.test(m[6]) ? N.kw : N.text }]);
  }
  return out;
}

// images, drawn with half blocks as previewImage does; read once, through
// a canvas, which works only from http(s)
const IMG = {};
function loadImage(src) {
  if (IMG[src] || typeof Image === 'undefined') return;
  const img = new Image();
  IMG[src] = { img, ready: false, cache: {} };
  img.onload = () => { IMG[src].ready = true; env.changed(); };
  img.src = src;
}
function imageLines(src, w, h) {
  const rec = IMG[src];
  if (!rec || !rec.ready) return null;
  const key = w + 'x' + h;
  if (rec.cache[key]) return rec.cache[key];
  const iw = rec.img.naturalWidth, ih = rec.img.naturalHeight;
  const s = Math.min(w / iw, (h * 2) / ih);
  const cwid = Math.max(1, Math.round(iw * s)), ch = Math.max(2, Math.round(ih * s / 2) * 2);
  const cv = document.createElement('canvas'); cv.width = cwid; cv.height = ch;
  const ctx = cv.getContext('2d'); ctx.drawImage(rec.img, 0, 0, cwid, ch);
  const d = ctx.getImageData(0, 0, cwid, ch).data;
  const px = (x, y) => { const i = (y * cwid + x) * 4; return d[i + 3] < 40 ? null : 'rgb(' + d[i] + ',' + d[i + 1] + ',' + d[i + 2] + ')'; };
  const lines = [];
  for (let y = 0; y < ch; y += 2) {
    const segs = [];
    for (let x = 0; x < cwid; x++) {
      const top = px(x, y), bot = px(x, y + 1);
      segs.push(top ? ['▀', { fg: top, bg: bot || undefined }] : bot ? ['▄', { fg: bot }] : [' ', {}]);
    }
    lines.push(segs);
  }
  return (rec.cache[key] = lines);
}

// previewOf is the contents of f for a pane w columns by h rows, as lines
// of spans, or { error: mime type } when it cannot be shown. A directory
// shows f.listing as it is, else f.ls as ls -F --color draws it.
export function previewOf(f, w, h) {
  const P = env.P;
  if (f.isDir) {
    if (f.listing) return f.listing.map(l => [[l, {}]]);
    return (f.ls || []).map(e => {
      if (e.endsWith('/')) return [[e.slice(0, -1), { fg: P.lsDir, b: true }], ['/', {}]];
      if (e.endsWith('*')) return [[e.slice(0, -1), { fg: P.lsExec, b: true }], ['*', {}]];
      return [[e, {}]];
    });
  }
  if (f.img) {
    loadImage(f.img);
    return imageLines(f.img, w, h) || [[['Loading…', { fg: P.PreviewError }]]];
  }
  const e = ext(f.name);
  if (f.mime || MIME[e]) return { error: f.mime || MIME[e] };
  const body = (f.text || '').replace(/\t/g, '    ').split('\n');
  const lang = f.lang || LEX[e];
  return body.map(l => (lang ? highlight(l, lang) : [[l, { fg: P.syn.text }]]));
}
