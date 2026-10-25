"""Stamp the scripts and styles of docs/ with a version as Pages deploys it.

gomi.dev sends JavaScript and CSS to be cached for hours, so a browser that
saw the site before a deploy would go on running its old scripts with the
new page. This gives every module and stylesheet a URL of the deploy's own:
the import map in index.html sends each module to ?v=VERSION, which reaches
the modules the page's modules import too, and the stylesheet links get the
same. Without it, as when docs/ is served locally, the page loads them as
they are.

Usage: python3 scripts/site-stamp.py DOCS VERSION
"""
import json
import pathlib
import sys

docs, version = pathlib.Path(sys.argv[1]), sys.argv[2]
v = "?v=" + version

modules = sorted(p.relative_to(docs).as_posix() for p in docs.rglob("*.js"))
importmap = {"imports": {"./" + m: "./" + m + v for m in modules}}

page = docs / "index.html"
html = page.read_text()
empty = '<script type="importmap">{"imports": {}}</script>'
if empty not in html:
    sys.exit("site-stamp: no empty import map in index.html")
html = html.replace(empty, '<script type="importmap">' + json.dumps(importmap) + "</script>")
html = html.replace('href="site.css"', 'href="site.css' + v + '"')
page.write_text(html)

notfound = docs / "404.html"
notfound.write_text(notfound.read_text().replace('href="/site.css"', 'href="/site.css' + v + '"'))

print(f"site-stamp: {len(modules)} modules and site.css at {v}")
