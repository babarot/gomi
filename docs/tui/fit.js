// fit measures the font of a screen element: how many columns of it fit,
// and whether its letters differ in width. A cell is made exactly as wide
// as a character (--cw, site.css), so that lines drawn cell by cell meet
// the text around them.
export function fit(screen) {
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
  screen.style.setProperty('--cw', cw + 'px');
  const cs = getComputedStyle(screen);
  const inner = screen.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight);
  return {
    cols: Math.floor(inner / cw),
    // a font whose i and M differ in width is not monospaced
    cells: Math.abs(width('i'.repeat(50)) - width('M'.repeat(50))) > 1,
  };
}
