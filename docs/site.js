const $ = id => document.getElementById(id);
const desk = $('desk'), can = $('can'), bubble = $('bubble'), log = $('log'), count = $('count'), screen = $('screen');
const say = t => bubble.textContent = t;
const bump = cls => { can.classList.remove('gulp', 'spit'); void can.offsetWidth; can.classList.add(cls); };
let logLines = [];
const out = t => { logLines.push(t); log.textContent = logLines.slice(-2).join('\n'); };
let deskFiles = [], runN = 0;

// ---- the terminal ----
const term = GomiTerm, st = term.state;
function measure() {
  const width = text => {
    const probe = document.createElement('span');
    probe.textContent = text;
    probe.style.visibility = 'hidden'; probe.style.position = 'absolute';
    screen.appendChild(probe);
    const w = probe.getBoundingClientRect().width;
    probe.remove();
    return w;
  };
  const cw = width('0'.repeat(100)) / 100;
  // a font whose i and M differ in width is not monospaced; the terminal
  // then puts every letter in a cell of its own
  st.cells = Math.abs(width('i'.repeat(50)) - width('M'.repeat(50))) > 1;
  // a cell is exactly as wide as a character of the font, so that lines
  // drawn cell by cell meet the text around them
  screen.style.setProperty('--cw', cw + 'px');
  const cs = getComputedStyle(screen);
  const inner = screen.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight);
  return Math.max(60, Math.min(160, Math.floor(inner / cw)));
}
function draw() {
  const P = term.palette();
  screen.style.background = P.bg; screen.style.color = P.fg;
  screen.innerHTML = term.render();
  count.textContent = term.files.length;
  const v = term.view();
  document.querySelectorAll('#viewButtons button').forEach(b => b.setAttribute('aria-pressed', b.dataset.view === v));
}
st.onChange = draw;
function resize() { term.setSize(measure(), 32); draw(); }
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
// throwOut runs rm on a desk file: typed in the terminal, or clicked
function throwOut(f, typed) {
  if (!deskFiles.includes(f)) return;
  deskFiles = deskFiles.filter(x => x !== f);
  f.el.classList.add('fly');
  const wasOpen = term.isOpen();
  if (wasOpen) term.quit();
  if (!typed) term.cmdLine('rm ' + (f.isDir ? '-r ' : '') + f.name + (f.isDir ? '/' : ''));
  term.add({ name: f.name, dir: '~/work', at: new Date(), run: 'd' + (runN++), size: f.size, isDir: !!f.isDir, text: f.text, img: f.img });
  if (wasOpen) term.openTui();
  draw();
  setTimeout(() => {
    f.el.remove();
    bump('gulp');
    out('$ rm ' + f.name);
    say(deskFiles.length ? 'Got it. ' + f.name + ' is safe with me.' : 'Desk is clear. Click me to look inside.');
  }, 330);
}
st.desk = () => deskFiles.map(f => f.name + (f.isDir ? '/' : ''));
st.throwByName = (name, typed) => {
  const f = deskFiles.find(x => x.name === name.replace(/\/$/, ''));
  if (!f) return false;
  throwOut(f, typed); return true;
};
st.onRestore = f => {
  if (f.dir !== '~/work') return;
  setTimeout(() => {
    chip({ name: f.name, size: f.size, isDir: f.isDir, text: f.text, img: f.img }, true);
    bump('spit');
    out('Restored ' + f.name);
    say('Here you go: ' + f.name + '.');
  }, 120);
};
can.addEventListener('click', () => {
  if (!term.isOpen()) term.openTui();
  draw();
  say('Have a look inside.');
  $('inside').scrollIntoView({ behavior: 'smooth', block: 'start' });
  setTimeout(() => screen.focus({ preventScroll: true }), 400);
});

function reset() {
  term.quit();
  st.history = []; st.shellBuf = ''; st.ask = null; st.mode = 'shell';
  term.files.splice(0);
  SEED.forEach(s => term.add(Object.assign({}, s, { at: new Date(Date.now() - s.m * 60000) })));
  desk.innerHTML = ''; deskFiles = []; logLines = []; log.textContent = '';
  DESK_FILES.forEach(f => chip(Object.assign({}, f)));
  term.cmdLine('git status --short');
  term.say([' M site/index.html', {}]);
  term.cmdLine('ls');
  term.say([st.desk().join('  '), {}]);
  term.openTui();
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
  navigator.clipboard.writeText(cmd.textContent).then(() => { e.target.textContent = 'Copied!'; setTimeout(() => e.target.textContent = 'Copy', 2000); });
});

// ---- site theme: follows the OS until the visitor picks one ----
const root = document.documentElement, themeBtn = $('theme');
const dark = () => root.dataset.theme ? root.dataset.theme === 'dark' : matchMedia('(prefers-color-scheme: dark)').matches;
// the terminal follows the page's theme
const label = () => { const d = dark(); themeBtn.textContent = d ? 'light' : 'dark'; root.dataset.theme = d ? 'dark' : 'light'; st.dark = d; draw(); };
themeBtn.addEventListener('click', () => {
  root.dataset.theme = dark() ? 'light' : 'dark';
  try { localStorage.setItem('gomi-theme', root.dataset.theme); } catch (e) {}
  label();
});
label();


reset();
resize();
