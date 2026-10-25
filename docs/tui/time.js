// Times and sizes in the formats of the Go code, seen from env.now().
import { env } from './env.js';

const MON = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
const p2 = n => String(n).padStart(2, '0');

export const hhmm = t => p2(t.getHours()) + ':' + p2(t.getMinutes());               // 15:04
export const ymdhm = t => t.getFullYear() + '-' + p2(t.getMonth() + 1) + '-' + p2(t.getDate()) + ' ' + hhmm(t);
export const janD = t => MON[t.getMonth()] + ' ' + t.getDate();                     // Jan 2

// dayLabel names the day t falls on: Today, Yesterday, the weekday within
// a week, else the date (filelist.go)
export function dayLabel(t) {
  const n = env.now();
  const a = new Date(n.getFullYear(), n.getMonth(), n.getDate()), b = new Date(t.getFullYear(), t.getMonth(), t.getDate());
  const days = Math.round((a - b) / 86400000);
  if (days <= 0) return 'Today';
  if (days === 1) return 'Yesterday';
  if (days < 7) return DAYS[t.getDay()];
  return DAYS[t.getDay()].slice(0, 3) + ', ' + janD(t);
}

// shortAgo is how long ago t was in at most 7 columns (view_list.go)
export function shortAgo(t) {
  const d = (env.now() - t) / 1000;
  if (d < 60) return 'now';
  if (d < 3600) return Math.floor(d / 60) + 'm ago';
  if (d < 86400) return Math.floor(d / 3600) + 'h ago';
  if (d < 7 * 86400) return Math.floor(d / 86400) + 'd ago';
  return janD(t);
}

// relTime is humanize.RelTime(t, now, "ago", "from now")
export function relTime(t) {
  const d = (env.now() - t) / 1000, M = 60, H = 3600, D = 86400, W = 7 * D, MO = 30 * D, Y = 12 * MO;
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

// compactBytes writes a size in at most six columns: 794B, 3.1K, 31M
export function compactBytes(n) {
  if (n < 1024) return n + 'B';
  let v = n / 1024, i = 0; const u = 'KMGTP';
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return (v < 10 ? v.toFixed(1) : Math.round(v)) + u[i];
}

export const plural = (n, w) => (n === 1 ? w : w + 's');
