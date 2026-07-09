// Client-side docs search. Fetches /search-data.json (built by Jigyll from
// the docs collection) on first focus and matches against titles + body text.
// No external services or libraries.
(function () {
  var input = document.getElementById('docs-search-input');
  var results = document.getElementById('docs-search-results');
  if (!input || !results) return;

  var docs = null;
  var selected = -1;

  function load() {
    if (docs) return Promise.resolve(docs);
    return fetch(input.dataset.index)
      .then(function (r) { return r.json(); })
      .then(function (d) { docs = d; return d; });
  }

  function score(doc, terms) {
    var title = doc.title.toLowerCase();
    var text = doc.text.toLowerCase();
    var total = 0;
    for (var i = 0; i < terms.length; i++) {
      var t = terms[i];
      var inTitle = title.indexOf(t) !== -1;
      var inText = text.indexOf(t) !== -1;
      if (!inTitle && !inText) return 0; // every term must match somewhere
      if (inTitle) total += title.indexOf(t) === 0 || title.indexOf(' ' + t) !== -1 ? 20 : 10;
      if (inText) total += Math.min(5, text.split(t).length - 1);
    }
    return total;
  }

  function excerpt(doc, terms) {
    var text = doc.text;
    var idx = -1;
    for (var i = 0; i < terms.length && idx === -1; i++) {
      idx = text.toLowerCase().indexOf(terms[i]);
    }
    if (idx === -1) return text.slice(0, 90);
    var start = Math.max(0, idx - 30);
    return (start > 0 ? '…' : '') + text.slice(start, start + 110) + '…';
  }

  function hide() {
    results.hidden = true;
    selected = -1;
  }

  function render(matches, terms) {
    results.innerHTML = '';
    if (!matches.length) { hide(); return; }
    matches.forEach(function (doc) {
      var li = document.createElement('li');
      var a = document.createElement('a');
      a.href = doc.url;
      var title = document.createElement('span');
      title.className = 'search-result-title';
      title.textContent = doc.title;
      var snip = document.createElement('small');
      snip.textContent = excerpt(doc, terms);
      a.appendChild(title);
      a.appendChild(snip);
      li.appendChild(a);
      results.appendChild(li);
    });
    results.hidden = false;
    selected = -1;
  }

  function search() {
    var q = input.value.trim().toLowerCase();
    if (q.length < 2) { hide(); return; }
    var terms = q.split(/\s+/);
    load().then(function (d) {
      var matches = d
        .map(function (doc) { return { doc: doc, s: score(doc, terms) }; })
        .filter(function (m) { return m.s > 0; })
        .sort(function (a, b) { return b.s - a.s; })
        .slice(0, 8)
        .map(function (m) { return m.doc; });
      render(matches, terms);
    });
  }

  function move(delta) {
    var items = results.querySelectorAll('li');
    if (!items.length) return;
    if (selected >= 0) items[selected].classList.remove('selected');
    selected = (selected + delta + items.length) % items.length;
    items[selected].classList.add('selected');
    items[selected].scrollIntoView({ block: 'nearest' });
  }

  input.addEventListener('focus', load);
  input.addEventListener('input', search);
  input.addEventListener('keydown', function (e) {
    if (e.key === 'ArrowDown') { e.preventDefault(); move(1); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); move(-1); }
    else if (e.key === 'Enter' && selected >= 0) {
      e.preventDefault();
      window.location = results.querySelectorAll('li a')[selected].href;
    } else if (e.key === 'Escape') { hide(); input.blur(); }
  });
  document.addEventListener('click', function (e) {
    if (!e.target.closest('.search')) hide();
  });
})();
