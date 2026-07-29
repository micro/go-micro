---
layout: default
---

# Search Documentation

Type below to search page titles and content.

<input id="gm-search" type="text" placeholder="Search docs..." style="width:100%; padding:.6rem .75rem; border:1px solid #d0d7de; border-radius:6px; margin: .5rem 0 1.25rem;" />
<div id="gm-results"></div>

<script src="https://cdn.jsdelivr.net/npm/fuse.js@6.6.2"></script>
<script>
(function(){
  const pages = [
    {% assign docs = site.pages | where_exp: "p", "p.url contains '/docs/'" %}
    {% for p in docs %}
      {
        url: '{{ p.url }}',
        title: {{ p.title | default: p.url | jsonify }},
        content: {{ p.content | strip_html | replace: '\n',' ' | truncate: 400 | jsonify }}
      }{% unless forloop.last %},{% endunless %}
    {% endfor %}
  ];
  const fuse = new Fuse(pages, { keys: ['title','content'], threshold: 0.4 });
  const input = document.getElementById('gm-search');
  const out = document.getElementById('gm-results');
  input.addEventListener('input', function(){
    const q = this.value.trim();
    if(!q){ out.innerHTML=''; return; }
    const results = fuse.search(q, { limit: 12 });
    out.innerHTML = '<ul style="list-style:none; padding:0; margin:0;">' +
      results.map(r => '<li style="margin:.6rem 0;">'+
        '<a href="'+r.item.url+'" style="font-weight:600">'+r.item.title+'</a><br />'+
        '<span style="font-size:.75rem; color:#555;">'+(r.item.content.substring(0,160))+'...</span>'+
      '</li>').join('') + '</ul>';
  });
})();
</script>
