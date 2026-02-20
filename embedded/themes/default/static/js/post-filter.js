(function() {
  var items = document.querySelectorAll('.post-item');
  var allTags = {};
  var allCategories = {};

  items.forEach(function(item) {
    var tags = item.getAttribute('data-tags');
    var cats = item.getAttribute('data-categories');
    if (tags) {
      tags.split(',').forEach(function(t) {
        t = t.trim();
        if (t) allTags[t] = true;
      });
    }
    if (cats) {
      cats.split(',').forEach(function(c) {
        c = c.trim();
        if (c) allCategories[c] = true;
      });
    }
  });

  var tagNames = Object.keys(allTags).sort();
  var catNames = Object.keys(allCategories).sort();

  if (tagNames.length === 0 && catNames.length === 0) return;

  var controls = document.getElementById('filter-controls');
  var btnContainer = document.getElementById('filter-buttons');
  controls.classList.remove('hidden');

  var activeFilter = null;
  var activeType = null;

  function makeBtn(label, type, value) {
    var btn = document.createElement('button');
    btn.textContent = label;
    btn.className = 'badge text-xs cursor-pointer transition-colors';
    btn.setAttribute('data-filter-type', type);
    btn.setAttribute('data-filter-value', value);
    if (type === 'all') {
      btn.classList.add('badge-active');
    }
    btn.addEventListener('click', function() {
      if (type === 'all') {
        activeFilter = null;
        activeType = null;
      } else {
        activeFilter = value;
        activeType = type;
      }
      applyFilter();
    });
    return btn;
  }

  btnContainer.appendChild(makeBtn('All', 'all', ''));

  if (catNames.length > 0) {
    var catLabel = document.createElement('span');
    catLabel.textContent = 'Categories:';
    catLabel.className = 'text-xs text-muted-foreground font-medium ml-2 self-center';
    btnContainer.appendChild(catLabel);
    catNames.forEach(function(c) {
      btnContainer.appendChild(makeBtn(c, 'category', c));
    });
  }

  if (tagNames.length > 0) {
    var tagLabel = document.createElement('span');
    tagLabel.textContent = 'Tags:';
    tagLabel.className = 'text-xs text-muted-foreground font-medium ml-2 self-center';
    btnContainer.appendChild(tagLabel);
    tagNames.forEach(function(t) {
      btnContainer.appendChild(makeBtn(t, 'tag', t));
    });
  }

  function applyFilter() {
    var btns = btnContainer.querySelectorAll('button');
    btns.forEach(function(b) {
      b.classList.remove('badge-active');
      var bType = b.getAttribute('data-filter-type');
      var bVal = b.getAttribute('data-filter-value');
      if (activeFilter === null && bType === 'all') {
        b.classList.add('badge-active');
      } else if (bType === activeType && bVal === activeFilter) {
        b.classList.add('badge-active');
      }
    });

    items.forEach(function(item) {
      if (activeFilter === null) {
        item.style.display = '';
        return;
      }
      var attr = activeType === 'tag' ? 'data-tags' : 'data-categories';
      var values = (item.getAttribute(attr) || '').split(',').map(function(v) { return v.trim(); });
      item.style.display = values.indexOf(activeFilter) !== -1 ? '' : 'none';
    });
  }
})();
