// The page: the desk and the can in the hero, the terminal below them, the
// install menu and the theme.
import { createTerminal } from './tui/terminal.js';
import { fit } from './tui/fit.js';
import { SEED, DESK_FILES } from './trash.js';

const $ = id => document.getElementById(id);
const desk = $('desk'), can = $('can'), bubble = $('bubble'), log = $('log'), count = $('count'), screen = $('screen');
const say = t => { bubble.textContent = t; };
const bump = cls => { can.classList.remove('gulp', 'spit'); void can.offsetWidth; can.classList.add(cls); };
let logLines = [];
const out = t => { logLines.push(t); log.textContent = logLines.slice(-2).join('\n'); };
let deskFiles = [], runN = 0;

// ---- the terminal ----
const term = createTerminal({
  shell: {
    ls: () => deskFiles.map(f => f.name + (f.isDir ? '/' : '')),
    // rm typed at the prompt throws out a file on the desk
    rm(name) {
      const f = deskFiles.find(x => x.name === name.replace(/\/$/, ''));
      if (f) throwOut(f, true);
      return !!f;
    },
  },
});

function draw() {
  const P = term.palette();
  screen.style.background = P.bg; screen.style.color = P.fg;
  screen.innerHTML = term.html();
  count.textContent = term.files.length;
  document.querySelectorAll('#viewButtons button').forEach(b => b.setAttribute('aria-pressed', b.dataset.view === term.view));
}
term.on('change', draw);

// the terminal is as wide as the screen allows, as gomi takes the terminal's width
function resize() {
  const f = fit(screen);
  term.setCells(f.cells);
  term.size(Math.max(40, Math.min(160, f.cols)), 32);
  draw();
}
new ResizeObserver(resize).observe(screen);
document.fonts.ready.then(resize);

const KEYS = ['Enter', 'Escape', 'Backspace', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Tab', 'PageUp', 'PageDown', 'Home', 'End'];
screen.addEventListener('keydown', e => {
  if (e.metaKey || e.altKey) return;
  let k = e.key;
  if (e.ctrlKey) { if (k === 'c') k = 'ctrl+c'; else return; }
  else if (k === 'Tab' && e.shiftKey) k = 'BackTab';
  else if (!(k.length === 1 || KEYS.includes(k))) return;
  e.preventDefault();
  term.key(k);
});
document.querySelectorAll('#tkeys button').forEach(b => b.addEventListener('click', () => { term.key(b.dataset.k); screen.focus({ preventScroll: true }); }));
document.querySelectorAll('#viewButtons button').forEach(b => b.addEventListener('click', () => { term.setView(b.dataset.view); draw(); }));

// ---- the desk ----
function chip(f, pop) {
  const b = document.createElement('button');
  b.className = 'file px' + (pop ? ' pop' : '');
  b.type = 'button';
  b.textContent = f.name + (f.isDir ? '/' : '');
  b.setAttribute('aria-label', 'Throw out ' + f.name);
  b.addEventListener('click', () => throwOut(f, false));
  f.el = b;
  deskFiles.push(f);
  desk.appendChild(b);
}

// throwOut runs rm on a desk file, typed at the prompt or clicked. An open
// TUI is left first, and opened again with the file in it.
function throwOut(f, typed) {
  if (!deskFiles.includes(f)) return;
  deskFiles = deskFiles.filter(x => x !== f);
  f.el.classList.add('fly');
  const wasOpen = term.isOpen;
  if (wasOpen) term.quit();
  if (!typed) term.echo('rm ' + (f.isDir ? '-r ' : '') + f.name + (f.isDir ? '/' : ''));
  const { el, ...file } = f;
  term.add(Object.assign(file, { dir: '~/work', at: new Date(), run: 'd' + runN++, isDir: !!f.isDir }));
  if (wasOpen) term.run();
  draw();
  setTimeout(() => {
    f.el.remove();
    bump('gulp');
    out('$ rm ' + f.name);
    say(deskFiles.length ? 'Got it. ' + f.name + ' is safe with me.' : 'Desk is clear. Click me to get things back.');
  }, 330);
}

// what is restored to ~/work lands back on the desk
term.on('restore', f => {
  if (f.dir !== '~/work') { say('Sent ' + f.name + ' back to ' + f.dir + '.'); return; }
  setTimeout(() => {
    chip({ name: f.name, size: f.size, isDir: f.isDir, text: f.text, img: f.img, ls: f.ls }, true);
    bump('spit');
    out('Restored ' + f.name);
    say('Here you go: ' + f.name + '.');
  }, 120);
});

// the can gives back what is on top of it, the way rm -b and enter do
can.addEventListener('click', () => {
  if (!term.files.length) { say("I'm empty. Nothing to give back."); return; }
  const mode = term.restoreNewest();
  draw();
  if (mode === 'ask') {
    say('Its name is taken there. Give it a new one below.');
    $('inside').scrollIntoView({ behavior: 'smooth', block: 'start' });
    setTimeout(() => screen.focus({ preventScroll: true }), 400);
  }
});

function reset() {
  term.reset();
  SEED.forEach(s => term.add(Object.assign({}, s, { at: new Date(Date.now() - s.m * 60000) })));
  desk.innerHTML = ''; deskFiles = []; logLines = []; log.textContent = '';
  DESK_FILES.forEach(f => chip(Object.assign({}, f)));
  term.echo('git status --short');
  term.print(' M site/index.html');
  term.echo('ls');
  term.print(deskFiles.map(f => f.name + (f.isDir ? '/' : '')).join('  '));
  term.run();
  say('Click a file to throw it out.');
  draw();
}
$('reset').addEventListener('click', () => { reset(); screen.focus({ preventScroll: true }); });

// ---- install menu ----
const cmd = $('cmd'), note = $('note');
document.querySelectorAll('.menu li button').forEach(b => b.addEventListener('click', () => {
  document.querySelectorAll('.menu li button').forEach(x => x.setAttribute('aria-selected', x === b));
  cmd.textContent = b.dataset.cmd; note.textContent = b.dataset.note;
}));
$('copy').addEventListener('click', e => {
  navigator.clipboard.writeText(cmd.textContent).then(() => { e.target.textContent = 'Copied!'; setTimeout(() => { e.target.textContent = 'Copy'; }, 2000); });
});

// ---- the theme: follows the OS until the visitor picks one ----
const root = document.documentElement, themeBtn = $('theme');
const dark = () => (root.dataset.theme ? root.dataset.theme === 'dark' : matchMedia('(prefers-color-scheme: dark)').matches);
// the terminal follows the page's theme
const label = () => { const d = dark(); themeBtn.textContent = d ? 'light' : 'dark'; root.dataset.theme = d ? 'dark' : 'light'; term.setDark(d); draw(); };
themeBtn.addEventListener('click', () => {
  root.dataset.theme = dark() ? 'light' : 'dark';
  try { localStorage.setItem('gomi-theme', root.dataset.theme); } catch (e) {}
  label();
});
label();

reset();
resize();
