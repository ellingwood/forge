(function() {
  const btn = document.getElementById('mobile-nav-toggle');
  const menu = document.getElementById('mobile-nav');
  if (!btn || !menu) return;
  btn.addEventListener('click', function() {
    const expanded = btn.getAttribute('aria-expanded') === 'true';
    btn.setAttribute('aria-expanded', !expanded);
    menu.setAttribute('aria-hidden', expanded);
    menu.classList.toggle('hidden');
  });
})();
