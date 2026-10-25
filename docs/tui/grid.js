// The screen: a grid of cells, drawn into HTML or plain text.
import { cw } from './text.js';

// glyphs drawn with CSS (site.css), so that they meet the next cell
const BOX = { '─': 'h', '│': 'v', '┌': 'dr', '└': 'ur', '├': 'vr', '╭': 'rdr', '╮': 'rdl', '╰': 'rur', '╯': 'rul', '▎': 'bar' };

export class Grid {
  constructor(w, h) {
    this.w = w; this.h = h;
    this.c = Array.from({ length: h }, () => Array.from({ length: w }, () => ({ ch: ' ', st: null })));
  }

  // put draws a line of spans from column x of row y, cut at the edge
  put(x, y, segs) {
    if (y < 0 || y >= this.h) return;
    for (const [t, st] of segs) for (const ch of t) {
      const k = cw(ch);
      if (x + k > this.w) return;
      if (x >= 0) { this.c[y][x] = { ch, st }; if (k === 2) this.c[y][x + 1] = { ch: '', st }; }
      x += k;
    }
  }

  // text is the screen as the golden files have it: what shows, without
  // trailing blanks and blank rows at the end
  text() {
    const rows = this.c.map(row => row.map(c => c.ch).join('').replace(/\s+$/, ''));
    while (rows.length && rows[rows.length - 1] === '') rows.pop();
    return rows.join('\n');
  }

  // html draws the cells with P's colors. cells puts every letter in a
  // cell of its own, for a font whose letters differ in width.
  html(P, cells) {
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
    const attr = s => (s ? ' style="' + s + '"' : '');
    return this.c.map(row => {
      let out = '', run = '', cur = null;
      const flush = () => { if (run) { out += cur ? '<span style="' + cur + '">' + esc(run) + '</span>' : esc(run); run = ''; } };
      for (const cell of row) {
        if (cell.ch === '') continue;
        const s = css(cell.st);
        if (cell.ch.charCodeAt(0) >= 128) {
          // a glyph outside ASCII may come from a fallback font; it gets
          // exactly one cell (two for a wide one) so that the columns hold
          flush();
          const shape = BOX[cell.ch];
          if (shape) {
            out += '<span class="c bx ' + shape + '"' + attr(s) + '></span>';
          } else if (cell.ch === '▀' || cell.ch === '▄') {
            const st = cell.st || {};
            const top = cell.ch === '▀' ? st.fg : (st.bg || 'transparent');
            const bot = cell.ch === '▀' ? (st.bg || 'transparent') : st.fg;
            out += '<span class="c" style="background:linear-gradient(' + top + ' 50%,' + bot + ' 50%)"></span>';
          } else {
            out += '<span class="c' + (cw(cell.ch) === 2 ? ' w' : '') + '"' + attr(s) + '>' + esc(cell.ch) + '</span>';
          }
          cur = null; continue;
        }
        if (cells) {
          flush();
          out += '<span class="c"' + attr(s) + '>' + esc(cell.ch) + '</span>';
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
