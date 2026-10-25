// An emulation of gomi's TUI (gomi -b), drawn the way internal/ui draws it:
// view_list.go, view_detail.go, overlay.go and the golden files under
// internal/ui/testdata/golden are the reference for every column and color.
const GomiTerm = (() => {
  // ---------- palette ----------
  // The roles are those of internal/ui/styles/palette.go; the colors are the
  // site's, so that the terminal sits in the page: the can's purple for the
  // cursor, the desk's peach for what matches the filter.
  const PALETTE = {
    dark: {
      bg: '#120d22', fg: '#d9d0f5',
      Name: '#ece6ff', DirName: '#9db7ff', CursorRow: '#261d45', Heading: '#c9bff0',
      Cursor: '#a48cf0', Selected: '#7fdca8', FilterMatch: '#ff9e7a', FilterPrompt: '#9db7ff',
      Description: '#8a80b0', Border: '#5d4f96', PreviewBorder: '#3a2f63', PreviewError: '#8a80b0',
      Dialog: '#ff6b9a', HelpKey: '#8a80b0', HelpDesc: '#5d5384', HelpSeparator: '#3a2f63', HelpEllipsis: '#3a2f63',
      lsDir: '#9db7ff', lsExec: '#7fdca8', c240: '#6f6698', ask: '#7fdca8', err: '#ff6b9a',
      syn: { text: '#d9d0f5', kw: '#b9a3ff', fn: '#8fd3ff', cls: '#ffc6a8', str: '#9fe0b8', esc: '#ffd27a', num: '#ffb08a', com: '#6f6698', punct: '#c9bff0' },
    },
    light: {
      bg: '#fbf9ff', fg: '#2a1d4d',
      Name: '#231740', DirName: '#3557c9', CursorRow: '#ece5fb', Heading: '#4b3a86',
      Cursor: '#6a4fc4', Selected: '#2f8f5b', FilterMatch: '#d9582b', FilterPrompt: '#3557c9',
      Description: '#7a6ea3', Border: '#b7a9e6', PreviewBorder: '#ddd3f5', PreviewError: '#7a6ea3',
      Dialog: '#d6336c', HelpKey: '#7a6ea3', HelpDesc: '#a89ccc', HelpSeparator: '#d3c9f0', HelpEllipsis: '#d3c9f0',
      lsDir: '#3557c9', lsExec: '#2f8f5b', c240: '#a89ccc', ask: '#2f8f5b', err: '#d6336c',
      syn: { text: '#2a1d4d', kw: '#6a4fc4', fn: '#1f74a8', cls: '#b5532a', str: '#2f8f5b', esc: '#a86a00', num: '#c2410c', com: '#9a8fc0', punct: '#4b3a86' },
    },
  };
  let P = PALETTE.dark;

  // ---------- text width and cutting (charmbracelet/x/ansi) ----------
  const cw = ch => {
    const c = ch.codePointAt(0);
    if (c >= 0x1100 && (c <= 0x115f || (c >= 0x2e80 && c <= 0xa4cf && c !== 0x303f) || (c >= 0xac00 && c <= 0xd7a3) ||
      (c >= 0xf900 && c <= 0xfaff) || (c >= 0xfe30 && c <= 0xfe4f) || (c >= 0xff00 && c <= 0xff60) ||
      (c >= 0xffe0 && c <= 0xffe6) || (c >= 0x1f300 && c <= 0x1faff) || (c >= 0x20000 && c <= 0x3fffd))) return 2;
    return 1;
  };
  const sw = s => { let w = 0; for (const ch of s) w += cw(ch); return w; };
  const ELL = '…';
  // truncate cuts s to w columns, ending in tail when it cut anything
  function truncate(s, w, tail = '') {
    if (sw(s) <= w) return s;
    const room = w - sw(tail);
    if (room < 0) return '';
    let out = '', u = 0;
    for (const ch of s) { const c = cw(ch); if (u + c > room) break; out += ch; u += c; }
    return out + tail;
  }
  const spaces = n => ' '.repeat(Math.max(0, n));
  const padR = (s, w) => s + spaces(w - sw(s));
  const padL = (s, w) => spaces(w - sw(s)) + s;

  // shortPath keeps the head of p and as much of its tail as fits
  function shortPath(p, w) {
    if (sw(p) <= w) return p;
    const parts = p.split('/');
    if (parts.length < 3) return truncate(p, w, ELL);
    const head = parts[0] === '' ? '/' + parts[1] : parts[0];
    let tail = '', kept = 0;
    for (let i = parts.length - 1; i >= 2; i--) {
      const next = '/' + parts[i] + tail;
      if (sw(head + '/' + ELL + next) > w) break;
      tail = next; kept++;
    }
    if (kept < 2 && parts.length > 3) {
      const last = '/' + parts[parts.length - 1], prev = parts[parts.length - 2];
      const room = w - sw(head + '/' + ELL + '/' + last);
      if (room >= 6) return head + '/' + ELL + '/' + truncate(prev, room, ELL) + last;
    }
    if (tail === '') return truncate(p, w, ELL);
    return head + '/' + ELL + tail;
  }

  // ---------- spans: a line is a list of [text, style] ----------
  const width = segs => segs.reduce((a, s) => a + sw(s[0]), 0);
  function cutSegs(segs, w) {
    const out = []; let u = 0;
    for (const [t, st] of segs) {
      if (u >= w) break;
      const tw = sw(t);
      if (u + tw <= w) { out.push([t, st]); u += tw; continue; }
      out.push([truncate(t, w - u), st]); u = w;
    }
    return out;
  }

  // ---------- the screen: a grid of cells ----------
  const BOX = { '─': 'h', '│': 'v', '┌': 'dr', '└': 'ur', '├': 'vr', '╭': 'rdr', '╮': 'rdl', '╰': 'rur', '╯': 'rul', '▎': 'bar' };
  class Grid {
    constructor(w, h) { this.w = w; this.h = h; this.c = Array.from({ length: h }, () => Array.from({ length: w }, () => ({ ch: ' ', st: null }))); }
    put(x, y, segs) {
      if (y < 0 || y >= this.h) return;
      for (const [t, st] of segs) for (const ch of t) {
        const k = cw(ch);
        if (x + k > this.w) return;
        if (x >= 0) { this.c[y][x] = { ch, st }; if (k === 2) this.c[y][x + 1] = { ch: '', st }; }
        x += k;
      }
    }
    html() {
      const esc = s => s.replace(/&/g, '&amp;').replace(/</g, '&lt;');
      const css = st => {
        if (!st) return '';
        let fg = st.fg, bg = st.bg;
        if (st.r) { const f = fg || P.fg; fg = bg || P.bg; bg = f; }
        let s = '';
        if (fg) s += 'color:' + fg + ';';
        if (bg) s += 'background:' + bg + ';';
        if (st.b) s += 'font-weight:700;';
        if (st.i) s += 'font-style:italic;';
        if (st.u) s += 'text-decoration:underline;';
        return s;
      };
      return this.c.map(row => {
        let out = '', run = '', cur = null;
        const flush = () => { if (run) { out += cur ? '<span style="' + cur + '">' + esc(run) + '</span>' : esc(run); run = ''; } };
        for (const cell of row) {
          if (cell.ch === '') continue;
          const s = css(cell.st);
          const ascii = cell.ch.charCodeAt(0) < 128;
          if (!ascii) {
            // a glyph outside ASCII may come from a fallback font; give it
            // exactly one cell (two for a wide one) so the columns hold.
            // Lines, the cursor bar and half blocks are drawn with CSS, so
            // that they meet the next cell as they do in a terminal.
            flush();
            const shape = BOX[cell.ch];
            if (shape) {
              out += '<span class="c bx ' + shape + '"' + (s ? ' style="' + s + '"' : '') + '></span>';
            } else if (cell.ch === '▀' || cell.ch === '▄') {
              const st = cell.st || {}, top = cell.ch === '▀' ? st.fg : (st.bg || 'transparent'), bot = cell.ch === '▀' ? (st.bg || 'transparent') : st.fg;
              out += '<span class="c" style="background:linear-gradient(' + top + ' 50%,' + bot + ' 50%)"></span>';
            } else {
              out += '<span class="c' + (cw(cell.ch) === 2 ? ' w' : '') + '"' + (s ? ' style="' + s + '"' : '') + '>' + esc(cell.ch) + '</span>';
            }
            cur = null; continue;
          }
          if (state.cells) {
            // a font whose letters differ in width: every letter gets a cell
            flush();
            out += '<span class="c"' + (s ? ' style="' + s + '"' : '') + '>' + esc(cell.ch) + '</span>';
            cur = null; continue;
          }
          if (s !== cur) { flush(); cur = s; }
          run += cell.ch;
        }
        flush();
        return '<div class="row">' + (out || ' ') + '</div>';
      }).join('');
    }
  }

  // ---------- time (the Go code's formats) ----------
  const MON = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
  const p2 = n => String(n).padStart(2, '0');
  const hhmm = t => p2(t.getHours()) + ':' + p2(t.getMinutes());
  const ymdhm = t => t.getFullYear() + '-' + p2(t.getMonth() + 1) + '-' + p2(t.getDate()) + ' ' + hhmm(t);
  const janD = t => MON[t.getMonth()] + ' ' + t.getDate();
  const now = () => new Date();
  function dayLabel(t) {
    const n = now();
    const a = new Date(n.getFullYear(), n.getMonth(), n.getDate()), b = new Date(t.getFullYear(), t.getMonth(), t.getDate());
    const days = Math.round((a - b) / 86400000);
    if (days <= 0) return 'Today';
    if (days === 1) return 'Yesterday';
    if (days < 7) return DAYS[t.getDay()];
    return DAYS[t.getDay()].slice(0, 3) + ', ' + janD(t);
  }
  function shortAgo(t) {
    const d = (now() - t) / 1000;
    if (d < 60) return 'now';
    if (d < 3600) return Math.floor(d / 60) + 'm ago';
    if (d < 86400) return Math.floor(d / 3600) + 'h ago';
    if (d < 7 * 86400) return Math.floor(d / 86400) + 'd ago';
    return janD(t);
  }
  // humanize.RelTime
  function relTime(t) {
    const d = (now() - t) / 1000, M = 60, H = 3600, D = 86400, W = 7 * D, MO = 30 * D, Y = 12 * MO;
    const f = (n, u) => n + ' ' + u + ' ago';
    if (d < 1) return 'now';
    if (d < 2) return '1 second ago';
    if (d < M) return f(Math.floor(d), 'seconds');
    if (d < 2 * M) return '1 minute ago';
    if (d < H) return f(Math.floor(d / M), 'minutes');
    if (d < 2 * H) return '1 hour ago';
    if (d < D) return f(Math.floor(d / H), 'hours');
    if (d < 2 * D) return '1 day ago';
    if (d < W) return f(Math.floor(d / D), 'days');
    if (d < 2 * W) return '1 week ago';
    if (d < MO) return f(Math.floor(d / W), 'weeks');
    if (d < 2 * MO) return '1 month ago';
    if (d < Y) return f(Math.floor(d / MO), 'months');
    return '1 year ago';
  }
  function compactBytes(n) {
    if (n < 1024) return n + 'B';
    let v = n / 1024, i = 0; const u = 'KMGTP';
    while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
    return (v < 10 ? v.toFixed(1) : Math.round(v)) + u[i];
  }
  const plural = (n, w) => n === 1 ? w : w + 's';

  // ---------- the trash ----------
  const HOME = '/Users/you';
  const CWD = '~/work';
  const fullPath = f => (f.dir + '/' + f.name).replace(/^~/, HOME);
  let seq = 0;
  const T = { files: [], keep: [] };   // files newest first

  // ---------- previews ----------
  const LS = {
    'dist': ['assets/', 'favicon.svg', 'index.html', 'robots.txt'],
    'nvim': ['after/', 'init.lua', 'lazy-lock.json', 'lua/'],
    'figures': ['fig1-latency.png', 'fig2-throughput.png', 'fig3-errors.png', 'raw/'],
    'go-build4123': ['b001/', 'b002/', 'b003/', 'b004/', 'b005/', 'importcfg', 'importcfg.link'],
    'gomi_1.8.0_darwin_arm64': ['CHANGELOG.md', 'LICENSE', 'README.md', 'gomi*'],
    'build': ['app*', 'app.dSYM/', 'obj/', 'version.txt'],
  };
  const MIME = { pdf: 'application/pdf', docx: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', gz: 'application/gzip', zip: 'application/zip', bin: 'application/octet-stream' };
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
  function highlight(line, lang) {
    const N = P.syn;
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

  // images: drawn with half blocks, as previewImage does
  const IMG = {};
  function loadImage(src) {
    if (IMG[src]) return;
    const img = new Image();
    IMG[src] = { img, ready: false, cache: {} };
    img.onload = () => { IMG[src].ready = true; if (state.onChange) state.onChange(); };
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

  // previewOf is the contents of f in the pane, as lines of spans; null
  // while it is read
  function previewOf(f, w, h) {
    if (f.isDir) {
      return (LS[f.name] || ['(empty)']).map(e => {
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
    return body.map(l => lang ? highlight(l, lang) : [[l, { fg: P.syn.text }]]);
  }

  // ---------- the model ----------
  const VIEWS = ['recent', 'places', 'batches', 'picker'];
  const SORTS = [
    ['Deleted, newest first', (a, b) => b.at - a.at],
    ['Deleted, oldest first', (a, b) => a.at - b.at],
    ['Name', (a, b) => a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1],
    ['Size, largest first', (a, b) => b.size - a.size],
    ['Deleted from', (a, b) => a.dir < b.dir ? -1 : a.dir > b.dir ? 1 : 0],
  ];
  const state = {
    W: 120, H: 28, ROWS: 32, dark: true,
    cells: false,             // draw each letter in a cell of its own
    mode: 'shell',            // shell | tui | ask
    history: [],              // shell lines, as spans
    shellBuf: '',
    tui: null,
    ask: null,                // the new-name prompt while restoring
    onChange: null, onRestore: null, onThrow: null,
  };

  function newTui() {
    return {
      view: 'recent', sort: 0, scoped: false, open: {}, seen: {},
      filter: '', filtering: false, cursor: 0, top: 0, sel: new Set(),
      detail: false, overlay: null, sortSel: 0, vy: 0, previewed: new Set(), status: '',
    };
  }

  const here = f => f.dir === CWD || f.dir.startsWith(CWD + '/');
  function fuzzy(s, q) {
    const pos = []; let j = 0; const rs = [...s];
    for (let i = 0; i < rs.length && j < q.length; i++) if (rs[i].toLowerCase() === q[j]) { pos.push(i); j++; }
    return j === q.length ? pos : null;
  }
  function pickScore(f, q) {
    let p = fuzzy(f.name, q); if (p) return p[p.length - 1] - p[0] - p.length + 1;
    p = fuzzy(fullPath(f), q); if (p) return 1000 + p[p.length - 1] - p[0];
    return null;
  }
  const query = m => m.filter.trim().toLowerCase();
  function matches(m, f) {
    if (m.scoped && !here(f)) return false;
    const q = query(m); if (!q) return true;
    if (m.view === 'picker') return pickScore(f, q) !== null;
    return f.name.toLowerCase().includes(q) || f.dir.toLowerCase().includes(q);
  }
  const shown = m => T.files.filter(f => matches(m, f));

  function rows(m) {
    if (m.view === 'places' || m.view === 'batches') return foldRows(m);
    if (m.view === 'picker') return pickRows(m);
    return dayRows(m);
  }
  function dayRows(m) {
    const files = shown(m), out = []; let day = '', head = null;
    files.forEach((f, i) => {
      const d = dayLabel(f.at);
      if (d !== day) { head = { kind: 'day', day: d, files: [] }; out.push(head); day = d; }
      head.files.push(f);
      const first = i === 0 || files[i - 1].run !== f.run || dayLabel(files[i - 1].at) !== d;
      const last = i === files.length - 1 || files[i + 1].run !== f.run || dayLabel(files[i + 1].at) !== d;
      out.push({ kind: 'file', file: f, first, last, batch: !first || !last });
    });
    return out;
  }
  const foldOf = (m, f) => m.view === 'places' ? f.dir : f.run;
  function foldRows(m) {
    const folds = [], by = {};
    for (const f of T.files) {
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
  const keyOf = r => r.kind === 'file' ? r.file.id : r.kind === 'fold' ? 'fold:' + r.key : '';
  function fixCursor(m, rs) {
    if (!rs.length) { m.cursor = 0; return; }
    m.cursor = Math.max(0, Math.min(m.cursor, rs.length - 1));
    let i = m.cursor; while (i < rs.length && rs[i].kind === 'day') i++;
    if (i >= rs.length) { i = m.cursor; while (i > 0 && rs[i].kind === 'day') i--; }
    m.cursor = i;
  }
  function find(m, key) {
    const rs = rows(m); m.cursor = 0;
    rs.forEach((r, i) => { if (key !== '' && keyOf(r) === key && m.cursor === 0) m.cursor = i; });
    fixCursor(m, rs);
  }
  const currentRow = m => { const rs = rows(m); fixCursor(m, rs); return rs[m.cursor]; };
  const currentFile = m => { const r = currentRow(m); return r && r.kind === 'file' ? r.file : null; };

  function setView(m, view) {
    const f = currentFile(m);
    m.view = view;
    m.filtering = view === 'picker';
    if (view === 'picker') m.filter = '';
    const rs = rows(m);
    if (view !== 'recent' && view !== 'picker' && rs.length && !m.seen[view]) {
      m.open[view + ':' + rs[0].key] = true;
      if (view === 'places') rs.forEach(r => { if (r.kind === 'fold' && here({ dir: r.key })) m.open[view + ':' + r.key] = true; });
      m.seen[view] = true;
    }
    if (!f) { m.cursor = 0; fixCursor(m, rows(m)); return; }
    find(m, f.id);
    const c = currentFile(m);
    if (!c || c.id !== f.id) find(m, 'fold:' + foldOf(m, f));
  }
  function move(m, n) {
    const rs = rows(m); let i = m.cursor;
    const step = n > 0 ? 1 : -1; let left = Math.abs(n);
    while (left > 0) { let j = i + step; while (j >= 0 && j < rs.length && rs[j].kind === 'day') j += step; if (j < 0 || j >= rs.length) break; i = j; left--; }
    m.cursor = i; m.vy = 0;
  }
  function scroll(m, rs, n) {
    if (m.cursor < m.top) { m.top = m.cursor; if (m.top > 0 && rs[m.top - 1].kind === 'day') m.top--; }
    if (m.cursor >= m.top + n) m.top = m.cursor - n + 1;
    m.top = Math.max(0, Math.min(m.top, Math.max(0, rs.length - n)));
  }

  // ---------- drawing the list (view_list.go) ----------
  const fg = c => ({ fg: c });
  const onRow = cur => st => cur ? Object.assign({}, st, { bg: P.CursorRow }) : st;

  function rowStart(cur, mark) {
    const on = onRow(cur);
    return [[cur ? '▎' : ' ', on(fg(P.Cursor))], [' ', on({})], [mark || ' ', on({ fg: P.Selected, b: true })], [' ', on({})]];
  }
  function fillRow(segs, cur, w) { segs.push([spaces(w - width(segs)), onRow(cur)({})]); return cutSegs(segs, w); }

  function nameCell(m, f, w, on) {
    let name = f.name + (f.isDir ? '/' : '');
    const color = f.isDir ? P.DirName : P.Name;
    name = truncate(name, w, ELL);
    const base = on({ fg: color, b: true }), hit = on({ fg: P.FilterMatch, b: true, u: true });
    const q = query(m);
    if (m.view === 'picker') {
      const pos = new Set(q ? fuzzy(name, q) || [] : []);
      return [...name].map((c, i) => [c, pos.has(i) ? hit : base]);
    }
    if (!q) return [[name, base]];
    const i = name.toLowerCase().indexOf(q);
    if (i < 0) return [[name, base]];
    const r = [...name];
    return [[r.slice(0, i).join(''), base], [r.slice(i, i + q.length).join(''), hit], [r.slice(i + q.length).join(''), base]];
  }
  function originMark(f, on) {
    if (f.origin === 'taken') return [['!', on({ fg: P.Dialog, b: true })]];
    if (f.origin === 'gone') return [['+', on(fg(P.Description))]];
    return [[' ', on({})]];
  }
  const sizeOf = files => compactBytes(files.reduce((a, f) => a + f.size, 0));

  function dayLine(r, w) {
    const right = r.files.length + ' ' + plural(r.files.length, 'file') + ' · ' + sizeOf(r.files);
    const fill = w - 3 - sw(r.day) - 2 - sw(right) - 3;
    const rule = fg(P.HelpSeparator);
    if (fill < 1) return [['── ', rule], [r.day, { fg: P.Heading, b: true }]];
    return [['── ', rule], [r.day, { fg: P.Heading, b: true }], [' ', {}], ['─'.repeat(fill), rule], [' ', {}], [right, fg(P.Description)], [' ', {}], ['──', rule]];
  }
  function fileLine(m, r, cur, w) {
    const f = r.file, on = onRow(cur), plain = on({});
    const segs = rowStart(cur, m.sel.has(f.id) ? '✓' : '');
    const clock = r.first ? hhmm(f.at) : '';
    segs.push([clock, on(fg(P.Description))], [spaces(6 - sw(clock)), plain]);
    const join = !r.batch ? ' ' : r.first ? '┌' : r.last ? '└' : '│';
    segs.push([join, on(fg(P.HelpSeparator))], [' ', plain]);
    const avail = Math.max(0, w - 12 - 2 - 6 - 3);
    const nameW = Math.max(10, Math.floor(avail * 45 / 100)), placeW = Math.max(0, avail - nameW);
    const name = nameCell(m, f, nameW, on);
    segs.push(...name, [spaces(nameW - width(name)), plain], ['  ', plain]);
    const place = shortPath(f.dir, placeW);
    segs.push([place, on(fg(P.Description))], [spaces(placeW - sw(place)), plain]);
    const size = compactBytes(f.size);
    segs.push([spaces(6 - sw(size)), plain], [size, on(fg(P.Description))], [' ', plain], ...originMark(f, on), [' ', plain]);
    return cutSegs(segs, w);
  }
  function pickLine(m, r, cur, w) {
    const f = r.file, on = onRow(cur), plain = on({});
    const right = padL(shortAgo(f.at), 7) + '  ' + padL(compactBytes(f.size), 6);
    const segs = rowStart(cur, m.sel.has(f.id) ? '✓' : '');
    const room = Math.max(0, w - 3 - width(segs) - sw(right) - 2);
    const nameW = Math.max(10, Math.floor(room * 45 / 100));
    const name = nameCell(m, f, nameW, on);
    segs.push(...name, [spaces(Math.max(2, nameW + 2 - width(name))), plain]);
    segs.push([shortPath(f.dir, Math.max(0, room - nameW - 2)), on(fg(P.Description))]);
    segs.push([spaces(Math.max(1, w - 3 - width(segs) - sw(right))), plain], [right, on(fg(P.Description))], [' ', plain], ...originMark(f, on));
    return fillRow(segs, cur, w);
  }
  function foldLine(m, r, cur, w) {
    const on = onRow(cur), plain = on({});
    const nsel = r.files.filter(f => m.sel.has(f.id)).length;
    const mark = nsel && nsel === r.files.length ? '✓' : nsel ? '•' : '';
    let count = r.files.length + ' ' + plural(r.files.length, 'file');
    if (r.files.length < r.total) count = r.files.length + ' of ' + r.total + ' files';
    let right = count + ' · ' + sizeOf(r.files);
    if (m.view === 'places') right += '  ' + padL(shortAgo(r.files[0].at), 7);
    const room = Math.max(0, w - 6 - sw(right) - 2);
    const segs = rowStart(cur, mark);
    segs.push([r.open ? '▾' : '▸', on(fg(P.Cursor))], [' ', plain]);
    if (m.view === 'places') segs.push([shortPath(r.key, room), on({ fg: P.Name, b: true })]);
    else segs.push(...runLabel(r, room, on));
    segs.push([spaces(Math.max(1, w - 1 - width(segs) - sw(right))), plain], [right, on(fg(P.Description))]);
    return fillRow(segs, cur, w);
  }
  function runLabel(r, room, on) {
    const t = r.files[0].at;
    const when = dayLabel(t) !== 'Today' ? janD(t) + ' ' + hhmm(t) : hhmm(t);
    const places = new Set(r.files.map(f => f.dir));
    const where = places.size === 1 ? r.files[0].dir : places.size + ' places';
    const head = padR(when, 12) + ' ';
    const rest = Math.max(0, room - sw(head) - 3);
    const whereW = Math.min(sw(where), Math.floor(rest / 3));
    const nameW = Math.max(0, rest - whereW - 2);
    const names = r.files.map(f => f.name + (f.isDir ? '/' : '')).join(' ');
    const segs = [[head, on(fg(P.Description))], ['rm ', on(fg(P.HelpSeparator))], [truncate(names, nameW, ELL), on({ fg: P.Name, b: true })]];
    if (whereW > 0) segs.push(['  ', on({})], [shortPath(where, whereW), on(fg(P.Description))]);
    return segs;
  }
  function foldFileLine(m, r, cur, w) {
    const f = r.file, on = onRow(cur), plain = on({});
    const size = compactBytes(f.size);
    const right = m.view === 'batches' ? padL(size, 6) : padL(shortAgo(f.at), 7) + '  ' + padL(size, 6);
    const segs = rowStart(cur, m.sel.has(f.id) ? '✓' : '');
    segs.push(['  ', plain], [r.last ? '└' : '├', on(fg(P.HelpSeparator))], [' ', plain]);
    const room = Math.max(0, w - 3 - width(segs) - sw(right) - 2);
    if (m.view === 'batches') {
      const nameW = Math.floor(room / 2);
      const name = nameCell(m, f, nameW, on);
      segs.push(...name, [spaces(Math.max(2, nameW + 2 - width(name))), plain], [shortPath(f.dir, Math.max(0, room - nameW - 2)), on(fg(P.Description))]);
    } else segs.push(...nameCell(m, f, room, on));
    segs.push([spaces(Math.max(1, w - 3 - width(segs) - sw(right))), plain], [right, on(fg(P.Description))], [' ', plain], ...originMark(f, on));
    return fillRow(segs, cur, w);
  }
  function rowLine(m, r, cur, w) {
    if (r.kind === 'day') return dayLine(r, w);
    if (r.kind === 'fold') return foldLine(m, r, cur, w);
    if (r.depth > 0) return foldFileLine(m, r, cur, w);
    if (m.view === 'picker') return pickLine(m, r, cur, w);
    return fileLine(m, r, cur, w);
  }

  function headerLine(m, W) {
    const left = [['gomi', { fg: P.Cursor, b: true }], ['  ' + T.files.length + ' in the trash · ' + sizeOf(T.files), fg(P.Description)]];
    let right = [['view: ' + m.view, fg(P.Description)]];
    if (m.scoped) right = [['in ' + shortPath(CWD, Math.max(8, Math.floor(W / 3))), fg(P.FilterPrompt)], [' · ', fg(P.Description)], ...right];
    if (m.sel.size) right = [[m.sel.size + ' selected', fg(P.Selected)], [' · ', fg(P.Description)], ...right];
    const gap = W - 2 - width(left) - width(right);
    if (gap < 1) return cutSegs([[' ', {}], ...left], W);
    return [[' ', {}], ...left, [spaces(gap), {}], ...right];
  }
  function filterLine(m) {
    const prompt = m.view === 'picker' ? '❯ ' : '/ ';
    const segs = [[' ', {}], [prompt, { fg: P.FilterPrompt, b: true }], [m.filter, {}]];
    if (m.filtering) segs.push([' ', { r: true }]);
    return segs;
  }
  function countLine(m, rs, n, W) {
    const files = shown(m);
    let s = files.length + ' ' + plural(files.length, 'file');
    if (m.filter !== '') s = files.length + ' of ' + T.files.length + ' files';
    const below = rs.slice(Math.min(rs.length, m.top + n)).filter(r => r.kind === 'file').length;
    if (below > 0) s += ' · ↓ ' + below + ' more';
    if (files.some(f => f.origin)) s += '   ! a file has its name now   + its folder is gone';
    return [['  ' + truncate(s, W - 2, ELL), fg(P.Description)]];
  }
  // bubbles' help.ShortHelpView, at the width the UI gives it
  function helpLine(m, W) {
    const list = [['↑/k', 'up'], ['↓/j', 'down'], ['enter', 'restore'], ['space', 'detail'], ['tab', 'select'], ['/', 'filter'], ['v', 'view'], ['?', 'more']];
    const detail = [['space', 'back'], ['n', 'next'], ['p', 'prev'], ['?', 'more']];
    const binds = m.detail ? detail : list;
    const max = W - 2; let total = 0; const segs = [['  ', {}]];
    for (let i = 0; i < binds.length; i++) {
      const [k, d] = binds[i]; const sep = total > 0 ? ' • ' : '';
      const w = sw(sep + k + ' ' + d);
      if (total + w > max) { if (total + 2 < max) segs.push([' …', fg(P.HelpEllipsis)]); break; }
      if (sep) segs.push([sep, fg(P.HelpSeparator)]);
      segs.push([k, fg(P.HelpKey)], [' ' + d, fg(P.HelpDesc)]);
      total += w;
    }
    return segs;
  }

  // ---------- the preview pane (view_detail.go) ----------
  function restoreText(f) {
    if (f.origin === 'taken') return [['! a file has its name there now; gomi asks for a new name', fg(P.Dialog)]];
    if (f.origin === 'gone') return [['+ its folder is gone; gomi makes it again', fg(P.Description)]];
    return [['back to where it was', fg(P.Selected)]];
  }
  // ellipsize is ansi.Truncate(s, w, "…") on spans
  const ellipsize = (segs, w) => width(segs) > w ? cutSegs(segs, w - 1).concat([[ELL, segs[segs.length - 1][1]]]) : segs;
  function frameRow(segs, inner) { return [['│', fg(P.Border)], [' ', {}], ...cutSegs(segs, inner), [spaces(inner - Math.min(inner, width(segs))), {}], [' ', {}], ['│', fg(P.Border)]]; }
  const label = k => [padR(k, 9), fg(P.Description)];
  function pane(m, f, w, h) {
    const inner = Math.max(0, w - 4), bodyH = Math.max(0, h - 2 - 5);
    const lines = [];
    const title = [[truncate(f.name + (f.isDir ? '/' : ''), Math.max(0, w - 8), ELL), { fg: f.isDir ? P.DirName : P.Name, b: true }]];
    if (m.sel.has(f.id)) title.push([' ', {}], ['✓', { fg: P.Selected, b: true }]);
    const top = [['╭─ ', fg(P.Border)], ...title, [' ', {}]];
    lines.push([...top, ['─'.repeat(Math.max(0, w - 1 - width(top))) + '╮', fg(P.Border)]]);
    lines.push(frameRow([label('From'), [shortPath(f.dir, Math.max(0, inner - 9)), fg(P.Name)]], inner));
    lines.push(frameRow(ellipsize([label('Deleted'), [ymdhm(f.at), fg(P.Name)], ['  ' + relTime(f.at), fg(P.Description)]], inner), inner));
    lines.push(frameRow([label('Size'), [compactBytes(f.size), fg(P.Name)], ['  ' + (f.isDir ? 'directory' : 'file'), fg(P.Description)]], inner));
    lines.push(frameRow(ellipsize([label('Restore'), ...restoreText(f)], inner), inner));
    lines.push(frameRow([['╌'.repeat(inner), fg(P.PreviewBorder)]], inner));
    // the preview is read once the cursor rests on the file
    let body = null, total = 0;
    if (!m.previewed.has(f.id)) body = [[['Loading…', fg(P.PreviewError)]]];
    else {
      const pv = previewOf(f, inner, bodyH);
      if (pv.error) body = [[['cannot preview (' + pv.error + ')', fg(P.PreviewError)]]];
      else { total = pv.length; m.vy = Math.max(0, Math.min(m.vy, Math.max(0, total - bodyH))); body = pv.slice(m.vy, m.vy + bodyH); }
    }
    for (let i = 0; i < bodyH; i++) lines.push(frameRow(body[i] || [], inner));
    const pos = total > bodyH ? ' ' + (m.vy + 1) + '–' + Math.min(m.vy + bodyH, total) + '/' + total + ' ' : '';
    lines.push([['╰' + '─'.repeat(Math.max(0, w - 3 - sw(pos))) + pos + '─╯', fg(P.Border)]]);
    return lines;
  }
  function foldPane(m, r, w, h) {
    const inner = Math.max(0, w - 4);
    const title = m.view === 'places' ? shortPath(r.key, Math.max(0, w - 8)) : 'rm at ' + ymdhm(r.files[0].at);
    const top = [['╭─ ', fg(P.Border)], [title, { fg: P.Name, b: true }], [' ', {}]];
    const lines = [[...top, ['─'.repeat(Math.max(0, w - 1 - width(top))) + '╮', fg(P.Border)]]];
    const row = segs => frameRow(ellipsize(segs, inner), inner);
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
    const body = Math.max(0, h - 2 - 5);
    for (let i = 0; i < body; i++) {
      const f = r.files[i];
      if (!f) { lines.push(row([])); continue; }
      const size = compactBytes(f.size);
      const name = truncate(f.name + (f.isDir ? '/' : ''), Math.max(0, inner - 8), ELL);
      lines.push(row([[name, fg(f.isDir ? P.DirName : P.Name)], [spaces(Math.max(1, inner - sw(name) - size.length)), {}], [size, fg(P.Description)]]));
    }
    lines.push([['╰' + '─'.repeat(Math.max(0, w - 2)) + '╯', fg(P.Border)]]);
    return lines;
  }

  // ---------- layout (layout.go) ----------
  function layout(m) {
    let chrome = 1 + 2 + 1;
    if (m.filtering || m.filter !== '') chrome++;
    const avail = Math.max(1, state.H - chrome);
    const W = state.W;
    const l = { side: null, listW: W, rows: avail };
    if (W >= 100) {
      const paneW = Math.min(Math.max(Math.floor(W * 42 / 100), 40), 64);
      if (W - paneW - 1 >= 40 && avail >= 8) Object.assign(l, { side: 'right', paneW, paneH: avail, listW: W - paneW - 1 });
    }
    return l;
  }

  // ---------- overlays (overlay.go) ----------
  function helpBox(m) {
    const k = (key, d) => ({ key, d });
    const move = { name: 'Move', b: [k('↑/k', 'up'), k('↓/j', 'down'), k('g/home', 'go to start'), k('G/end', 'go to end')] };
    if (m.view === 'places' || m.view === 'batches') move.b.push(k('l/→', 'open'), k('h/←', 'close'));
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
      return [head, ...g.b.filter(b => b.key).map(b => [['  ', {}], [padR(b.key, 11), { fg: P.Cursor, b: true }], [b.d, fg(P.Name)]])];
    });
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
  function sortBox(m) {
    const body = [[[' Sort', { fg: P.Heading, b: true }]]];
    SORTS.forEach(([name], i) => {
      const on = onRow(i === m.sortSel);
      body.push([[' ' + (i + 1) + ' ', on({ fg: P.Cursor, b: true })], [padR(name, 24), on(fg(P.Name))], [(i === m.sort ? '✓' : ' ') + ' ', on(fg(P.Selected))]]);
    });
    return boxed(body, P.Cursor, 0);
  }
  function boxed(body, color, padX = 1) {
    const inner = Math.max(...body.map(width));
    const b = fg(color), pad = spaces(padX);
    return [
      [['╭' + '─'.repeat(inner + padX * 2) + '╮', b]],
      ...body.map(l => [['│', b], [pad, {}], ...l, [spaces(inner - width(l)) + pad, {}], ['│', b]]),
      [['╰' + '─'.repeat(inner + padX * 2) + '╯', b]],
    ];
  }

  // ---------- the whole TUI, H rows ----------
  function tuiLines(m) {
    const W = state.W, H = state.H;
    let out;
    if (m.detail) {
      const f = currentFile(m);
      const ph = Math.max(8, H - 3);
      out = [headerLine(m, W), ...(f ? pane(m, f, W, ph) : []), m.status ? [['  ' + m.status, fg(P.Selected)]] : [], helpLine(m, W)];
    } else {
      out = [headerLine(m, W)];
      if (m.filtering || m.filter !== '') out.push(filterLine(m));
      const l = layout(m), n = l.rows, rs = rows(m);
      fixCursor(m, rs); scroll(m, rs, n);
      const shownRows = [];
      for (let i = 0; i < n; i++) {
        const j = m.top + i;
        shownRows.push(j < rs.length ? rowLine(m, rs[j], j === m.cursor, l.listW) : []);
      }
      if (!rs.length) shownRows[0] = [['  ' + (m.filter ? 'No files match "' + m.filter + '"' : m.scoped ? 'Nothing was deleted here; . shows every file' : 'The trash is empty'), fg(P.Description)]];
      if (l.side === 'right') {
        const r = rs[m.cursor];
        const p = !r ? [] : r.kind === 'fold' ? foldPane(m, r, l.paneW, l.paneH) : pane(m, r.file, l.paneW, l.paneH);
        shownRows.forEach((s, i) => { shownRows[i] = [...s, [spaces(l.listW - width(s)), {}], [' ', {}], ...(p[i] || [])]; });
      }
      out.push(...shownRows, countLine(m, rs, n, W), m.status ? [['  ' + m.status, fg(P.Selected)]] : [], helpLine(m, W));
    }
    // overlays go over the middle
    const box = m.overlay === 'help' ? helpBox(m) : m.overlay === 'sort' ? sortBox(m) : null;
    if (box) {
      const g = new Grid(W, out.length);
      out.forEach((l, y) => g.put(0, y, l));
      const bw = width(box[0]);
      const x = Math.max(0, Math.floor((W - bw) / 2)), top = Math.max(0, Math.floor((out.length - box.length) / 2));
      box.forEach((l, i) => { g.put(x, top + i, [[spaces(bw), {}]]); g.put(x, top + i, l); });
      return { grid: g };
    }
    return { lines: out };
  }

  // ---------- the shell around it ----------
  const PS1 = () => [['~/work', fg(P.DirName)], [' ❯ ', { fg: P.Cursor, b: true }]];
  const say = (...segs) => state.history.push(segs);
  const cmdLine = c => say(...PS1(), [c, {}]);

  function openTui() {
    cmdLine('rm -b');
    if (!T.files.length) { say(['The trash is empty. Let\'s try deleting a file first', {}]); state.mode = 'shell'; return; }
    state.tui = newTui();
    // ui.scope auto: what was deleted here, when there is any
    state.tui.scoped = T.files.some(here);
    state.mode = 'tui';
    fixCursor(state.tui, rows(state.tui));
    schedulePreview();
  }
  function quitTui(msg) {
    state.tui = null; state.mode = 'shell';
    if (msg) say([msg, {}]);
  }
  // choose ends the TUI with these files and restores them one by one
  function choose(files) {
    state.tui = null;
    state.ask = { queue: files.slice(), value: '', error: '' };
    nextRestore();
  }
  function nextRestore() {
    const a = state.ask;
    while (a.queue.length) {
      const f = a.queue[0];
      if (f.origin === 'taken') { state.mode = 'ask'; a.value = ''; a.error = ''; return; }
      a.queue.shift();
      finishRestore(f, f.name);
    }
    state.ask = null; state.mode = 'shell';
  }
  function finishRestore(f, newName) {
    T.files = T.files.filter(x => x.id !== f.id);
    const restored = Object.assign({}, f, { name: newName });
    say(['Restored \'' + f.name + '\' to ' + fullPath(restored), {}]);
    if (state.onRestore) state.onRestore(restored);
  }
  function askKey(k) {
    const a = state.ask, f = a.queue[0];
    if (k === 'Escape' || k === 'ctrl+c') {
      say(['? ', fg(P.ask)], ['New name to avoid to overwrite: ', {}]);
      say(['Canceled! No new filename input.', {}]);
      a.queue.shift(); return nextRestore();
    }
    if (k === 'Enter') {
      if (!a.value) { a.error = 'min: 1 characters'; return; }
      if (/[\s/]/.test(a.value)) { a.error = 'not valid chars are included'; return; }
      say(['? ', fg(P.ask)], ['New name to avoid to overwrite: ', {}], [a.value, {}]);
      a.queue.shift();
      finishRestore(f, a.value);
      return nextRestore();
    }
    if (k === 'Backspace') a.value = a.value.slice(0, -1);
    else if (k.length === 1) a.value += k;
    a.error = '';
  }
  function shellKey(k) {
    if (k === 'Enter') {
      const c = state.shellBuf.trim() || 'rm -b';
      state.shellBuf = '';
      if (/^(rm|gomi) (-b|--restore)$/.test(c)) return openTui();
      cmdLine(c);
      if (c === 'clear') { state.history = []; return; }
      if (c === 'ls') { const names = state.desk ? state.desk() : []; if (names.length) say([names.join('  '), {}]); return; }
      const rm = c.match(/^(?:rm|gomi)(?: -[rf]+)? (.+)$/);
      if (rm) {
        const names = rm[1].split(/\s+/);
        for (const n of names) if (!(state.throwByName && state.throwByName(n, true))) say(['gomi: ' + n + ': no such file or directory', {}]);
        return;
      }
      say(['zsh: command not found: ' + c.split(' ')[0], {}]);
      return;
    }
    if (k === 'Tab' && !state.shellBuf) { state.shellBuf = 'rm -b'; return; }
    if (k === 'Backspace') state.shellBuf = state.shellBuf.slice(0, -1);
    else if (k.length === 1) state.shellBuf += k;
  }

  // the preview is read after the cursor rests for 80ms (previewDelay)
  let pvTimer = null;
  function schedulePreview() {
    clearTimeout(pvTimer);
    const m = state.tui; if (!m) return;
    const f = currentFile(m); if (!f || m.previewed.has(f.id)) return;
    pvTimer = setTimeout(() => { if (state.tui === m) { m.previewed.add(f.id); state.onChange && state.onChange(); } }, 80 + Math.random() * 60);
  }

  // ---------- keys (update.go) ----------
  function tuiKey(k) {
    const m = state.tui;
    m.status = '';
    if (m.overlay === 'help') { if (['?', 'Escape', 'q'].includes(k)) m.overlay = null; return; }
    if (m.overlay === 'sort') {
      if (k === 'Escape' || k === 's') m.overlay = null;
      else if (k === 'j' || k === 'ArrowDown') m.sortSel = Math.min(SORTS.length - 1, m.sortSel + 1);
      else if (k === 'k' || k === 'ArrowUp') m.sortSel = Math.max(0, m.sortSel - 1);
      else if (/^[1-5]$/.test(k)) { m.sort = +k - 1; m.overlay = null; m.cursor = 0; }
      else if (k === 'Enter') { m.sort = m.sortSel; m.overlay = null; m.cursor = 0; }
      return;
    }
    if (k === 'ctrl+c') return quitTui('bye!');
    if (m.filtering) {
      if (k === 'Escape') {
        if (m.view === 'picker') m.filtering = false;
        else { m.filter = ''; m.filtering = false; }
      } else if (k === 'Enter') {
        if (m.view === 'picker') { const f = currentFile(m); if (f) return choose(m.sel.size ? T.files.filter(x => m.sel.has(x.id)) : [f]); }
        m.filtering = false;
      } else if (k === 'Backspace') { m.filter = m.filter.slice(0, -1); m.cursor = 0; }
      else if (k === 'ArrowDown') move(m, 1);
      else if (k === 'ArrowUp') move(m, -1);
      else if (k === 'Tab') { toggleSel(m, true); }
      else if (k.length === 1) { m.filter += k; m.cursor = 0; fixCursor(m, rows(m)); }
      return;
    }
    if (m.detail) {
      const pv = currentFile(m);
      switch (k) {
        case 'j': case 'ArrowDown': m.vy++; return;
        case 'k': case 'ArrowUp': m.vy = Math.max(0, m.vy - 1); return;
        case 'd': m.vy += 10; return;
        case 'u': m.vy = Math.max(0, m.vy - 10); return;
        case 'n': moveFile(m, 1); return;
        case 'p': moveFile(m, -1); return;
        case ' ': case 'Escape': m.detail = false; m.vy = 0; return;
        case 'Tab': toggleSel(m, true); return;
        case 'Enter': if (pv) choose(m.sel.size ? T.files.filter(x => m.sel.has(x.id)) : [pv]); return;
        case '?': m.overlay = 'help'; return;
        case 'q': return quitTui('bye!');
      }
      return;
    }
    const rs = rows(m), r = rs[m.cursor], page = layout(m).rows;
    const folds = m.view === 'places' || m.view === 'batches';
    switch (k) {
      case 'j': case 'ArrowDown': move(m, 1); break;
      case 'k': case 'ArrowUp': move(m, -1); break;
      case 'g': case 'Home': m.cursor = 0; fixCursor(m, rs); break;
      case 'G': case 'End': m.cursor = rs.length - 1; fixCursor(m, rs); break;
      case 'l': case 'ArrowRight': case 'PageDown':
        if (folds && k !== 'PageDown') { if (r && r.kind === 'fold') m.open[m.view + ':' + r.key] = true; }
        else move(m, page);
        break;
      case 'h': case 'ArrowLeft': case 'PageUp':
        if (folds && k !== 'PageUp') {
          if (r && r.kind === 'fold') m.open[m.view + ':' + r.key] = false;
          else if (r) { const key = foldOf(m, r.file); m.open[m.view + ':' + key] = false; find(m, 'fold:' + key); }
        } else move(m, -page);
        break;
      case ' ':
        if (r && r.kind === 'fold') m.open[m.view + ':' + r.key] = !r.open;
        else if (r) { m.detail = true; m.vy = 0; }
        break;
      case 'Tab': toggleSel(m, true); break;
      case 'BackTab': toggleSel(m, false); break;
      case 'Enter': {
        if (m.sel.size) return choose(T.files.filter(f => m.sel.has(f.id)));
        if (!r) break;
        return choose(r.kind === 'fold' ? r.files : [r.file]);
      }
      case '/': m.filtering = true; break;
      case 'v': setView(m, VIEWS[(VIEWS.indexOf(m.view) + 1) % VIEWS.length]); break;
      case '.': { const key = r ? keyOf(r) : ''; m.scoped = !m.scoped; find(m, key); break; }
      case 's': if (m.view === 'picker') { m.overlay = 'sort'; m.sortSel = m.sort; } break;
      case '?': m.overlay = 'help'; break;
      case 'Escape':
        if (m.filter) { m.filter = ''; }
        else if (m.sel.size) m.sel.clear();
        break;
      case 'q': return quitTui('bye!');
      default: return;
    }
  }
  function moveFile(m, n) {
    const rs = rows(m); let i = m.cursor;
    do { i += n; } while (i >= 0 && i < rs.length && rs[i].kind !== 'file');
    if (i >= 0 && i < rs.length) { m.cursor = i; m.vy = 0; }
  }
  function toggleSel(m, on) {
    const r = currentRow(m); if (!r) return;
    const files = r.kind === 'fold' ? r.files : [r.file];
    files.forEach(f => on ? m.sel.add(f.id) : m.sel.delete(f.id));
    if (on) move(m, 1);
  }

  function key(k) {
    if (state.mode === 'tui') tuiKey(k);
    else if (state.mode === 'ask') askKey(k);
    else shellKey(k);
    if (state.mode === 'tui') schedulePreview();
    state.onChange && state.onChange();
  }

  // ---------- the terminal ----------
  function render() {
    P = state.dark ? PALETTE.dark : PALETTE.light;
    const W = state.W, R = state.ROWS;
    const lines = state.history.slice();
    let grid = null, tuiTop = 0;
    if (state.mode === 'tui') {
      const t = tuiLines(state.tui);
      const hist = lines.slice(Math.max(0, lines.length - (R - state.H)));
      const g = new Grid(W, R);
      hist.forEach((l, y) => g.put(0, y, l));
      tuiTop = hist.length;
      if (t.grid) t.grid.c.forEach((row, y) => { if (tuiTop + y < R) g.c[tuiTop + y] = row; });
      else t.lines.forEach((l, y) => g.put(0, tuiTop + y, l));
      grid = g;
    } else {
      if (state.mode === 'ask') {
        const a = state.ask, f = a.queue[0];
        lines.push([['? ', fg(P.ask)], ['New name to avoid to overwrite: ', {}], ...(a.value ? [[a.value, {}], [' ', { r: true }]] : [[f.name[0], { r: true }], [f.name.slice(1), fg(P.c240)]])]);
        if (a.error) lines.push([['✘ ', fg(P.err)], [a.error, fg(P.c240)]]);
      } else {
        const ghost = state.shellBuf ? [] : [['rm -b', fg(P.c240)]];
        lines.push([...PS1(), [state.shellBuf, {}], ...(ghost.length ? [[ghost[0][0][0], { r: true, fg: P.c240 }], [ghost[0][0].slice(1), fg(P.c240)]] : [[' ', { r: true }]])]);
      }
      const shown = lines.slice(Math.max(0, lines.length - R));
      grid = new Grid(W, R);
      shown.forEach((l, y) => grid.put(0, y, l));
    }
    return grid.html();
  }

  // ---------- the files ----------
  function add(f) {
    const file = Object.assign({ id: 'f' + (seq++), origin: '', isDir: false }, f);
    T.files.push(file);
    T.files.sort((a, b) => b.at - a.at);
    return file;
  }

  return {
    state, key, render, add, openTui, cmdLine, say,
    get files() { return T.files; },
    palette: () => PALETTE[state.dark ? 'dark' : 'light'],
    setSize(w, rowsN) { state.W = w; state.ROWS = rowsN; state.H = rowsN - 4; },
    isOpen: () => state.mode === 'tui',
    setView(v) { if (state.mode !== 'tui') openTui(); setView(state.tui, v); schedulePreview(); },
    view: () => state.tui && state.tui.view,
    quit: () => { if (state.mode === 'tui') quitTui(''); },
  };
})();
