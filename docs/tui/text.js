// Text in columns, as charmbracelet/x/ansi measures and cuts it, and spans:
// a line is a list of [text, style].

export const ELL = '…';

// cw is how many columns ch takes: two for wide characters
export function cw(ch) {
  const c = ch.codePointAt(0);
  if (c >= 0x1100 && (c <= 0x115f || (c >= 0x2e80 && c <= 0xa4cf && c !== 0x303f) || (c >= 0xac00 && c <= 0xd7a3) ||
    (c >= 0xf900 && c <= 0xfaff) || (c >= 0xfe30 && c <= 0xfe4f) || (c >= 0xff00 && c <= 0xff60) ||
    (c >= 0xffe0 && c <= 0xffe6) || (c >= 0x1f300 && c <= 0x1faff) || (c >= 0x20000 && c <= 0x3fffd))) return 2;
  return 1;
}

export function sw(s) { let w = 0; for (const ch of s) w += cw(ch); return w; }

// truncate cuts s to w columns, ending in tail when it cut anything
export function truncate(s, w, tail = '') {
  if (sw(s) <= w) return s;
  const room = w - sw(tail);
  if (room < 0) return '';
  let out = '', u = 0;
  for (const ch of s) { const c = cw(ch); if (u + c > room) break; out += ch; u += c; }
  return out + tail;
}

export const spaces = n => ' '.repeat(Math.max(0, n));
export const padR = (s, w) => s + spaces(w - sw(s));
export const padL = (s, w) => spaces(w - sw(s)) + s;

// shortPath keeps the head of p and as much of its tail as fits: ~/…/drop/test.
// When only the last part fits whole, the one before it is cut instead.
export function shortPath(p, w) {
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

// width is how many columns a line of spans takes
export const width = segs => segs.reduce((a, s) => a + sw(s[0]), 0);

// cutSegs cuts a line of spans to w columns
export function cutSegs(segs, w) {
  const out = []; let u = 0;
  for (const [t, st] of segs) {
    if (u >= w) break;
    const tw = sw(t);
    if (u + tw <= w) { out.push([t, st]); u += tw; continue; }
    out.push([truncate(t, w - u), st]); u = w;
  }
  return out;
}

// ellipsize is ansi.Truncate(s, w, "…") on spans
export const ellipsize = (segs, w) =>
  width(segs) > w ? cutSegs(segs, w - 1).concat([[ELL, segs[segs.length - 1][1]]]) : segs;

export const fg = c => ({ fg: c });
